// Package ecaptcha 提供自研人机验证 SDK
// 支持多种验证方式：图片验证码、滑动拼图、行为分析无感验证
package ecaptcha

import (
	"context"
	"errors"
	"time"
)

// CaptchaType 验证类型
type CaptchaType string

const (
	TypeImage    CaptchaType = "image"    // 图片验证码
	TypeSlider   CaptchaType = "slider"   // 滑动拼图
	TypeBehavior CaptchaType = "behavior" // 行为分析无感验证
)

// 错误定义
var (
	ErrCaptchaNotFound = errors.New("验证码不存在或已过期")
	ErrCaptchaInvalid  = errors.New("验证码错误")
	ErrCaptchaExpired  = errors.New("验证码已过期")
	ErrTooManyAttempts = errors.New("验证次数过多，请重新获取")
	ErrRiskDetected    = errors.New("检测到异常行为")
	ErrInvalidToken    = errors.New("无效的验证 token")
)

// Challenge 验证挑战 (发给前端)
type Challenge struct {
	ID        string      `json:"id"`         // 挑战 ID
	Type      CaptchaType `json:"type"`       // 验证类型
	Data      interface{} `json:"data"`       // 验证数据 (图片/拼图等)
	ExpiresAt int64       `json:"expires_at"` // 过期时间戳
}

// VerifyRequest 验证请求 (前端提交)
type VerifyRequest struct {
	ID     string      `json:"id"`     // 挑战 ID
	Type   CaptchaType `json:"type"`   // 验证类型
	Answer interface{} `json:"answer"` // 用户答案
	// 行为数据 (用于无感验证)
	Behavior *BehaviorData `json:"behavior,omitempty"`
}

// BehaviorData 用户行为数据
type BehaviorData struct {
	MouseTrack   []Point `json:"mouse_track,omitempty"`   // 鼠标轨迹
	KeyTiming    []int64 `json:"key_timing,omitempty"`    // 按键时间间隔
	TouchTrack   []Point `json:"touch_track,omitempty"`   // 触摸轨迹
	DeviceInfo   string  `json:"device_info,omitempty"`   // 设备信息
	ScreenSize   string  `json:"screen_size,omitempty"`   // 屏幕尺寸
	TimeOnPage   int64   `json:"time_on_page,omitempty"`  // 页面停留时间(ms)
	ScrollDepth  float64 `json:"scroll_depth,omitempty"`  // 滚动深度
	FocusChanges int     `json:"focus_changes,omitempty"` // 焦点切换次数
}

// Point 坐标点
type Point struct {
	X int   `json:"x"`
	Y int   `json:"y"`
	T int64 `json:"t,omitempty"` // 时间戳
}

// VerifyResult 验证结果
type VerifyResult struct {
	Success   bool    `json:"success"`             // 是否通过
	Token     string  `json:"token,omitempty"`     // 验证通过后的 token (用于后续请求)
	Score     float64 `json:"score,omitempty"`     // 风险评分 (0-1, 越低越安全)
	Message   string  `json:"message,omitempty"`   // 提示信息
	ExpiresAt int64   `json:"expires_at,omitempty"` // token 过期时间
}

// Config 配置
type Config struct {
	// 通用配置
	Expire       time.Duration `json:",default=5m"`  // 验证码过期时间
	MaxAttempts  int           `json:",default=3"`   // 最大尝试次数
	TokenExpire  time.Duration `json:",default=10m"` // 验证 token 过期时间
	
	// 图片验证码配置
	ImageWidth   int `json:",default=150"` // 图片宽度
	ImageHeight  int `json:",default=50"`  // 图片高度
	ImageLength  int `json:",default=4"`   // 验证码长度
	
	// 滑动拼图配置
	SliderWidth  int `json:",default=300"` // 拼图宽度
	SliderHeight int `json:",default=150"` // 拼图高度
	PieceSize    int `json:",default=50"`  // 拼图块大小
	Tolerance    int `json:",default=5"`   // 容差像素
	
	// 行为分析配置
	BehaviorThreshold float64 `json:",default=0.7"` // 风险阈值 (超过则需要二次验证)
}

// Provider 验证提供者接口
type Provider interface {
	// Generate 生成验证挑战
	Generate(ctx context.Context) (*Challenge, error)
	
	// Verify 验证用户答案
	Verify(ctx context.Context, req *VerifyRequest) (*VerifyResult, error)
	
	// Type 返回验证类型
	Type() CaptchaType
}

// Store 存储接口
type Store interface {
	// Set 存储验证数据
	Set(ctx context.Context, id string, data []byte, expire time.Duration) error
	
	// Get 获取验证数据
	Get(ctx context.Context, id string) ([]byte, error)
	
	// Delete 删除验证数据
	Delete(ctx context.Context, id string) error
	
	// IncrAttempts 增加尝试次数，返回当前次数
	IncrAttempts(ctx context.Context, id string) (int, error)
}

// ECaptcha 验证服务
type ECaptcha struct {
	config    Config
	store     Store
	providers map[CaptchaType]Provider
}

// New 创建验证服务
func New(config Config, store Store) *ECaptcha {
	return &ECaptcha{
		config:    config,
		store:     store,
		providers: make(map[CaptchaType]Provider),
	}
}

// RegisterProvider 注册验证提供者
func (e *ECaptcha) RegisterProvider(p Provider) {
	e.providers[p.Type()] = p
}

// Generate 生成验证挑战
func (e *ECaptcha) Generate(ctx context.Context, captchaType CaptchaType) (*Challenge, error) {
	provider, ok := e.providers[captchaType]
	if !ok {
		return nil, errors.New("不支持的验证类型: " + string(captchaType))
	}
	return provider.Generate(ctx)
}

// Verify 验证用户答案
func (e *ECaptcha) Verify(ctx context.Context, req *VerifyRequest) (*VerifyResult, error) {
	provider, ok := e.providers[req.Type]
	if !ok {
		return nil, errors.New("不支持的验证类型: " + string(req.Type))
	}
	return provider.Verify(ctx, req)
}

// ValidateToken 验证 token 是否有效
func (e *ECaptcha) ValidateToken(ctx context.Context, token string) (bool, error) {
	if token == "" {
		return false, ErrInvalidToken
	}
	
	data, err := e.store.Get(ctx, "token:"+token)
	if err != nil {
		return false, ErrInvalidToken
	}
	
	// token 存在且未过期
	return len(data) > 0, nil
}

// GetConfig 获取配置
func (e *ECaptcha) GetConfig() Config {
	return e.config
}
