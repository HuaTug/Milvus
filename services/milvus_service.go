package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"image-search-go/config"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

// MilvusService Milvus向量数据库服务
type MilvusService struct {
	client     client.Client
	config     *config.MilvusConfig
	collection string
}

// SearchResult 搜索结果结构
type SearchResult struct {
	ID       int64   `json:"id"`
	Score    float32 `json:"score"`
	ImageID  string  `json:"image_id"`
	Distance float32 `json:"distance"`
}

// NewMilvusService 创建Milvus服务实例
func NewMilvusService(cfg *config.MilvusConfig) (*MilvusService, error) {
	// 连接到Milvus
	milvusClient, err := client.NewClient(context.Background(), client.Config{
		Address: fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
	})
	if err != nil {
		return nil, fmt.Errorf("连接Milvus失败: %v", err)
	}

	service := &MilvusService{
		client:     milvusClient,
		config:     cfg,
		collection: cfg.CollectionName,
	}

	// 初始化collection
	if err := service.initCollection(); err != nil {
		return nil, fmt.Errorf("初始化collection失败: %v", err)
	}

	log.Printf("成功连接到Milvus，collection: %s", cfg.CollectionName)
	return service, nil
}

// initCollection 初始化collection
func (s *MilvusService) initCollection() error {
	ctx := context.Background()

	// 检查collection是否存在
	hasCollection, err := s.client.HasCollection(ctx, s.collection)
	if err != nil {
		return fmt.Errorf("检查collection失败: %v", err)
	}

	if hasCollection {
		log.Printf("Collection %s 已存在", s.collection)
		return s.loadCollection()
	}

	// 创建collection
	log.Printf("创建新的collection: %s", s.collection)

	// 定义字段
	schema := &entity.Schema{
		CollectionName: s.collection,
		Description:    "图像特征向量存储",
		Fields: []*entity.Field{
			{
				Name:       "id",
				DataType:   entity.FieldTypeInt64,
				PrimaryKey: true,
				AutoID:     true,
			},
			{
				Name:     "image_id",
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "255",
				},
			},
			{
				Name:     "vector",
				DataType: entity.FieldTypeFloatVector,
				TypeParams: map[string]string{
					"dim": fmt.Sprintf("%d", s.config.Dimension),
				},
			},
			{
				Name:     "timestamp",
				DataType: entity.FieldTypeInt64,
			},
		},
	}

	// 创建collection
	err = s.client.CreateCollection(ctx, schema, entity.DefaultShardNumber)
	if err != nil {
		return fmt.Errorf("创建collection失败: %v", err)
	}

	// 创建索引
	return s.createIndex()
}

// createIndex 创建向量索引
func (s *MilvusService) createIndex() error {
	ctx := context.Background()

	// 定义索引参数
	indexParams := map[string]string{
		"index_type":  s.config.IndexType,
		"metric_type": s.config.MetricType,
		"params":      `{"nlist": 128}`, // IVF_FLAT参数
	}

	// 创建索引
	idx := entity.NewGenericIndex("vector_index", entity.IndexType(s.config.IndexType), indexParams)
	err := s.client.CreateIndex(ctx, s.collection, "vector", idx, false)
	if err != nil {
		return fmt.Errorf("创建索引失败: %v", err)
	}

	log.Printf("索引创建成功")
	return s.loadCollection()
}

// loadCollection 加载collection到内存
func (s *MilvusService) loadCollection() error {
	ctx := context.Background()

	err := s.client.LoadCollection(ctx, s.collection, false)
	if err != nil {
		return fmt.Errorf("加载collection失败: %v", err)
	}

	log.Printf("Collection %s 加载成功", s.collection)
	return nil
}

// InsertVectors 插入向量数据
func (s *MilvusService) InsertVectors(imageIDs []string, vectors [][]float32) error {
	if len(imageIDs) != len(vectors) {
		return fmt.Errorf("图片ID数量与向量数量不匹配")
	}

	ctx := context.Background()

	// 准备数据
	imageIDColumn := entity.NewColumnVarChar("image_id", imageIDs)

	// 转换向量数据
	vectorData := make([][]float32, len(vectors))
	for i, vec := range vectors {
		vectorData[i] = vec
	}
	vectorColumn := entity.NewColumnFloatVector("vector", s.config.Dimension, vectorData)

	// 时间戳
	timestamps := make([]int64, len(imageIDs))
	now := time.Now().Unix()
	for i := range timestamps {
		timestamps[i] = now
	}
	timestampColumn := entity.NewColumnInt64("timestamp", timestamps)

	// 执行插入
	_, err := s.client.Insert(ctx, s.collection, "", imageIDColumn, vectorColumn, timestampColumn)
	if err != nil {
		return fmt.Errorf("插入向量失败: %v", err)
	}

	// 刷新数据
	err = s.client.Flush(ctx, s.collection, false)
	if err != nil {
		return fmt.Errorf("刷新数据失败: %v", err)
	}

	log.Printf("成功插入 %d 个向量", len(imageIDs))
	return nil
}

// SearchSimilar 搜索相似向量
func (s *MilvusService) SearchSimilar(queryVector []float32, topK int) ([]*SearchResult, error) {
	ctx := context.Background()

	// 创建搜索参数
	sp, _ := entity.NewIndexIvfFlatSearchParam(16)

	// 执行搜索
	result, err := s.client.Search(
		ctx,
		s.collection,
		[]string{},           // 分区名称
		"",                   // 表达式
		[]string{"image_id"}, // 输出字段
		[]entity.Vector{entity.FloatVector(queryVector)}, // 查询向量
		"vector",                               // 向量字段名
		entity.MetricType(s.config.MetricType), // 距离度量
		topK,                                   // 返回数量
		sp,
	)
	if err != nil {
		return nil, fmt.Errorf("搜索失败: %v", err)
	}

	// 处理搜索结果
	var searchResults []*SearchResult
	for _, res := range result {
		for i := 0; i < res.ResultCount; i++ {
			imageID, _ := res.Fields.GetColumn("image_id").Get(i)

			searchResult := &SearchResult{
				ID:       res.IDs.(*entity.ColumnInt64).Data()[i],
				Score:    res.Scores[i],
				ImageID:  imageID.(string),
				Distance: res.Scores[i],
			}
			searchResults = append(searchResults, searchResult)
		}
	}

	return searchResults, nil
}

// DeleteVector 删除向量
func (s *MilvusService) DeleteVector(imageID string) error {
	ctx := context.Background()

	// 构建删除表达式
	expr := fmt.Sprintf("image_id == \"%s\"", imageID)

	// 执行删除
	err := s.client.Delete(ctx, s.collection, "", expr)
	if err != nil {
		return fmt.Errorf("删除向量失败: %v", err)
	}

	log.Printf("成功删除图片 %s 的向量", imageID)
	return nil
}

// GetCollectionStats 获取collection统计信息
func (s *MilvusService) GetCollectionStats() (map[string]interface{}, error) {
	ctx := context.Background()

	stats, err := s.client.GetCollectionStatistics(ctx, s.collection)
	if err != nil {
		return nil, fmt.Errorf("获取统计信息失败: %v", err)
	}

	// 解析统计信息
	return map[string]interface{}{
		"collection_stats": stats,
		"collection_name":  s.collection,
	}, nil
}

// Close 关闭连接
func (s *MilvusService) Close() {
	if s.client != nil {
		s.client.Close()
		log.Println("Milvus连接已关闭")
	}
}

// HealthCheck 健康检查
func (s *MilvusService) HealthCheck() error {
	ctx := context.Background()

	// 检查服务器状态
	state, err := s.client.GetLoadingProgress(ctx, s.collection, []string{})
	if err != nil {
		return fmt.Errorf("健康检查失败: %v", err)
	}

	log.Printf("Collection %s 加载进度: %d%%", s.collection, state)
	return nil
}
