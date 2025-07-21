# Makefile for image-search-go

.PHONY: all build run clean test deps docker-up docker-down help

# 默认目标
all: deps build

# 构建可执行文件
build:
	@echo "构建应用程序..."
	go build -o bin/image-search-server main.go

# 运行应用程序
run:
	@echo "启动应用程序..."
	go run main.go

# 安装依赖
deps:
	@echo "安装Go依赖..."
	go mod tidy
	go mod download

# 测试
test:
	@echo "运行测试..."
	go test -v ./...

# 清理生成的文件
clean:
	@echo "清理文件..."
	rm -f bin/image-search-server
	rm -rf uploads/*

# 启动Milvus服务
docker-up:
	@echo "启动Milvus服务..."
	docker-compose up -d
	@echo "等待服务启动完成..."
	sleep 30
	docker-compose ps

# 停止Milvus服务
docker-down:
	@echo "停止Milvus服务..."
	docker-compose down

# 重启Milvus服务
docker-restart: docker-down docker-up

# 查看Milvus日志
docker-logs:
	docker-compose logs -f standalone

# 开发环境完整设置
dev-setup: docker-up deps
	@echo "开发环境设置完成!"
	@echo "运行 'make run' 启动应用程序"

# 生产环境构建
prod-build:
	@echo "生产环境构建..."
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/image-search-server main.go

# 格式化代码
fmt:
	@echo "格式化代码..."
	go fmt ./...

# 代码检查
lint:
	@echo "代码检查..."
	golangci-lint run

# 创建上传目录
init-dirs:
	@echo "创建必要目录..."
	mkdir -p uploads
	mkdir -p bin

# 帮助信息
help:
	@echo "可用的命令:"
	@echo "  all          - 安装依赖并构建应用"
	@echo "  build        - 构建应用程序"
	@echo "  run          - 运行应用程序"
	@echo "  deps         - 安装Go依赖"
	@echo "  test         - 运行测试"
	@echo "  clean        - 清理生成的文件"
	@echo "  docker-up    - 启动Milvus服务"
	@echo "  docker-down  - 停止Milvus服务"
	@echo "  docker-logs  - 查看Milvus日志"
	@echo "  dev-setup    - 完整开发环境设置"
	@echo "  prod-build   - 生产环境构建"
	@echo "  fmt          - 格式化代码"
	@echo "  lint         - 代码检查"
	@echo "  init-dirs    - 创建必要目录"
	@echo "  help         - 显示此帮助信息" 