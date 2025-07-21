package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/google/uuid"

	"image-search-go/config"
	"image-search-go/models"
	"image-search-go/services"
	"image-search-go/utils"

	"github.com/gin-gonic/gin"
)

// ImageHandler 图像处理器
type ImageHandler struct {
	milvusService    *services.MilvusService
	featureExtractor models.FeatureExtractor
	config           *config.Config
}

// NewImageHandler 创建图像处理器
func NewImageHandler(milvusService *services.MilvusService, featureExtractor models.FeatureExtractor, cfg *config.Config) *ImageHandler {
	return &ImageHandler{
		milvusService:    milvusService,
		featureExtractor: featureExtractor,
		config:           cfg,
	}
}

// UploadImageRequest 上传图像请求
type UploadImageRequest struct {
	Description string `form:"description"`
}

// UploadImageResponse 上传图像响应
type UploadImageResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	ImageID   string `json:"image_id,omitempty"`
	ImagePath string `json:"image_path,omitempty"`
}

// SearchImageResponse 搜索图像响应
type SearchImageResponse struct {
	Success bool                      `json:"success"`
	Message string                    `json:"message"`
	Results []SearchResultWithDetails `json:"results,omitempty"`
	Total   int                       `json:"total"`
}

// SearchResultWithDetails 带详细信息的搜索结果
type SearchResultWithDetails struct {
	ImageID    string  `json:"image_id"`
	Score      float32 `json:"score"`
	Distance   float32 `json:"distance"`
	ImagePath  string  `json:"image_path"`
	Similarity string  `json:"similarity"`
}

// StatsResponse 统计信息响应
type StatsResponse struct {
	Success        bool                   `json:"success"`
	Message        string                 `json:"message"`
	CollectionInfo map[string]interface{} `json:"collection_info,omitempty"`
	ServerInfo     map[string]interface{} `json:"server_info,omitempty"`
}

// UploadImage 上传图像API
func (h *ImageHandler) UploadImage(c *gin.Context) {
	// 获取上传的文件
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, UploadImageResponse{
			Success: false,
			Message: "没有找到图像文件",
		})
		return
	}

	// 检查文件格式
	if !utils.IsValidImageFormat(file.Filename) {
		c.JSON(http.StatusBadRequest, UploadImageResponse{
			Success: false,
			Message: "不支持的图像格式",
		})
		return
	}

	// 检查文件大小
	if file.Size > h.config.Server.MaxFileSize {
		c.JSON(http.StatusBadRequest, UploadImageResponse{
			Success: false,
			Message: fmt.Sprintf("文件大小超过限制 (%d MB)", h.config.Server.MaxFileSize/(1024*1024)),
		})
		return
	}

	// 生成唯一的文件ID
	imageID := uuid.New().String()
	ext := filepath.Ext(file.Filename)
	filename := imageID + ext
	filePath := filepath.Join(h.config.Server.UploadPath, filename)

	// 保存文件
	if err := utils.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, UploadImageResponse{
			Success: false,
			Message: fmt.Sprintf("保存文件失败: %v", err),
		})
		return
	}

	// 加载图像进行特征提取
	img, err := utils.LoadImageFromFile(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, UploadImageResponse{
			Success: false,
			Message: fmt.Sprintf("加载图像失败: %v", err),
		})
		return
	}

	// 提取特征
	features, err := h.featureExtractor.ExtractFeatures(img)
	if err != nil {
		c.JSON(http.StatusInternalServerError, UploadImageResponse{
			Success: false,
			Message: fmt.Sprintf("特征提取失败: %v", err),
		})
		return
	}

	// 插入到Milvus
	if err := h.milvusService.InsertVectors([]string{imageID}, [][]float32{features}); err != nil {
		c.JSON(http.StatusInternalServerError, UploadImageResponse{
			Success: false,
			Message: fmt.Sprintf("向量存储失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, UploadImageResponse{
		Success:   true,
		Message:   "图像上传成功",
		ImageID:   imageID,
		ImagePath: filename,
	})
}

// SearchImage 搜索相似图像API
func (h *ImageHandler) SearchImage(c *gin.Context) {
	// 获取查询参数
	topKStr := c.DefaultQuery("top_k", "10")
	topK, err := strconv.Atoi(topKStr)
	if err != nil || topK <= 0 || topK > 100 {
		c.JSON(http.StatusBadRequest, SearchImageResponse{
			Success: false,
			Message: "无效的top_k参数 (1-100)",
		})
		return
	}

	// 获取上传的查询图像
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, SearchImageResponse{
			Success: false,
			Message: "没有找到查询图像文件",
		})
		return
	}

	// 检查文件格式
	if !utils.IsValidImageFormat(file.Filename) {
		c.JSON(http.StatusBadRequest, SearchImageResponse{
			Success: false,
			Message: "不支持的图像格式",
		})
		return
	}

	// 加载查询图像
	img, err := utils.LoadImageFromMultipart(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, SearchImageResponse{
			Success: false,
			Message: fmt.Sprintf("加载查询图像失败: %v", err),
		})
		return
	}

	// 提取查询图像特征
	queryFeatures, err := h.featureExtractor.ExtractFeatures(img)
	if err != nil {
		c.JSON(http.StatusInternalServerError, SearchImageResponse{
			Success: false,
			Message: fmt.Sprintf("查询图像特征提取失败: %v", err),
		})
		return
	}

	// 在Milvus中搜索相似向量
	searchResults, err := h.milvusService.SearchSimilar(queryFeatures, topK)
	if err != nil {
		c.JSON(http.StatusInternalServerError, SearchImageResponse{
			Success: false,
			Message: fmt.Sprintf("搜索失败: %v", err),
		})
		return
	}

	// 转换搜索结果
	var results []SearchResultWithDetails
	for _, result := range searchResults {
		similarity := h.calculateSimilarity(result.Distance)

		// 查找实际的文件路径
		actualFilePath := h.findActualImageFile(result.ImageID)

		results = append(results, SearchResultWithDetails{
			ImageID:    result.ImageID,
			Score:      result.Score,
			Distance:   result.Distance,
			ImagePath:  actualFilePath,
			Similarity: similarity,
		})
	}

	c.JSON(http.StatusOK, SearchImageResponse{
		Success: true,
		Message: "搜索完成",
		Results: results,
		Total:   len(results),
	})
}

// findActualImageFile 查找实际的图像文件路径
func (h *ImageHandler) findActualImageFile(imageID string) string {
	// 支持的图像扩展名
	extensions := []string{".jpg", ".jpeg", ".png", ".bmp", ".tiff", ".gif"}

	for _, ext := range extensions {
		filename := imageID + ext
		fullPath := filepath.Join(h.config.Server.UploadPath, filename)

		// 检查文件是否存在
		if _, err := os.Stat(fullPath); err == nil {
			return filename // 返回相对路径
		}
	}

	// 如果找不到文件，返回默认的jpg扩展名
	return imageID + ".jpg"
}

// DeleteImage 删除图像API
func (h *ImageHandler) DeleteImage(c *gin.Context) {
	imageID := c.Param("id")
	if imageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "图像ID不能为空",
		})
		return
	}

	// 从Milvus删除向量
	if err := h.milvusService.DeleteVector(imageID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": fmt.Sprintf("删除向量失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "图像删除成功",
	})
}

// GetStats 获取统计信息API
func (h *ImageHandler) GetStats(c *gin.Context) {
	// 获取collection统计信息
	collectionStats, err := h.milvusService.GetCollectionStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, StatsResponse{
			Success: false,
			Message: fmt.Sprintf("获取统计信息失败: %v", err),
		})
		return
	}

	// 服务器信息
	serverInfo := map[string]interface{}{
		"version":       "1.0.0",
		"feature_dim":   h.featureExtractor.GetDimension(),
		"upload_path":   h.config.Server.UploadPath,
		"max_file_size": h.config.Server.MaxFileSize,
		"timestamp":     time.Now().Unix(),
	}

	c.JSON(http.StatusOK, StatsResponse{
		Success:        true,
		Message:        "统计信息获取成功",
		CollectionInfo: collectionStats,
		ServerInfo:     serverInfo,
	})
}

// HealthCheck 健康检查API
func (h *ImageHandler) HealthCheck(c *gin.Context) {
	// 检查Milvus连接
	if err := h.milvusService.HealthCheck(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": fmt.Sprintf("服务不可用: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "服务正常",
		"timestamp": time.Now().Unix(),
	})
}

// calculateSimilarity 计算相似度百分比
func (h *ImageHandler) calculateSimilarity(distance float32) string {
	// 距离越小，相似度越高
	// 这里使用简单的转换公式，可以根据实际情况调整
	similarity := (1.0 - distance) * 100
	if similarity < 0 {
		similarity = 0
	}
	if similarity > 100 {
		similarity = 100
	}

	return fmt.Sprintf("%.1f%%", similarity)
}
