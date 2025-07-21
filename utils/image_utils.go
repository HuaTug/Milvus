package utils

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/nfnt/resize"
)

// SupportedImageTypes 支持的图像格式
var SupportedImageTypes = []string{".jpg", ".jpeg", ".png", ".bmp", ".tiff"}

// ImageInfo 图像信息结构
type ImageInfo struct {
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Format   string `json:"format"`
	Path     string `json:"path"`
}

// LoadImageFromFile 从文件路径加载图像
func LoadImageFromFile(imagePath string) (image.Image, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("无法打开图像文件: %v", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("无法解码图像: %v", err)
	}

	return img, nil
}

// LoadImageFromMultipart 从multipart文件加载图像
func LoadImageFromMultipart(fileHeader *multipart.FileHeader) (image.Image, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("无法打开上传文件: %v", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("无法解码图像: %v", err)
	}

	return img, nil
}

// ResizeImage 调整图像大小，保持宽高比
func ResizeImage(img image.Image, targetSize int) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// 计算新的尺寸，保持宽高比
	var newWidth, newHeight uint
	if width > height {
		newWidth = uint(targetSize)
		newHeight = uint(targetSize * height / width)
	} else {
		newHeight = uint(targetSize)
		newWidth = uint(targetSize * width / height)
	}

	return resize.Resize(newWidth, newHeight, img, resize.Lanczos3)
}

// CenterCrop 中心裁剪图像到指定大小
func CenterCrop(img image.Image, size int) image.Image {
	return imaging.CropCenter(img, size, size)
}

// PreprocessImage 预处理图像：调整大小并中心裁剪
func PreprocessImage(img image.Image, size int) image.Image {
	// 先调整大小，然后中心裁剪
	resized := ResizeImage(img, size+50) // 稍微大一点，便于裁剪
	return CenterCrop(resized, size)
}

// NormalizeImage 归一化图像像素值到[0,1]范围
func NormalizeImage(img image.Image) []float32 {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	pixels := make([]float32, width*height*3) // RGB三通道

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()

			// 将16位RGBA值转换为8位，然后归一化到[0,1]
			idx := ((y-bounds.Min.Y)*width + (x - bounds.Min.X)) * 3
			pixels[idx] = float32(r>>8) / 255.0   // R
			pixels[idx+1] = float32(g>>8) / 255.0 // G
			pixels[idx+2] = float32(b>>8) / 255.0 // B
		}
	}

	return pixels
}

// SaveImage 保存图像到文件
func SaveImage(img image.Image, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("无法创建文件: %v", err)
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".jpg", ".jpeg":
		return jpeg.Encode(file, img, &jpeg.Options{Quality: 90})
	case ".png":
		return png.Encode(file, img)
	default:
		return fmt.Errorf("不支持的图像格式: %s", ext)
	}
}

// SaveUploadedFile 保存上传的文件
func SaveUploadedFile(fileHeader *multipart.FileHeader, destPath string) error {
	src, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	dst, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

// IsValidImageFormat 检查文件是否为支持的图像格式
func IsValidImageFormat(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	for _, supportedExt := range SupportedImageTypes {
		if ext == supportedExt {
			return true
		}
	}
	return false
}

// GetImageInfo 获取图像信息
func GetImageInfo(filePath string) (*ImageInfo, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}

	img, format, err := image.DecodeConfig(file)
	if err != nil {
		return nil, err
	}

	return &ImageInfo{
		Filename: filepath.Base(filePath),
		Size:     fileInfo.Size(),
		Width:    img.Width,
		Height:   img.Height,
		Format:   format,
		Path:     filePath,
	}, nil
}

// ImageToBytes 将图像转换为字节数组
func ImageToBytes(img image.Image, format string) ([]byte, error) {
	var buf bytes.Buffer

	switch strings.ToLower(format) {
	case "jpeg", "jpg":
		err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
		return buf.Bytes(), err
	case "png":
		err := png.Encode(&buf, img)
		return buf.Bytes(), err
	default:
		return nil, fmt.Errorf("不支持的格式: %s", format)
	}
}
