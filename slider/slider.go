// Package slider 提供滑动拼图验证
package slider

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
	"time"

	"github.com/google/uuid"

	"global-track/pkg/ecaptcha"
)

// SliderData 滑动拼图数据 (发给前端)
type SliderData struct {
	Background string `json:"background"` // 背景图 Base64
	Piece      string `json:"piece"`      // 拼图块 Base64
	PieceY     int    `json:"piece_y"`    // 拼图块 Y 坐标
}

// SliderAnswer 滑动拼图答案 (前端提交)
type SliderAnswer struct {
	X         int     `json:"x"`          // 用户滑动到的 X 坐标
	Duration  int64   `json:"duration"`   // 滑动耗时 (ms)
	Trail     []int   `json:"trail"`      // 滑动轨迹 X 坐标数组
}

// storedData 存储的验证数据
type storedData struct {
	TargetX   int   `json:"target_x"`   // 正确的 X 坐标
	ExpiresAt int64 `json:"expires_at"`
}

// Provider 滑动拼图提供者
type Provider struct {
	config ecaptcha.Config
	store  ecaptcha.Store
}

// New 创建滑动拼图提供者
func New(config ecaptcha.Config, store ecaptcha.Store) *Provider {
	return &Provider{
		config: config,
		store:  store,
	}
}

// Type 返回验证类型
func (p *Provider) Type() ecaptcha.CaptchaType {
	return ecaptcha.TypeSlider
}

// Generate 生成滑动拼图
func (p *Provider) Generate(ctx context.Context) (*ecaptcha.Challenge, error) {
	width := p.config.SliderWidth
	height := p.config.SliderHeight
	pieceSize := p.config.PieceSize
	
	// 随机生成拼图块位置 (X 在右半部分，避免太靠边)
	minX := width / 3
	maxX := width - pieceSize - 10
	targetXBig, _ := rand.Int(rand.Reader, big.NewInt(int64(maxX-minX)))
	targetX := int(targetXBig.Int64()) + minX
	
	minY := 10
	maxY := height - pieceSize - 10
	targetYBig, _ := rand.Int(rand.Reader, big.NewInt(int64(maxY-minY)))
	targetY := int(targetYBig.Int64()) + minY
	
	// 生成背景图和拼图块
	background, piece := generateSliderImages(width, height, pieceSize, targetX, targetY)
	
	// 编码为 Base64
	var bgBuf, pieceBuf bytes.Buffer
	png.Encode(&bgBuf, background)
	png.Encode(&pieceBuf, piece)
	
	bgBase64 := "data:image/png;base64," + base64.StdEncoding.EncodeToString(bgBuf.Bytes())
	pieceBase64 := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pieceBuf.Bytes())
	
	// 生成挑战 ID
	id := uuid.New().String()
	expiresAt := time.Now().Add(p.config.Expire).Unix()
	
	// 存储验证数据
	data, _ := json.Marshal(&storedData{
		TargetX:   targetX,
		ExpiresAt: expiresAt,
	})
	if err := p.store.Set(ctx, id, data, p.config.Expire); err != nil {
		return nil, err
	}
	
	return &ecaptcha.Challenge{
		ID:        id,
		Type:      ecaptcha.TypeSlider,
		Data: &SliderData{
			Background: bgBase64,
			Piece:      pieceBase64,
			PieceY:     targetY,
		},
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
	var answer SliderAnswer
	answerBytes, _ := json.Marshal(req.Answer)
	if err := json.Unmarshal(answerBytes, &answer); err != nil {
		return &ecaptcha.VerifyResult{
			Success: false,
			Message: "答案格式错误",
		}, nil
	}
	
	// 验证位置 (允许一定容差)
	tolerance := p.config.Tolerance
	if abs(answer.X-stored.TargetX) > tolerance {
		return &ecaptcha.VerifyResult{
			Success: false,
			Message: "验证失败，请重试",
		}, nil
	}
	
	// 行为分析 (检测机器人)
	score := analyzeSliderBehavior(answer)
	if score > 0.8 {
		return &ecaptcha.VerifyResult{
			Success: false,
			Score:   score,
			Message: "检测到异常行为",
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
		Score:     score,
		Message:   "验证成功",
		ExpiresAt: tokenExpire,
	}, nil
}

// analyzeSliderBehavior 分析滑动行为，返回风险评分 (0-1)
func analyzeSliderBehavior(answer SliderAnswer) float64 {
	score := 0.0
	
	// 1. 滑动时间过短 (< 200ms) 可能是机器人
	if answer.Duration < 200 {
		score += 0.3
	}
	
	// 2. 滑动时间过长 (> 10s) 可能是异常
	if answer.Duration > 10000 {
		score += 0.1
	}
	
	// 3. 轨迹点太少可能是机器人
	if len(answer.Trail) < 5 {
		score += 0.3
	}
	
	// 4. 轨迹完全线性 (没有抖动) 可能是机器人
	if len(answer.Trail) > 2 {
		isLinear := true
		for i := 1; i < len(answer.Trail)-1; i++ {
			// 检查是否有微小抖动
			if answer.Trail[i] != answer.Trail[i-1] && answer.Trail[i] != answer.Trail[i+1] {
				isLinear = false
				break
			}
		}
		if isLinear {
			score += 0.2
		}
	}
	
	// 5. 轨迹单调递增 (没有回退) 可能是机器人
	if len(answer.Trail) > 3 {
		hasBacktrack := false
		for i := 1; i < len(answer.Trail); i++ {
			if answer.Trail[i] < answer.Trail[i-1] {
				hasBacktrack = true
				break
			}
		}
		if !hasBacktrack {
			score += 0.1
		}
	}
	
	return score
}

// generateSliderImages 生成背景图和拼图块
func generateSliderImages(width, height, pieceSize, targetX, targetY int) (*image.RGBA, *image.RGBA) {
	// 生成渐变背景
	background := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// 渐变色背景
			r := uint8(100 + x*50/width)
			g := uint8(150 + y*50/height)
			b := uint8(200 - x*30/width)
			background.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}
	
	// 添加一些随机图案
	addRandomPatterns(background)
	
	// 创建拼图块 (带凸起的形状)
	piece := image.NewRGBA(image.Rect(0, 0, pieceSize+10, pieceSize))
	
	// 从背景中提取拼图块区域
	for y := 0; y < pieceSize; y++ {
		for x := 0; x < pieceSize; x++ {
			if isInPieceShape(x, y, pieceSize) {
				// 复制背景像素到拼图块
				piece.Set(x, y, background.At(targetX+x, targetY+y))
				// 在背景上留下缺口 (半透明灰色)
				background.Set(targetX+x, targetY+y, color.RGBA{100, 100, 100, 180})
			}
		}
	}
	
	// 给拼图块添加边框
	addPieceBorder(piece, pieceSize)
	
	return background, piece
}

// isInPieceShape 判断点是否在拼图形状内 (带凸起)
func isInPieceShape(x, y, size int) bool {
	// 基本矩形
	if x >= 0 && x < size && y >= 0 && y < size {
		return true
	}
	// 右侧凸起 (半圆)
	cx := size
	cy := size / 2
	r := size / 5
	if (x-cx)*(x-cx)+(y-cy)*(y-cy) <= r*r {
		return true
	}
	return false
}

// addPieceBorder 给拼图块添加边框
func addPieceBorder(piece *image.RGBA, size int) {
	borderColor := color.RGBA{255, 255, 255, 200}
	bounds := piece.Bounds()
	
	for y := 0; y < bounds.Max.Y; y++ {
		for x := 0; x < bounds.Max.X; x++ {
			if isInPieceShape(x, y, size) {
				// 检查是否是边缘
				if !isInPieceShape(x-1, y, size) || !isInPieceShape(x+1, y, size) ||
					!isInPieceShape(x, y-1, size) || !isInPieceShape(x, y+1, size) {
					piece.Set(x, y, borderColor)
				}
			}
		}
	}
}

// addRandomPatterns 添加随机图案
func addRandomPatterns(img *image.RGBA) {
	bounds := img.Bounds()
	
	// 添加一些随机圆形
	for i := 0; i < 5; i++ {
		cx, _ := rand.Int(rand.Reader, big.NewInt(int64(bounds.Max.X)))
		cy, _ := rand.Int(rand.Reader, big.NewInt(int64(bounds.Max.Y)))
		r, _ := rand.Int(rand.Reader, big.NewInt(30))
		radius := int(r.Int64()) + 10
		
		circleColor := randomColor()
		for y := -radius; y <= radius; y++ {
			for x := -radius; x <= radius; x++ {
				if x*x+y*y <= radius*radius {
					px := int(cx.Int64()) + x
					py := int(cy.Int64()) + y
					if px >= 0 && px < bounds.Max.X && py >= 0 && py < bounds.Max.Y {
						// 混合颜色
						orig := img.At(px, py).(color.RGBA)
						blended := blendColors(orig, circleColor, 0.3)
						img.Set(px, py, blended)
					}
				}
			}
		}
	}
}

// blendColors 混合两个颜色
func blendColors(c1, c2 color.RGBA, ratio float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(c1.R)*(1-ratio) + float64(c2.R)*ratio),
		G: uint8(float64(c1.G)*(1-ratio) + float64(c2.G)*ratio),
		B: uint8(float64(c1.B)*(1-ratio) + float64(c2.B)*ratio),
		A: 255,
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
