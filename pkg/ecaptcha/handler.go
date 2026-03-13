package ecaptcha

import (
	"encoding/json"
	"net/http"
)

// Handler HTTP 处理器
type Handler struct {
	captcha *ECaptcha
}

// NewHandler 创建 HTTP 处理器
func NewHandler(captcha *ECaptcha) *Handler {
	return &Handler{captcha: captcha}
}

// GenerateRequest 生成验证码请求
type GenerateRequest struct {
	Type CaptchaType `json:"type"` // image, slider, behavior
}

// GenerateResponse 生成验证码响应
type GenerateResponse struct {
	Code    int        `json:"code"`
	Message string     `json:"message"`
	Data    *Challenge `json:"data,omitempty"`
}

// VerifyResponse 验证响应
type VerifyResponse struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Data    *VerifyResult `json:"data,omitempty"`
}

// ConfigResponse 配置响应 (供前端获取 SiteKey 等)
type ConfigResponse struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// HandleGenerate 处理生成验证码请求
// POST /ecaptcha/generate
func (h *Handler) HandleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, &GenerateResponse{
			Code:    405,
			Message: "Method not allowed",
		})
		return
	}

	var req GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, &GenerateResponse{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	// 默认使用图片验证码
	if req.Type == "" {
		req.Type = TypeImage
	}

	challenge, err := h.captcha.Generate(r.Context(), req.Type)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, &GenerateResponse{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, &GenerateResponse{
		Code:    200,
		Message: "success",
		Data:    challenge,
	})
}

// HandleVerify 处理验证请求
// POST /ecaptcha/verify
func (h *Handler) HandleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, &VerifyResponse{
			Code:    405,
			Message: "Method not allowed",
		})
		return
	}

	var req VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, &VerifyResponse{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	result, err := h.captcha.Verify(r.Context(), &req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, &VerifyResponse{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	code := 200
	if !result.Success {
		code = 400
	}

	writeJSON(w, http.StatusOK, &VerifyResponse{
		Code:    code,
		Message: result.Message,
		Data:    result,
	})
}

// HandleValidateToken 验证 token 是否有效
// POST /ecaptcha/validate
func (h *Handler) HandleValidateToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, &VerifyResponse{
			Code:    405,
			Message: "Method not allowed",
		})
		return
	}

	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, &VerifyResponse{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	valid, err := h.captcha.ValidateToken(r.Context(), req.Token)
	if err != nil || !valid {
		writeJSON(w, http.StatusOK, &VerifyResponse{
			Code:    400,
			Message: "token 无效或已过期",
			Data:    &VerifyResult{Success: false},
		})
		return
	}

	writeJSON(w, http.StatusOK, &VerifyResponse{
		Code:    200,
		Message: "token 有效",
		Data:    &VerifyResult{Success: true},
	})
}

// HandleConfig 获取验证配置 (供前端使用)
// GET /ecaptcha/config
func (h *Handler) HandleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, &ConfigResponse{
			Code:    405,
			Message: "Method not allowed",
		})
		return
	}

	config := h.captcha.GetConfig()
	writeJSON(w, http.StatusOK, &ConfigResponse{
		Code:    200,
		Message: "success",
		Data: map[string]interface{}{
			"types": []string{
				string(TypeImage),
				string(TypeSlider),
				string(TypeBehavior),
			},
			"image": map[string]interface{}{
				"width":  config.ImageWidth,
				"height": config.ImageHeight,
			},
			"slider": map[string]interface{}{
				"width":      config.SliderWidth,
				"height":     config.SliderHeight,
				"piece_size": config.PieceSize,
			},
			"expire_seconds": int(config.Expire.Seconds()),
		},
	})
}

// RegisterRoutes 注册路由 (用于 go-zero 或标准 http)
func (h *Handler) RegisterRoutes(mux *http.ServeMux, prefix string) {
	mux.HandleFunc(prefix+"/generate", h.HandleGenerate)
	mux.HandleFunc(prefix+"/verify", h.HandleVerify)
	mux.HandleFunc(prefix+"/validate", h.HandleValidateToken)
	mux.HandleFunc(prefix+"/config", h.HandleConfig)
}

// writeJSON 写入 JSON 响应
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
