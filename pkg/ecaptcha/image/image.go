// Package image 提供图片验证码生成和验证
package image

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"global-track/pkg/ecaptcha"
)

// ImageData 图片验证码数据 (发给前端)
type ImageData struct {
	Image string `json:"image"` // Base64 编码的图片
}

// ImageAnswer 图片验证码答案 (前端提交)
type ImageAnswer struct {
	Code string `json:"code"` // 用户输入的验证码
}

// storedData 存储的验证数据
type storedData struct {
	Code      string `json:"code"`
	ExpiresAt int64  `json:"expires_at"`
}

// Provider 图片验证码提供者
type Provider struct {
	config ecaptcha.Config
	store  ecaptcha.Store
}

// New 创建图片验证码提供者
func New(config ecaptcha.Config, store ecaptcha.Store) *Provider {
	return &Provider{
		config: config,
		store:  store,
	}
}

// Type 返回验证类型
func (p *Provider) Type() ecaptcha.CaptchaType {
	return ecaptcha.TypeImage
}

// Generate 生成图片验证码
func (p *Provider) Generate(ctx context.Context) (*ecaptcha.Challenge, error) {
	// 生成随机验证码
	code := generateCode(p.config.ImageLength)
	
	// 生成图片
	img := generateImage(code, p.config.ImageWidth, p.config.ImageHeight)
	
	// 编码为 Base64
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	imageBase64 := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
	
	// 生成挑战 ID
	id := uuid.New().String()
	expiresAt := time.Now().Add(p.config.Expire).Unix()
	
	// 存储验证数据
	data, _ := json.Marshal(&storedData{
		Code:      strings.ToLower(code), // 存储小写，验证时忽略大小写
		ExpiresAt: expiresAt,
	})
	if err := p.store.Set(ctx, id, data, p.config.Expire); err != nil {
		return nil, err
	}
	
	return &ecaptcha.Challenge{
		ID:        id,
		Type:      ecaptcha.TypeImage,
		Data:      &ImageData{Image: imageBase64},
		ExpiresAt: expiresAt,
	}, nil
}

// Verify 验证用户答案
func (p *Provider) Verify(ctx context.Context, req *ecaptcha.VerifyRequest) (*ecaptcha.VerifyResult, error) {
	// 检查尝试次数
	attempts, err := p.store.IncrAttempts(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if attempts > p.config.MaxAttempts {
		p.store.Delete(ctx, req.ID)
		return &ecaptcha.VerifyResult{
			Success: false,
			Message: "验证次数过多，请重新获取",
		}, nil
	}
	
	// 获取存储的验证数据
	data, err := p.store.Get(ctx, req.ID)
	if err != nil {
		return &ecaptcha.VerifyResult{
			Success: false,
			Message: "验证码不存在或已过期",
		}, nil
	}
	
	var stored storedData
	if err := json.Unmarshal(data, &stored); err != nil {
		return nil, err
	}
	
	// 检查是否过期
	if time.Now().Unix() > stored.ExpiresAt {
		p.store.Delete(ctx, req.ID)
		return &ecaptcha.VerifyResult{
			Success: false,
			Message: "验证码已过期",
		}, nil
	}
	
	// 解析用户答案
	var answer ImageAnswer
	answerBytes, _ := json.Marshal(req.Answer)
	if err := json.Unmarshal(answerBytes, &answer); err != nil {
		return &ecaptcha.VerifyResult{
			Success: false,
			Message: "答案格式错误",
		}, nil
	}
	
	// 验证答案 (忽略大小写)
	if strings.ToLower(strings.TrimSpace(answer.Code)) != stored.Code {
		return &ecaptcha.VerifyResult{
			Success: false,
			Message: "验证码错误",
		}, nil
	}
	
	// 验证成功，删除验证数据
	p.store.Delete(ctx, req.ID)
	
	// 生成验证 token
	token := uuid.New().String()
	tokenExpire := time.Now().Add(p.config.TokenExpire).Unix()
	p.store.Set(ctx, "token:"+token, []byte("valid"), p.config.TokenExpire)
	
	return &ecaptcha.VerifyResult{
		Success:   true,
		Token:     token,
		Score:     0,
		Message:   "验证成功",
		ExpiresAt: tokenExpire,
	}, nil
}

// generateCode 生成随机验证码
func generateCode(length int) string {
	// 使用易于辨认的字符 (排除 0O1lI 等易混淆字符)
	chars := "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
	code := make([]byte, length)
	for i := 0; i < length; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		code[i] = chars[n.Int64()]
	}
	return string(code)
}

// generateImage 生成验证码图片
func generateImage(code string, width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	
	// 背景色 (浅灰色)
	bgColor := color.RGBA{240, 240, 240, 255}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, bgColor)
		}
	}
	
	// 添加干扰线
	addNoiseLines(img, 5)
	
	// 添加干扰点
	addNoiseDots(img, 100)
	
	// 绘制文字
	drawText(img, code)
	
	return img
}

// addNoiseLines 添加干扰线
func addNoiseLines(img *image.RGBA, count int) {
	bounds := img.Bounds()
	for i := 0; i < count; i++ {
		x1, _ := rand.Int(rand.Reader, big.NewInt(int64(bounds.Max.X)))
		y1, _ := rand.Int(rand.Reader, big.NewInt(int64(bounds.Max.Y)))
		x2, _ := rand.Int(rand.Reader, big.NewInt(int64(bounds.Max.X)))
		y2, _ := rand.Int(rand.Reader, big.NewInt(int64(bounds.Max.Y)))
		
		lineColor := randomColor()
		drawLine(img, int(x1.Int64()), int(y1.Int64()), int(x2.Int64()), int(y2.Int64()), lineColor)
	}
}

// addNoiseDots 添加干扰点
func addNoiseDots(img *image.RGBA, count int) {
	bounds := img.Bounds()
	for i := 0; i < count; i++ {
		x, _ := rand.Int(rand.Reader, big.NewInt(int64(bounds.Max.X)))
		y, _ := rand.Int(rand.Reader, big.NewInt(int64(bounds.Max.Y)))
		img.Set(int(x.Int64()), int(y.Int64()), randomColor())
	}
}

// drawText 绘制文字
func drawText(img *image.RGBA, text string) {
	bounds := img.Bounds()
	face := basicfont.Face7x13
	
	// 计算文字位置 (居中)
	textWidth := len(text) * 10
	startX := (bounds.Max.X - textWidth) / 2
	startY := bounds.Max.Y/2 + 5
	
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.RGBA{50, 50, 50, 255}),
		Face: face,
		Dot:  fixed.Point26_6{X: fixed.I(startX), Y: fixed.I(startY)},
	}
	
	for _, c := range text {
		d.DrawString(string(c))
		d.Dot.X += fixed.I(2) // 字符间距
	}
}

// drawLine 绘制线条 (Bresenham 算法)
func drawLine(img *image.RGBA, x1, y1, x2, y2 int, c color.Color) {
	dx := abs(x2 - x1)
	dy := abs(y2 - y1)
	sx, sy := 1, 1
	if x1 >= x2 {
		sx = -1
	}
	if y1 >= y2 {
		sy = -1
	}
	err := dx - dy

	for {
		img.Set(x1, y1, c)
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := err * 2
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

// randomColor 生成随机颜色
func randomColor() color.RGBA {
	r, _ := rand.Int(rand.Reader, big.NewInt(200))
	g, _ := rand.Int(rand.Reader, big.NewInt(200))
	b, _ := rand.Int(rand.Reader, big.NewInt(200))
	return color.RGBA{uint8(r.Int64() + 50), uint8(g.Int64() + 50), uint8(b.Int64() + 50), 255}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
