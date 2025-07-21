package models

import (
	"fmt"
	"image"
	"image-search-go/utils"
	"math"
)

// FeatureExtractor 图像特征提取器接口
type FeatureExtractor interface {
	ExtractFeatures(img image.Image) ([]float32, error)
	GetDimension() int
}

// SimpleFeatureExtractor 简单的特征提取器（基于颜色直方图和纹理特征）
type SimpleFeatureExtractor struct {
	Dimension int
}

// NewSimpleFeatureExtractor 创建简单特征提取器
func NewSimpleFeatureExtractor() *SimpleFeatureExtractor {
	return &SimpleFeatureExtractor{
		Dimension: 512, // 特征向量维度
	}
}

// ExtractFeatures 提取图像特征
func (e *SimpleFeatureExtractor) ExtractFeatures(img image.Image) ([]float32, error) {
	// 预处理图像
	processed := utils.PreprocessImage(img, 224)

	// 提取颜色直方图特征
	colorFeatures := e.extractColorHistogram(processed)

	// 提取纹理特征
	textureFeatures := e.extractTextureFeatures(processed)

	// 提取空间特征
	spatialFeatures := e.extractSpatialFeatures(processed)

	// 合并所有特征
	features := append(colorFeatures, textureFeatures...)
	features = append(features, spatialFeatures...)

	// 确保特征向量长度为512维
	if len(features) > e.Dimension {
		features = features[:e.Dimension]
	} else if len(features) < e.Dimension {
		// 用零填充
		padding := make([]float32, e.Dimension-len(features))
		features = append(features, padding...)
	}

	// L2归一化
	return e.l2Normalize(features), nil
}

// GetDimension 获取特征向量维度
func (e *SimpleFeatureExtractor) GetDimension() int {
	return e.Dimension
}

// extractColorHistogram 提取颜色直方图特征
func (e *SimpleFeatureExtractor) extractColorHistogram(img image.Image) []float32 {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// RGB颜色直方图，每个通道16个bin
	histR := make([]int, 16)
	histG := make([]int, 16)
	histB := make([]int, 16)

	totalPixels := width * height

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()

			// 将16位值转换为8位，然后分bin
			rBin := (r >> 8) / 16
			gBin := (g >> 8) / 16
			bBin := (b >> 8) / 16

			if rBin > 15 {
				rBin = 15
			}
			if gBin > 15 {
				gBin = 15
			}
			if bBin > 15 {
				bBin = 15
			}

			histR[rBin]++
			histG[gBin]++
			histB[bBin]++
		}
	}

	// 归一化直方图
	features := make([]float32, 48) // 16*3
	for i := 0; i < 16; i++ {
		features[i] = float32(histR[i]) / float32(totalPixels)
		features[i+16] = float32(histG[i]) / float32(totalPixels)
		features[i+32] = float32(histB[i]) / float32(totalPixels)
	}

	return features
}

// extractTextureFeatures 提取纹理特征（基于灰度共生矩阵的简化版本）
func (e *SimpleFeatureExtractor) extractTextureFeatures(img image.Image) []float32 {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// 转换为灰度图像
	grayImg := make([][]float32, height)
	for i := range grayImg {
		grayImg[i] = make([]float32, width)
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			// 灰度转换
			gray := float32(0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8))
			grayImg[y-bounds.Min.Y][x-bounds.Min.X] = gray / 255.0
		}
	}

	// 计算纹理特征：对比度、能量、均匀性
	var contrast, energy, uniformity float32

	for y := 0; y < height-1; y++ {
		for x := 0; x < width-1; x++ {
			// 水平方向
			diff := grayImg[y][x] - grayImg[y][x+1]
			contrast += diff * diff

			// 垂直方向
			diff = grayImg[y][x] - grayImg[y+1][x]
			contrast += diff * diff

			// 能量
			energy += grayImg[y][x] * grayImg[y][x]
		}
	}

	// 归一化
	pixelCount := float32((width - 1) * (height - 1))
	contrast /= pixelCount
	energy /= float32(width * height)
	uniformity = energy // 简化的均匀性度量

	// 添加边缘检测特征
	edgeStrength := e.calculateEdgeStrength(grayImg)

	return []float32{contrast, energy, uniformity, edgeStrength}
}

// extractSpatialFeatures 提取空间特征
func (e *SimpleFeatureExtractor) extractSpatialFeatures(img image.Image) []float32 {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// 将图像分为4x4网格，计算每个区域的平均颜色
	gridSize := 4
	cellWidth := width / gridSize
	cellHeight := height / gridSize

	features := make([]float32, gridSize*gridSize*3) // 4x4网格，每个区域3个颜色通道

	for gy := 0; gy < gridSize; gy++ {
		for gx := 0; gx < gridSize; gx++ {
			var totalR, totalG, totalB float64
			var count int

			startX := gx * cellWidth
			endX := (gx + 1) * cellWidth
			startY := gy * cellHeight
			endY := (gy + 1) * cellHeight

			if endX > width {
				endX = width
			}
			if endY > height {
				endY = height
			}

			for y := startY; y < endY; y++ {
				for x := startX; x < endX; x++ {
					r, g, b, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
					totalR += float64(r >> 8)
					totalG += float64(g >> 8)
					totalB += float64(b >> 8)
					count++
				}
			}

			if count > 0 {
				idx := (gy*gridSize + gx) * 3
				features[idx] = float32(totalR / float64(count) / 255.0)
				features[idx+1] = float32(totalG / float64(count) / 255.0)
				features[idx+2] = float32(totalB / float64(count) / 255.0)
			}
		}
	}

	return features
}

// calculateEdgeStrength 计算边缘强度
func (e *SimpleFeatureExtractor) calculateEdgeStrength(grayImg [][]float32) float32 {
	height := len(grayImg)
	width := len(grayImg[0])

	var totalEdgeStrength float32

	// Sobel算子
	sobelX := [][]int{{-1, 0, 1}, {-2, 0, 2}, {-1, 0, 1}}
	sobelY := [][]int{{-1, -2, -1}, {0, 0, 0}, {1, 2, 1}}

	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			var gx, gy float32

			// 应用Sobel算子
			for ky := 0; ky < 3; ky++ {
				for kx := 0; kx < 3; kx++ {
					pixel := grayImg[y+ky-1][x+kx-1]
					gx += pixel * float32(sobelX[ky][kx])
					gy += pixel * float32(sobelY[ky][kx])
				}
			}

			// 计算梯度幅值
			magnitude := float32(math.Sqrt(float64(gx*gx + gy*gy)))
			totalEdgeStrength += magnitude
		}
	}

	return totalEdgeStrength / float32((width-2)*(height-2))
}

// l2Normalize L2归一化
func (e *SimpleFeatureExtractor) l2Normalize(features []float32) []float32 {
	var norm float64
	for _, f := range features {
		norm += float64(f * f)
	}
	norm = math.Sqrt(norm)

	if norm == 0 {
		return features
	}

	normalized := make([]float32, len(features))
	for i, f := range features {
		normalized[i] = float32(float64(f) / norm)
	}

	return normalized
}

// BatchExtractFeatures 批量提取特征
func (e *SimpleFeatureExtractor) BatchExtractFeatures(images []image.Image) ([][]float32, error) {
	features := make([][]float32, len(images))

	for i, img := range images {
		feature, err := e.ExtractFeatures(img)
		if err != nil {
			return nil, fmt.Errorf("提取第%d张图像特征失败: %v", i, err)
		}
		features[i] = feature
	}

	return features, nil
}
