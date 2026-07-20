package services

import (
	"bytes"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"math/big"
	"strings"
	"sync"
	"time"

	"goWFM/config"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// captchaStore 内存验证码存储
type captchaStore struct {
	mu      sync.RWMutex
	records map[string]*captchaRecord
}

type captchaRecord struct {
	answer   string
	expireAt time.Time
}

// GlobalCaptchaStore 全局验证码存储实例
var GlobalCaptchaStore = &captchaStore{
	records: make(map[string]*captchaRecord),
}

//go:embed fonts/*
var embeddedFonts embed.FS

// customFontData 存储解析后的 TTF 字体，若未提供自定义字体则为 nil
var customFontData *truetype.Font

func init() {
	entries, err := embeddedFonts.ReadDir("fonts")
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".ttf") {
			data, err := embeddedFonts.ReadFile("fonts/" + entry.Name())
			if err != nil {
				continue
			}
			f, err := truetype.Parse(data)
			if err != nil {
				continue
			}
			customFontData = f
			break
		}
	}
}

// getFontFace 返回用于绘制验证码文字的 font.Face。
// 若已加载自定义 TTF 字体则优先使用，否则回退到 basicfont。
func getFontFace() font.Face {
	if customFontData != nil {
		return truetype.NewFace(customFontData, &truetype.Options{
			Size:    24,
			DPI:     72,
			Hinting: font.HintingFull,
		})
	}
	return basicfont.Face7x13
}

// GenerateCaptcha 生成验证码
// 返回 captchaID, answer, svgImage
func GenerateCaptcha() (string, string, string) {
	// 排除容易混淆的 0, O, 1, I
	const captchaChars = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"
	codeLength := config.GetSecurity().CaptchaCodeLength

	// 生成4位验证码
	answer := make([]byte, codeLength)
	for i := range answer {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(captchaChars))))
		answer[i] = captchaChars[n.Int64()]
	}
	answerStr := string(answer)

	// 生成 captchaID
	idBytes := make([]byte, 16)
	rand.Read(idBytes)
	captchaID := fmt.Sprintf("%x", idBytes)

	// 生成 PNG 图片（Base64 编码）
	png := generateCaptchaPng(answerStr)

	// 存储验证码（5分钟有效）
	GlobalCaptchaStore.mu.Lock()
	GlobalCaptchaStore.records[captchaID] = &captchaRecord{
		answer:   strings.ToUpper(answerStr),
		expireAt: time.Now().Add(5 * time.Minute),
	}
	GlobalCaptchaStore.mu.Unlock()

	return captchaID, answerStr, png
}

// VerifyCaptcha 验证验证码
func VerifyCaptcha(captchaID, userInput string) bool {
	if captchaID == "" || userInput == "" {
		return false
	}

	GlobalCaptchaStore.mu.Lock()
	defer GlobalCaptchaStore.mu.Unlock()

	record, ok := GlobalCaptchaStore.records[captchaID]
	if !ok {
		return false
	}

	// 验证后立即删除（一次性使用）
	delete(GlobalCaptchaStore.records, captchaID)

	// 检查是否过期
	if time.Now().After(record.expireAt) {
		return false
	}

	return strings.ToUpper(userInput) == record.answer
}

// CleanExpiredCaptchas 清理过期验证码
func CleanExpiredCaptchas() int {
	GlobalCaptchaStore.mu.Lock()
	defer GlobalCaptchaStore.mu.Unlock()

	now := time.Now()
	count := 0
	for id, record := range GlobalCaptchaStore.records {
		if now.After(record.expireAt) {
			delete(GlobalCaptchaStore.records, id)
			count++
		}
	}
	return count
}

// generateCaptchaPng 生成验证码png图片
// 返回 base64 字符串
func generateCaptchaPng(text string) string {
	const (
		imgWidth  = 114
		imgHeight = 48
	)
	// 1. 創建文字暫存畫布（先將文字繪製到這裡，再進行扭曲）
	textImg := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))
	// 背景設為完全透明，方便後續像素抽樣
	draw.Draw(textImg, textImg.Bounds(), image.Transparent, image.Point{}, draw.Src)

	// 2. 繪製原始文字（加入動態間距與隨機上下位移）
	face := getFontFace()

	if customFontData != nil {
		// TTF 自定義字體渲染：根據實際字符寬度計算間距並居中
		drawer := &font.Drawer{Dst: textImg, Face: face}
		totalAdv := font.MeasureString(face, text)
		startX := (imgWidth - totalAdv.Ceil()) / 2

		currentX := startX
		for _, char := range text {
			charStr := string(char)
			charAdv := font.MeasureString(face, charStr)

			x := currentX + randomInt(6)
			y := imgHeight/2 + face.Metrics().Ascent.Ceil()/2 - 2 + randomInt(10) - 5

			drawer.Src = image.NewUniform(randomColor(0, 80))
			drawer.Dot = fixed.Point26_6{
				X: fixed.Int26_6(x << 6),
				Y: fixed.Int26_6(y << 6),
			}
			drawer.DrawString(charStr)

			currentX += charAdv.Ceil() + randomInt(3)
		}
	} else {
		// 回退到內建 basicfont.Face7x13
		cp := imgWidth / len(text) / 2
		for i, char := range text {
			charColor := randomColor(0, 80)
			x := cp + i*cp*2 + randomInt(6)
			y := 20 + randomInt(20)

			dot := fixed.Point26_6{X: fixed.Int26_6(x << 6), Y: fixed.Int26_6(y << 6)}
			d := &font.Drawer{
				Dst:  textImg,
				Src:  image.NewUniform(charColor),
				Face: face,
				Dot:  dot,
			}
			d.DrawString(string(char))
		}
	}

	// 4. 創建最終畫布並填滿背景色
	finalImg := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))
	bgColor := randomColor(220, 255)
	draw.Draw(finalImg, finalImg.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)

	// 5. 像素級非線性幾何扭曲（核心抗 OCR 演算法）
	// 將 textImg 的像素經過正弦波與隨機斜體運算後，映射到 finalImg
	amplitude := 3.0 + float64(randomInt(4))       // 隨機縱向波浪振幅
	frequency := 0.04 + float64(randomInt(3))*0.01 // 隨機頻率
	skewFactor := float64(randomInt(10)-5) / 30.0  // 隨機左右傾斜斜率

	for x := 0; x < imgWidth; x++ {
		for y := 0; y < imgHeight; y++ {
			// 取得原始像素顏色
			srcColor := textImg.At(x, y)
			_, _, _, alpha := srcColor.RGBA()

			if alpha > 0 { // 如果該點有文字像素
				// 計算波浪形變 Y 軸偏移
				offsetY := amplitude * math.Sin(float64(x)*frequency)
				// 計算傾斜形變 X 軸偏移
				offsetX := float64(y-imgHeight/2) * skewFactor

				newX := x + int(offsetX)
				newY := y + int(offsetY)

				// 確保像素座標在畫布範圍內
				if newX >= 0 && newX < imgWidth && newY >= 0 && newY < imgHeight {
					finalImg.Set(newX, newY, srcColor)
					// 放大像素（上下左右擴展），使 7x13 點陣字在外觀上變大變厚
					finalImg.Set(newX+1, newY, srcColor)
					finalImg.Set(newX, newY+1, srcColor)
					finalImg.Set(newX+1, newY+1, srcColor)
				}
			}
		}
	}

	// 6. 繪製防自動化干擾線（交叉多條）
	for i := 0; i < 3; i++ {
		lineColor := randomColor(120, 180)
		amplitude := 4.0 + float64(randomInt(8))
		phase := float64(randomInt(360))
		frequency := 0.06
		for x := 0; x < imgWidth; x++ {
			y := float64(imgHeight)/2 + amplitude*math.Sin(float64(x)*frequency+phase)
			finalImg.Set(x, int(y), lineColor)
			finalImg.Set(x, int(y)+1, lineColor) // 線條加粗
		}
	}

	// 7. 繪製隨機噪點
	for i := 0; i < 80; i++ {
		nx := randomInt(imgWidth)
		ny := randomInt(imgHeight)
		finalImg.Set(nx, ny, randomColor(100, 200))
	}

	// 8. 輸出 Base64
	var buf bytes.Buffer
	_ = png.Encode(&buf, finalImg)
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

// 随机颜色
func randomColor(min, max int) color.RGBA {
	delta := max - min
	return color.RGBA{
		R: uint8(min + randomInt(delta)),
		G: uint8(min + randomInt(delta)),
		B: uint8(min + randomInt(delta)),
		A: 255,
	}
}

func randomInt(max int) int {
	nBig, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
	return int(nBig.Int64())
}
