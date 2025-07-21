#!/bin/bash

# API测试脚本
# 用于测试图像搜索系统的各个API接口

BASE_URL="http://localhost:8080"
API_BASE="$BASE_URL/api/v1"

echo "=== 图像搜索系统 API 测试 ==="
echo "Base URL: $BASE_URL"
echo

# 测试服务状态
echo "1. 测试服务状态..."
curl -s "$BASE_URL/" | jq .
echo -e "\n"

# 测试健康检查
echo "2. 测试健康检查..."
curl -s "$API_BASE/system/health" | jq .
echo -e "\n"

# 测试统计信息
echo "3. 测试统计信息..."
curl -s "$API_BASE/system/stats" | jq .
echo -e "\n"

# 测试API文档
echo "4. 测试API文档..."
curl -s "$BASE_URL/api" | jq .
echo -e "\n"

# 如果有测试图片，进行上传和搜索测试
if [ -f "test_image.jpg" ]; then
    echo "5. 测试图像上传..."
    UPLOAD_RESULT=$(curl -s -X POST "$API_BASE/images/upload" -F "image=@test_image.jpg")
    echo $UPLOAD_RESULT | jq .
    
    # 提取image_id
    IMAGE_ID=$(echo $UPLOAD_RESULT | jq -r '.image_id')
    
    if [ "$IMAGE_ID" != "null" ] && [ "$IMAGE_ID" != "" ]; then
        echo -e "\n6. 测试图像搜索..."
        curl -s -X POST "$API_BASE/images/search?top_k=5" -F "image=@test_image.jpg" | jq .
        
        echo -e "\n7. 测试图像删除..."
        curl -s -X DELETE "$API_BASE/images/$IMAGE_ID" | jq .
    else
        echo "上传失败，跳过后续测试"
    fi
else
    echo "5. 跳过图像测试（需要test_image.jpg文件）"
fi

echo -e "\n=== 测试完成 ===" 