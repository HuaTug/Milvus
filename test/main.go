package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"image-search-go/config"
	"image-search-go/models"
	"image-search-go/services"
	"image-search-go/utils"

	"github.com/google/uuid"
)

type BatchInserter struct {
	milvusService    *services.MilvusService
	featureExtractor models.FeatureExtractor
	config           *config.Config
}

func NewBatchInserter(cfg *config.Config) (*BatchInserter, error) {
	// 初始化特征提取器
	featureExtractor := models.NewSimpleFeatureExtractor()

	// 初始化Milvus服务
	milvusService, err := services.NewMilvusService(&cfg.Milvus)
	if err != nil {
		return nil, fmt.Errorf("初始化Milvus服务失败: %v", err)
	}

	return &BatchInserter{
		milvusService:    milvusService,
		featureExtractor: featureExtractor,
		config:           cfg,
	}, nil
}

func (bi *BatchInserter) ProcessDataset(datasetPath string, batchSize int, maxWorkers int) error {
	log.Printf("开始处理数据集: %s", datasetPath)

	// 获取所有图像文件
	imageFiles, err := bi.findImageFiles(datasetPath)
	if err != nil {
		return fmt.Errorf("查找图像文件失败: %v", err)
	}

	log.Printf("找到 %d 个图像文件", len(imageFiles))

	// 创建工作池
	jobs := make(chan []string, 100)
	results := make(chan BatchResult, 100)
	var wg sync.WaitGroup

	// 启动工作协程
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go bi.worker(jobs, results, &wg)
	}

	// 启动结果收集协程
	go bi.resultCollector(results, len(imageFiles))

	// 分批发送任务
	totalBatches := (len(imageFiles) + batchSize - 1) / batchSize
	for i := 0; i < len(imageFiles); i += batchSize {
		end := i + batchSize
		if end > len(imageFiles) {
			end = len(imageFiles)
		}

		batch := imageFiles[i:end]
		jobs <- batch

		log.Printf("发送批次 %d/%d，包含 %d 个文件",
			(i/batchSize)+1, totalBatches, len(batch))
	}

	close(jobs)
	wg.Wait()
	close(results)

	log.Printf("数据集处理完成")
	return nil
}

type BatchResult struct {
	Success        bool
	ProcessedCount int
	ErrorCount     int
	BatchID        string
	Error          error
}

func (bi *BatchInserter) worker(jobs <-chan []string, results chan<- BatchResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for batch := range jobs {
		result := bi.processBatch(batch)
		results <- result
	}
}

func (bi *BatchInserter) processBatch(imagePaths []string) BatchResult {
	batchID := uuid.New().String()[:8]
	log.Printf("[批次 %s] 开始处理 %d 个图像", batchID, len(imagePaths))

	var imageIDs []string
	var vectors [][]float32
	var successCount, errorCount int

	for _, imagePath := range imagePaths {
		// 加载图像
		img, err := utils.LoadImageFromFile(imagePath)
		if err != nil {
			log.Printf("[批次 %s] 加载图像失败 %s: %v", batchID, imagePath, err)
			errorCount++
			continue
		}

		// 提取特征
		features, err := bi.featureExtractor.ExtractFeatures(img)
		if err != nil {
			log.Printf("[批次 %s] 提取特征失败 %s: %v", batchID, imagePath, err)
			errorCount++
			continue
		}

		// 生成图像ID并复制文件
		imageID := uuid.New().String()
		destPath := filepath.Join(bi.config.Server.UploadPath, imageID+filepath.Ext(imagePath))

		if err := bi.copyImageFile(imagePath, destPath); err != nil {
			log.Printf("[批次 %s] 复制文件失败 %s: %v", batchID, imagePath, err)
			errorCount++
			continue
		}

		imageIDs = append(imageIDs, imageID)
		vectors = append(vectors, features)
		successCount++
	}

	// 批量插入到Milvus
	if len(imageIDs) > 0 {
		if err := bi.milvusService.InsertVectors(imageIDs, vectors); err != nil {
			log.Printf("[批次 %s] 插入Milvus失败: %v", batchID, err)
			return BatchResult{
				Success:        false,
				ProcessedCount: 0,
				ErrorCount:     len(imagePaths),
				BatchID:        batchID,
				Error:          err,
			}
		}
	}

	log.Printf("[批次 %s] 完成: 成功 %d, 失败 %d", batchID, successCount, errorCount)
	return BatchResult{
		Success:        true,
		ProcessedCount: successCount,
		ErrorCount:     errorCount,
		BatchID:        batchID,
		Error:          nil,
	}
}

func (bi *BatchInserter) copyImageFile(srcPath, destPath string) error {
	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	// 加载并保存图像（这样可以统一格式）
	img, err := utils.LoadImageFromFile(srcPath)
	if err != nil {
		return err
	}

	return utils.SaveImage(img, destPath)
}

func (bi *BatchInserter) findImageFiles(rootPath string) ([]string, error) {
	var imageFiles []string

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && utils.IsValidImageFormat(info.Name()) {
			imageFiles = append(imageFiles, path)
		}

		return nil
	})

	return imageFiles, err
}

func (bi *BatchInserter) resultCollector(results <-chan BatchResult, totalFiles int) {
	var totalProcessed, totalErrors int
	var batchCount int

	for result := range results {
		batchCount++
		totalProcessed += result.ProcessedCount
		totalErrors += result.ErrorCount

		if result.Error != nil {
			log.Printf("批次处理失败: %v", result.Error)
		}

		// 定期输出进度
		if batchCount%10 == 0 {
			log.Printf("进度报告: 已处理 %d 个文件，成功 %d，失败 %d",
				totalProcessed+totalErrors, totalProcessed, totalErrors)
		}
	}

	log.Printf("最终统计: 总文件 %d，成功 %d，失败 %d",
		totalFiles, totalProcessed, totalErrors)
}

func main() {
	var (
		datasetPath = flag.String("dataset", "", "数据集路径")
		batchSize   = flag.Int("batch", 50, "批处理大小")
		workers     = flag.Int("workers", 4, "并发工作协程数")
	)
	flag.Parse()

	if *datasetPath == "" {
		log.Fatal("请指定数据集路径: -dataset /path/to/dataset")
	}

	// 检查数据集路径是否存在
	if _, err := os.Stat(*datasetPath); os.IsNotExist(err) {
		log.Fatalf("数据集路径不存在: %s", *datasetPath)
	}

	// 加载配置
	cfg := config.LoadConfig()
	log.Printf("配置加载完成")

	// 创建批量插入器
	inserter, err := NewBatchInserter(cfg)
	if err != nil {
		log.Fatalf("创建批量插入器失败: %v", err)
	}

	// 开始处理
	startTime := time.Now()
	if err := inserter.ProcessDataset(*datasetPath, *batchSize, *workers); err != nil {
		log.Fatalf("处理数据集失败: %v", err)
	}

	duration := time.Since(startTime)
	log.Printf("数据集插入完成，耗时: %v", duration)
}
