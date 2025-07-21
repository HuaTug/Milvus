package main

import (
	"log"
	"net/http"
	"os"

	"image-search-go/config"
	"image-search-go/handlers"
	"image-search-go/models"
	"image-search-go/services"

	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	cfg := config.LoadConfig()
	log.Printf("配置加载完成: %+v", cfg)

	// 确保上传目录存在
	if err := os.MkdirAll(cfg.Server.UploadPath, 0755); err != nil {
		log.Fatalf("创建上传目录失败: %v", err)
	}

	// 初始化特征提取器
	featureExtractor := models.NewSimpleFeatureExtractor()
	log.Printf("特征提取器初始化完成，维度: %d", featureExtractor.GetDimension())

	// 初始化Milvus服务
	milvusService, err := services.NewMilvusService(&cfg.Milvus)
	if err != nil {
		log.Fatalf("Milvus服务初始化失败: %v", err)
	}
	defer milvusService.Close()

	// 初始化处理器
	imageHandler := handlers.NewImageHandler(milvusService, featureExtractor, cfg)

	// 设置Gin模式
	if os.Getenv("GIN_MODE") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建路由器
	router := gin.Default()

	// 设置静态文件服务（用于显示上传的图片）
	router.Static("/uploads", cfg.Server.UploadPath)

	// 添加CORS中间件
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Authorization, Content-Type")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	// API路由组
	v1 := router.Group("/api/v1")
	{
		// 图像相关API
		images := v1.Group("/images")
		{
			images.POST("/upload", imageHandler.UploadImage) // 上传图像
			images.POST("/search", imageHandler.SearchImage) // 搜索相似图像
			images.DELETE("/:id", imageHandler.DeleteImage)  // 删除图像
		}

		// 系统API
		system := v1.Group("/system")
		{
			system.GET("/stats", imageHandler.GetStats)     // 获取统计信息
			system.GET("/health", imageHandler.HealthCheck) // 健康检查
		}
	}

	// 添加根路径处理
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "图像搜索服务",
			"version": "1.0.0",
			"endpoints": gin.H{
				"upload": "POST /api/v1/images/upload",
				"search": "POST /api/v1/images/search",
				"delete": "DELETE /api/v1/images/:id",
				"stats":  "GET /api/v1/system/stats",
				"health": "GET /api/v1/system/health",
			},
		})
	})

	// 添加API文档路由
	router.GET("/api", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service":     "图像搜索API",
			"version":     "1.0.0",
			"description": "基于Milvus的以图搜图服务",
			"apis": []gin.H{
				{
					"path":        "/api/v1/images/upload",
					"method":      "POST",
					"description": "上传图像并提取特征存储到向量数据库",
					"parameters":  "image (multipart file)",
				},
				{
					"path":        "/api/v1/images/search",
					"method":      "POST",
					"description": "搜索相似图像",
					"parameters":  "image (multipart file), top_k (query parameter, default: 10)",
				},
				{
					"path":        "/api/v1/images/:id",
					"method":      "DELETE",
					"description": "删除指定图像的向量数据",
					"parameters":  "id (path parameter)",
				},
				{
					"path":        "/api/v1/system/stats",
					"method":      "GET",
					"description": "获取系统统计信息",
					"parameters":  "none",
				},
				{
					"path":        "/api/v1/system/health",
					"method":      "GET",
					"description": "健康检查",
					"parameters":  "none",
				},
			},
		})
	})

	// 启动服务器
	address := cfg.Server.Host + ":" + cfg.Server.Port
	log.Printf("服务器启动在: http://%s", address)
	log.Printf("API文档: http://%s/api", address)
	log.Printf("上传目录: %s", cfg.Server.UploadPath)

	if err := router.Run(address); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
