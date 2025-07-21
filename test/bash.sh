#!/bin/bash
# FILEPATH: /data/workspace/image-search-go/scripts/download_datasets.sh
# INSTRUCTIONS: 创建数据集下载脚本

set -e

DATASETS_DIR="./datasets"
mkdir -p "$DATASETS_DIR"

echo "=== 图像搜索数据集下载工具 ==="

download_cifar10() {
    echo "下载 CIFAR-10 数据集..."
    cd "$DATASETS_DIR"
    
    if [ ! -d "cifar-10" ]; then
        # 下载Python版本的CIFAR-10
        wget -c https://www.cs.toronto.edu/~kriz/cifar-10-python.tar.gz
        tar -xzf cifar-10-python.tar.gz
        mv cifar-10-batches-py cifar-10
        rm cifar-10-python.tar.gz
        
        # 转换为图像文件
        python3 << 'EOF'
import pickle
import numpy as np
from PIL import Image
import os

def unpickle(file):
    with open(file, 'rb') as fo:
        dict = pickle.load(fo, encoding='bytes')
    return dict

# 类别名称
label_names = ['airplane', 'automobile', 'bird', 'cat', 'deer', 'dog', 'frog', 'horse', 'ship', 'truck']

# 创建输出目录
os.makedirs('cifar-10-images', exist_ok=True)
for name in label_names:
    os.makedirs(f'cifar-10-images/{name}', exist_ok=True)

# 处理训练数据
for i in range(1, 6):
    batch = unpickle(f'cifar-10/data_batch_{i}')
    data = batch[b'data']
    labels = batch[b'labels']
    
    for j, (img_data, label) in enumerate(zip(data, labels)):
        # 重塑数据为32x32x3
        img_data = img_data.reshape(3, 32, 32).transpose(1, 2, 0)
        img = Image.fromarray(img_data)
        
        # 保存图像
        filename = f'cifar-10-images/{label_names[label]}/batch{i}_{j:04d}.png'
        img.save(filename)

# 处理测试数据
test_batch = unpickle('cifar-10/test_batch')
data = test_batch[b'data']
labels = test_batch[b'labels']

for j, (img_data, label) in enumerate(zip(data, labels)):
    img_data = img_data.reshape(3, 32, 32).transpose(1, 2, 0)
    img = Image.fromarray(img_data)
    filename = f'cifar-10-images/{label_names[label]}/test_{j:04d}.png'
    img.save(filename)

print("CIFAR-10 图像转换完成")
EOF
        
        echo "CIFAR-10 数据集准备完成: $DATASETS_DIR/cifar-10-images"
    else
        echo "CIFAR-10 数据集已存在"
    fi
    
    cd ..
}

download_sample_images() {
    echo "下载示例图像数据集..."
    cd "$DATASETS_DIR"
    
    if [ ! -d "sample-images" ]; then
        mkdir -p sample-images
        cd sample-images
        
        # 下载一些示例图像
        echo "下载示例图像..."
        
        # 使用 Unsplash API 下载一些示例图像
        for i in {1..50}; do
            wget -q -O "sample_${i}.jpg" "https://picsum.photos/400/300?random=${i}" || true
            echo "下载图像 ${i}/50"
        done
        
        echo "示例图像下载完成: $DATASETS_DIR/sample-images"
        cd ..
    else
        echo "示例图像数据集已存在"
    fi
    
    cd ..
}

# 主菜单
echo "请选择要下载的数据集:"
echo "1) CIFAR-10 (推荐，约170MB)"
echo "2) 示例图像 (50张随机图像，约5MB)"
echo "3) 全部下载"
echo "4) 退出"

read -p "请输入选择 (1-4): " choice

case $choice in
    1)
        download_cifar10
        ;;
    2)
        download_sample_images
        ;;
    3)
        download_cifar10
        download_sample_images
        ;;
    4)
        echo "退出"
        exit 0
        ;;
    *)
        echo "无效选择"
        exit 1
        ;;
esac

echo ""
echo "=== 数据集下载完成 ==="
echo "可用数据集:"
ls -la "$DATASETS_DIR"
echo ""
echo "使用方法:"
echo "go run cmd/batch_insert/main.go -dataset ./datasets/cifar-10-images -batch 50 -workers 4"
