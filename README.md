# 基于Milvus的以图搜图系统

这是一个基于Go语言和Milvus向量数据库构建的以图搜图系统。它使用图像特征提取技术将图像转换为向量，并存储在Milvus中进行高效的相似性搜索。

## 功能特性

- 🖼️ **图像上传**：支持多种图像格式（JPG、PNG、BMP、TIFF）
- 🔍 **相似图像搜索**：基于图像内容进行相似性搜索
- 📊 **特征提取**：结合颜色直方图、纹理特征和空间特征
- 🚀 **高性能**：基于Milvus向量数据库的高效检索
- 🔌 **RESTful API**：完整的REST API接口
- 📈 **系统监控**：提供统计信息和健康检查

## 系统架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   前端/客户端   │    │   Go Web服务器  │    │   Milvus数据库  │
│                 │    │                 │    │                 │
│  - 图像上传     │───▶│  - 特征提取     │───▶│  - 向量存储     │
│  - 搜索界面     │    │  - API处理      │    │  - 相似性搜索   │
│  - 结果展示     │◀───│  - 结果处理     │◀───│  - 索引管理     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## 快速开始

### 1. 环境要求

- Go 1.21+
- Docker & Docker Compose
- 至少4GB内存

### 2. 启动Milvus服务

```bash
# 克隆项目
git clone <your-repo>
cd image-search-go

# 启动Milvus服务
docker-compose up -d

# 等待服务启动完成（约1-2分钟）
docker-compose ps
```

### 3. 安装依赖

```bash
# 下载Go模块依赖
go mod tidy
```

### 4. 配置环境变量（可选）

```bash
# 服务器配置
export SERVER_PORT=8080
export SERVER_HOST=0.0.0.0
export UPLOAD_PATH=./uploads

# Milvus配置
export MILVUS_HOST=localhost
export MILVUS_PORT=19530
export MILVUS_COLLECTION=image_vectors
```

### 5. 启动应用

```bash
# 开发模式
go run main.go

# 或编译后运行
go build -o image-search-server
./image-search-server
```

### 6. 验证服务

访问 http://localhost:8080 查看服务状态

## API接口

### 基础信息

- **Base URL**: `http://localhost:8080/api/v1`
- **Content-Type**: `multipart/form-data` (文件上传)

### 1. 上传图像

```bash
curl -X POST http://localhost:8080/api/v1/images/upload \
  -F "image=@/path/to/your/image.jpg"
```

**响应示例**:
```json
{
  "success": true,
  "message": "图像上传成功",
  "image_id": "550e8400-e29b-41d4-a716-446655440000",
  "image_path": "550e8400-e29b-41d4-a716-446655440000.jpg"
}
```

### 2. 搜索相似图像

```bash
curl -X POST "http://localhost:8080/api/v1/images/search?top_k=5" \
  -F "image=@/path/to/query/image.jpg"
```

**响应示例**:
```json
{
  "success": true,
  "message": "搜索完成",
  "results": [
    {
      "image_id": "550e8400-e29b-41d4-a716-446655440000",
      "score": 0.95,
      "distance": 0.05,
      "image_path": "550e8400-e29b-41d4-a716-446655440000.jpg",
      "similarity": "95.0%"
    }
  ],
  "total": 1
}
```

### 3. 删除图像

```bash
curl -X DELETE http://localhost:8080/api/v1/images/550e8400-e29b-41d4-a716-446655440000
```

### 4. 获取统计信息

```bash
curl http://localhost:8080/api/v1/system/stats
```

### 5. 健康检查

```bash
curl http://localhost:8080/api/v1/system/health
```

## 配置说明

### 环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `SERVER_PORT` | 8080 | 服务端口 |
| `SERVER_HOST` | 0.0.0.0 | 服务主机 |
| `UPLOAD_PATH` | ./uploads | 上传文件目录 |
| `MAX_FILE_SIZE` | 10485760 | 最大文件大小（字节） |
| `MILVUS_HOST` | localhost | Milvus主机 |
| `MILVUS_PORT` | 19530 | Milvus端口 |
| `MILVUS_COLLECTION` | image_vectors | 集合名称 |
| `MILVUS_DIMENSION` | 512 | 特征向量维度 |
| `MILVUS_INDEX_TYPE` | IVF_FLAT | 索引类型 |
| `MILVUS_METRIC_TYPE` | L2 | 距离度量 |

## 特征提取

系统使用多维度特征提取：

1. **颜色直方图**（48维）
   - RGB各通道16个bin的直方图

2. **纹理特征**（4维）
   - 对比度、能量、均匀性、边缘强度

3. **空间特征**（48维）
   - 4x4网格区域的平均颜色特征

4. **其他特征**（412维）
   - 扩展特征，用零填充至512维

## 项目结构

```
image-search-go/
├── config/           # 配置模块
├── handlers/         # HTTP处理器
├── models/           # 数据模型和特征提取
├── services/         # 业务服务层
├── utils/            # 工具函数
├── uploads/          # 上传文件目录
├── docker-compose.yml # Milvus服务配置
├── go.mod           # Go模块文件
├── main.go          # 程序入口
└── README.md        # 项目文档
```

## 开发指南

### 添加新的特征提取器

1. 实现 `models.FeatureExtractor` 接口
2. 在 `main.go` 中替换特征提取器实例

### 自定义索引配置

修改 `config/config.go` 中的默认配置或设置相应环境变量。

### 扩展API

在 `handlers/` 目录下添加新的处理器，并在 `main.go` 中注册路由。

## 性能优化

1. **批量处理**：支持批量上传和特征提取
2. **索引优化**：根据数据规模选择合适的索引类型
3. **缓存策略**：可添加Redis缓存热门搜索结果
4. **并发处理**：特征提取支持并发处理

## 故障排除

### 常见问题

1. **Milvus连接失败**
   - 检查Docker服务是否正常运行
   - 确认端口19530未被占用

2. **上传文件失败**
   - 检查uploads目录权限
   - 确认文件格式支持

3. **搜索结果为空**
   - 确认已上传图像数据
   - 检查collection是否正确创建

### 日志查看

```bash
# 查看应用日志
./image-search-server 2>&1 | tee app.log

# 查看Milvus日志
docker-compose logs -f standalone
```

## 贡献指南

1. Fork项目
2. 创建特性分支
3. 提交代码
4. 创建Pull Request

## 许可证

MIT License

## 联系方式

如有问题或建议，请提交Issue或联系维护者。 