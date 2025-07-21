package config

import (
	"os"
	"strconv"
)

// Config 应用配置结构
type Config struct {
	Server ServerConfig `json:"server"`
	Milvus MilvusConfig `json:"milvus"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port        string `json:"port"`
	Host        string `json:"host"`
	UploadPath  string `json:"upload_path"`
	MaxFileSize int64  `json:"max_file_size"`
}

// MilvusConfig Milvus数据库配置
type MilvusConfig struct {
	Host           string `json:"host"`
	Port           string `json:"port"`
	CollectionName string `json:"collection_name"`
	Dimension      int    `json:"dimension"`
	IndexType      string `json:"index_type"`
	MetricType     string `json:"metric_type"`
}

// LoadConfig 加载配置，从环境变量或使用默认值
func LoadConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:        getEnv("SERVER_PORT", "8888"),
			Host:        getEnv("SERVER_HOST", "0.0.0.0"),
			UploadPath:  getEnv("UPLOAD_PATH", "./uploads"),
			MaxFileSize: getEnvAsInt64("MAX_FILE_SIZE", 10*1024*1024), // 10MB
		},
		Milvus: MilvusConfig{
			Host:           getEnv("MILVUS_HOST", "localhost"),
			Port:           getEnv("MILVUS_PORT", "19530"),
			CollectionName: getEnv("MILVUS_COLLECTION", "image_vectors"),
			Dimension:      getEnvAsInt("MILVUS_DIMENSION", 512), // ResNet特征维度
			IndexType:      getEnv("MILVUS_INDEX_TYPE", "IVF_FLAT"),
			MetricType:     getEnv("MILVUS_METRIC_TYPE", "L2"),
		},
	}
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvAsInt 获取环境变量并转换为整数
func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvAsInt64 获取环境变量并转换为int64
func getEnvAsInt64(key string, defaultValue int64) int64 {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}
