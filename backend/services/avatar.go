package services

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"

	_ "golang.org/x/image/webp"
)

const MaxAvatarBytes int64 = 2 << 20

var supportedAvatarTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

func BuildAvatarData(data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("头像文件不能为空")
	}
	if int64(len(data)) > MaxAvatarBytes {
		return "", fmt.Errorf("头像文件不能超过 2 MB")
	}

	contentType := http.DetectContentType(data)
	if !supportedAvatarTypes[contentType] {
		return "", fmt.Errorf("仅支持 JPG、PNG 或 WebP 格式的头像")
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width <= 0 || config.Height <= 0 {
		return "", fmt.Errorf("头像图片内容无效")
	}
	if config.Width > 4096 || config.Height > 4096 || int64(config.Width)*int64(config.Height) > 16_000_000 {
		return "", fmt.Errorf("头像图片尺寸不能超过 4096 × 4096")
	}

	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}
