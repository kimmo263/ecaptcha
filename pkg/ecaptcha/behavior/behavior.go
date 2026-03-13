// Package behavior 提供基于行为分析的无感验证
package behavior

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"time"

	"github.com/google/uuid"

	"global-track/pkg/ecaptcha"
)

// BehaviorData 行为数据 (发给前端收集)
type BehaviorData struct {
	CollectItems []string `json:"collect_items"` // 需要收集的行为项
	Timeout      int      `json:"timeout"`       // 收集超时时间 (ms)
}

// BehaviorAnswer 行为验证答案 (前端提交)
type BehaviorAnswer struct {
	MouseTrack   []ecaptcha.Point `json:"mouse_track"`   // 鼠标轨迹
	KeyTiming    []int64          `json:"key_timing"`    // 按键时间间隔
	TouchTrack   []ecaptcha.Point `json:"touch_track"`   // 触摸轨迹
	DeviceInfo   string           `json:"device_info"`   // 设备信息
	ScreenSize   string           `json:"screen_size"`   // 屏幕尺寸
	TimeOnPage   int64            `json:"time_on_page"`  // 页面停留时间(ms)
	ScrollDepth  float64          `json:"scroll_depth"`  // 滚动深度
	FocusChanges int              `json:"focus_changes"` // 焦点切换次数
	Timestamp    int64            `json:"timestamp"`     // 提交时间戳
	Signature    string           `json:"signature"`     // 数据签名
}

// storedData 存储的验证数据
type storedData struct {
	Secret    string `json:"secret"`     // 签名密钥
	ExpiresAt int64  `json:"expires_at"`
}

// Provider 行为分析提供者
type Provider struct {
	config    ecaptcha.Config
	store     ecaptcha.Store
	secretKey string // 用于生成签名的密钥
}

// New 创建行为分析提供者
func New(config ecaptcha.Config, store ecaptcha.Store, secretKey string) *Provider {
	if secretKey == "" {
		secretKey = "ecaptcha-behavior-secret-2026"
	}
	return &Provider{
		config:    config,
		store:     store,
		secretKey: secretKey,
	}
}

// Type 返回验证类型
func (p *Provider) Type() ecaptcha.CaptchaType {
	return ecaptcha.TypeBehavior
}

// Generate 生成行为验证挑战
func (p *Provider) Generate(ctx context.Context) (*ecaptcha.Challenge, error) {
	// 生成挑战 ID 和签名密钥
	id := uuid.New().String()
	secret := uuid.New().String()
	expiresAt := time.Now().Add(p.config.Expire).Unix()
	
	// 存储验证数据
	data, _ := json.Marshal(&storedData{
		Secret:    secret,
		ExpiresAt: expiresAt,
	})
	if err := p.store.Set(ctx, id, data, p.config.Expire); err != nil {
		return nil, err
	}
	
	return &ecaptcha.Challenge{
		ID:   id,
		Type: ecaptcha.TypeBehavior,
		Data: &BehaviorData{
			CollectItems: []string{
				"mouse_track",
				"key_timing",
				"touch_track",
				"device_info",
				"screen_size",
				"time_on_page",
				"scroll_depth",
				"focus_changes",
			},
			Timeout: 30000, // 30 秒收集时间
		},
		ExpiresAt: expiresAt,
	}, nil
}

// Verify 验证行为数据
func (p *Provider) Verify(ctx context.Context, req *ecaptcha.VerifyRequest) (*ecaptcha.VerifyResult, error) {
	// 获取存储的验证数据
	data, err := p.store.Get(ctx, req.ID)
	if err != nil {
		return &ecaptcha.VerifyResult{
			Success: false,
			Message: "验证会话不存在或已过期",
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
			Message: "验证会话已过期",
		}, nil
	}
	
	// 解析行为数据
	var answer BehaviorAnswer
	answerBytes, _ := json.Marshal(req.Answer)
	if err := json.Unmarshal(answerBytes, &answer); err != nil {
		return &ecaptcha.VerifyResult{
			Success: false,
			Message: "数据格式错误",
		}, nil
	}
	
	// 分析行为，计算风险评分
	score := p.analyzeBehavior(&answer)
	
	// 删除验证数据
	p.store.Delete(ctx, req.ID)
	
	// 根据阈值判断是否通过
	if score > p.config.BehaviorThreshold {
		return &ecaptcha.VerifyResult{
			Success: false,
			Score:   score,
			Message: "检测到异常行为，请完成额外验证",
		}, nil
	}
	
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

// analyzeBehavior 分析行为数据，返回风险评分 (0-1, 越高风险越大)
func (p *Provider) analyzeBehavior(answer *BehaviorAnswer) float64 {
	var totalScore float64
	var factors int
	
	// 1. 鼠标轨迹分析
	if len(answer.MouseTrack) > 0 {
		mouseScore := analyzeMouseTrack(answer.MouseTrack)
		totalScore += mouseScore
		factors++
	}
	
	// 2. 按键时间分析
	if len(answer.KeyTiming) > 0 {
		keyScore := analyzeKeyTiming(answer.KeyTiming)
		totalScore += keyScore
		factors++
	}
	
	// 3. 页面停留时间分析
	timeScore := analyzeTimeOnPage(answer.TimeOnPage)
	totalScore += timeScore
	factors++
	
	// 4. 设备信息分析
	deviceScore := analyzeDeviceInfo(answer.DeviceInfo, answer.ScreenSize)
	totalScore += deviceScore
	factors++
	
	// 5. 滚动行为分析
	scrollScore := analyzeScrollBehavior(answer.ScrollDepth)
	totalScore += scrollScore
	factors++
	
	// 6. 焦点切换分析
	focusScore := analyzeFocusChanges(answer.FocusChanges)
	totalScore += focusScore
	factors++
	
	if factors == 0 {
		return 1.0 // 没有任何行为数据，高风险
	}
	
	return totalScore / float64(factors)
}

// analyzeMouseTrack 分析鼠标轨迹
func analyzeMouseTrack(track []ecaptcha.Point) float64 {
	if len(track) < 3 {
		return 0.8 // 轨迹点太少，可疑
	}
	
	var score float64
	
	// 检查轨迹是否过于线性
	linearCount := 0
	for i := 2; i < len(track); i++ {
		// 计算三点是否共线
		if isCollinear(track[i-2], track[i-1], track[i]) {
			linearCount++
		}
	}
	linearRatio := float64(linearCount) / float64(len(track)-2)
	if linearRatio > 0.8 {
		score += 0.3 // 过于线性
	}
	
	// 检查速度是否恒定
	if len(track) > 2 && track[0].T > 0 {
		speeds := make([]float64, 0)
		for i := 1; i < len(track); i++ {
			if track[i].T > track[i-1].T {
				dx := float64(track[i].X - track[i-1].X)
				dy := float64(track[i].Y - track[i-1].Y)
				dt := float64(track[i].T - track[i-1].T)
				speed := math.Sqrt(dx*dx+dy*dy) / dt
				speeds = append(speeds, speed)
			}
		}
		
		if len(speeds) > 2 {
			// 计算速度标准差
			stdDev := calculateStdDev(speeds)
			if stdDev < 0.1 {
				score += 0.2 // 速度过于恒定
			}
		}
	}
	
	return score
}

// analyzeKeyTiming 分析按键时间间隔
func analyzeKeyTiming(timing []int64) float64 {
	if len(timing) < 3 {
		return 0.3 // 数据太少
	}
	
	// 转换为 float64
	floatTiming := make([]float64, len(timing))
	for i, t := range timing {
		floatTiming[i] = float64(t)
	}
	
	// 计算标准差
	stdDev := calculateStdDev(floatTiming)
	
	// 人类打字通常有较大的时间间隔变化
	if stdDev < 10 {
		return 0.5 // 间隔过于均匀，可疑
	}
	
	// 检查是否有异常快的输入
	for _, t := range timing {
		if t < 20 { // 小于 20ms
			return 0.4 // 输入过快
		}
	}
	
	return 0.0
}

// analyzeTimeOnPage 分析页面停留时间
func analyzeTimeOnPage(timeMs int64) float64 {
	// 停留时间过短 (< 1秒)
	if timeMs < 1000 {
		return 0.8
	}
	// 停留时间过长 (> 5分钟) 可能是挂机
	if timeMs > 300000 {
		return 0.3
	}
	return 0.0
}

// analyzeDeviceInfo 分析设备信息
func analyzeDeviceInfo(deviceInfo, screenSize string) float64 {
	// 没有设备信息
	if deviceInfo == "" {
		return 0.5
	}
	
	// 检查是否是已知的自动化工具 User-Agent
	suspiciousKeywords := []string{
		"headless", "phantom", "selenium", "puppeteer", "playwright",
		"webdriver", "bot", "crawler", "spider",
	}
	
	for _, keyword := range suspiciousKeywords {
		if containsIgnoreCase(deviceInfo, keyword) {
			return 0.9
		}
	}
	
	// 检查屏幕尺寸是否异常
	if screenSize == "" || screenSize == "0x0" {
		return 0.4
	}
	
	return 0.0
}

// analyzeScrollBehavior 分析滚动行为
func analyzeScrollBehavior(scrollDepth float64) float64 {
	// 没有滚动
	if scrollDepth == 0 {
		return 0.2
	}
	// 滚动深度异常 (> 100%)
	if scrollDepth > 1.0 {
		return 0.1
	}
	return 0.0
}

// analyzeFocusChanges 分析焦点切换
func analyzeFocusChanges(changes int) float64 {
	// 没有焦点切换可能是自动化
	if changes == 0 {
		return 0.2
	}
	// 焦点切换过多可能是异常
	if changes > 50 {
		return 0.3
	}
	return 0.0
}

// isCollinear 判断三点是否共线
func isCollinear(p1, p2, p3 ecaptcha.Point) bool {
	// 使用叉积判断
	cross := (p2.X-p1.X)*(p3.Y-p1.Y) - (p3.X-p1.X)*(p2.Y-p1.Y)
	return abs(cross) < 5 // 允许小误差
}

// calculateStdDev 计算标准差
func calculateStdDev(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	// 计算平均值
	var sum float64
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))
	
	// 计算方差
	var variance float64
	for _, v := range values {
		variance += (v - mean) * (v - mean)
	}
	variance /= float64(len(values))
	
	return math.Sqrt(variance)
}

// containsIgnoreCase 忽略大小写检查字符串包含
func containsIgnoreCase(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalIgnoreCase(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

// equalIgnoreCase 忽略大小写比较字符串
func equalIgnoreCase(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 32
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}

// generateSignature 生成数据签名
func generateSignature(data string, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
