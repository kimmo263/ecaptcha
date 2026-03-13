# ECaptcha - 自研人机验证 SDK

ECaptcha 是一个独立的人机验证 SDK，支持多种验证方式，可用于内部项目集成。

## 特性

- **图片验证码** - 经典的图片数字/字母验证码
- **滑动拼图** - 类似极验的滑动拼图验证，带行为分析
- **无感验证** - 基于用户行为分析的无感验证

## 安装

```go
import "global-track/pkg/ecaptcha"
```

## 快速开始

### 1. 初始化

```go
import (
    "github.com/redis/go-redis/v9"
    "global-track/pkg/ecaptcha"
    "global-track/pkg/ecaptcha/image"
    "global-track/pkg/ecaptcha/slider"
    "global-track/pkg/ecaptcha/behavior"
)

// 创建 Redis 存储
redisClient := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})
store := ecaptcha.NewRedisStore(redisClient)

// 创建验证服务
config := ecaptcha.Config{
    Expire:      5 * time.Minute,
    MaxAttempts: 3,
    TokenExpire: 10 * time.Minute,
    // 图片验证码配置
    ImageWidth:  150,
    ImageHeight: 50,
    ImageLength: 4,
    // 滑动拼图配置
    SliderWidth:  300,
    SliderHeight: 150,
    PieceSize:    50,
    Tolerance:    5,
    // 行为分析配置
    BehaviorThreshold: 0.7,
}

captcha := ecaptcha.New(config, store)

// 注册验证提供者
captcha.RegisterProvider(image.New(config, store))
captcha.RegisterProvider(slider.New(config, store))
captcha.RegisterProvider(behavior.New(config, store, "your-secret-key"))
```

### 2. HTTP 接口

```go
// 创建 HTTP Handler
handler := ecaptcha.NewHandler(captcha)

// 注册路由
mux := http.NewServeMux()
handler.RegisterRoutes(mux, "/ecaptcha")

// 或手动注册
// POST /ecaptcha/generate - 生成验证码
// POST /ecaptcha/verify   - 验证答案
// POST /ecaptcha/validate - 验证 token
// GET  /ecaptcha/config   - 获取配置
```

### 3. 在业务代码中使用

```go
// 生成验证码
challenge, err := captcha.Generate(ctx, ecaptcha.TypeImage)
// 返回给前端: challenge.ID, challenge.Data

// 验证用户答案
result, err := captcha.Verify(ctx, &ecaptcha.VerifyRequest{
    ID:     "challenge-id",
    Type:   ecaptcha.TypeImage,
    Answer: map[string]string{"code": "AB12"},
})

if result.Success {
    // 验证成功，result.Token 可用于后续请求
}

// 验证 token (在敏感操作前)
valid, err := captcha.ValidateToken(ctx, token)
```

## API 接口

### POST /ecaptcha/generate

生成验证挑战。

**请求:**
```json
{
    "type": "image"  // image, slider, behavior
}
```

**响应:**
```json
{
    "code": 200,
    "message": "success",
    "data": {
        "id": "uuid",
        "type": "image",
        "data": {
            "image": "data:image/png;base64,..."
        },
        "expires_at": 1234567890
    }
}
```

### POST /ecaptcha/verify

验证用户答案。

**请求 (图片验证码):**
```json
{
    "id": "challenge-id",
    "type": "image",
    "answer": {
        "code": "AB12"
    }
}
```

**请求 (滑动拼图):**
```json
{
    "id": "challenge-id",
    "type": "slider",
    "answer": {
        "x": 150,
        "duration": 800,
        "trail": [0, 10, 30, 60, 100, 130, 150]
    }
}
```

**请求 (行为验证):**
```json
{
    "id": "challenge-id",
    "type": "behavior",
    "answer": {
        "mouse_track": [{"x": 0, "y": 0, "t": 0}, ...],
        "key_timing": [100, 80, 120, ...],
        "time_on_page": 5000,
        "device_info": "Mozilla/5.0...",
        "screen_size": "1920x1080"
    }
}
```

**响应:**
```json
{
    "code": 200,
    "message": "验证成功",
    "data": {
        "success": true,
        "token": "uuid",
        "score": 0.1,
        "expires_at": 1234567890
    }
}
```

### POST /ecaptcha/validate

验证 token 是否有效。

**请求:**
```json
{
    "token": "uuid"
}
```

**响应:**
```json
{
    "code": 200,
    "message": "token 有效",
    "data": {
        "success": true
    }
}
```

## 前端集成

### 图片验证码

```html
<img id="captcha-image" src="" />
<input id="captcha-input" type="text" placeholder="请输入验证码" />
<button onclick="refreshCaptcha()">刷新</button>

<script>
let captchaId = '';

async function refreshCaptcha() {
    const res = await fetch('/ecaptcha/generate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ type: 'image' })
    });
    const data = await res.json();
    captchaId = data.data.id;
    document.getElementById('captcha-image').src = data.data.data.image;
}

async function verifyCaptcha() {
    const code = document.getElementById('captcha-input').value;
    const res = await fetch('/ecaptcha/verify', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            id: captchaId,
            type: 'image',
            answer: { code }
        })
    });
    const data = await res.json();
    if (data.data.success) {
        // 使用 data.data.token 进行后续操作
    }
}
</script>
```

### 滑动拼图

```html
<div id="slider-container">
    <img id="slider-bg" src="" />
    <img id="slider-piece" src="" />
    <div id="slider-bar"></div>
</div>

<script>
let captchaId = '';
let startX = 0;
let trail = [];

async function initSlider() {
    const res = await fetch('/ecaptcha/generate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ type: 'slider' })
    });
    const data = await res.json();
    captchaId = data.data.id;
    document.getElementById('slider-bg').src = data.data.data.background;
    document.getElementById('slider-piece').src = data.data.data.piece;
    // 设置拼图块 Y 位置
    document.getElementById('slider-piece').style.top = data.data.data.piece_y + 'px';
}

// 监听滑动事件，记录轨迹，提交验证...
</script>
```

## 行为分析指标

无感验证会分析以下用户行为：

| 指标 | 说明 | 风险判断 |
|------|------|----------|
| 鼠标轨迹 | 鼠标移动路径 | 过于线性、速度恒定 |
| 按键时间 | 键盘输入间隔 | 间隔过于均匀、输入过快 |
| 页面停留 | 在页面的时间 | 过短 (<1s) 或过长 (>5min) |
| 设备信息 | User-Agent | 包含自动化工具特征 |
| 滚动深度 | 页面滚动比例 | 没有滚动行为 |
| 焦点切换 | 窗口焦点变化 | 没有焦点切换 |

风险评分 0-1，超过阈值 (默认 0.7) 需要二次验证。

## 配置说明

```go
type Config struct {
    // 通用配置
    Expire       time.Duration // 验证码过期时间，默认 5 分钟
    MaxAttempts  int           // 最大尝试次数，默认 3 次
    TokenExpire  time.Duration // 验证 token 过期时间，默认 10 分钟
    
    // 图片验证码
    ImageWidth   int // 图片宽度，默认 150
    ImageHeight  int // 图片高度，默认 50
    ImageLength  int // 验证码长度，默认 4
    
    // 滑动拼图
    SliderWidth  int // 拼图宽度，默认 300
    SliderHeight int // 拼图高度，默认 150
    PieceSize    int // 拼图块大小，默认 50
    Tolerance    int // 容差像素，默认 5
    
    // 行为分析
    BehaviorThreshold float64 // 风险阈值，默认 0.7
}
```

## 与 Turnstile 对比

| 特性 | ECaptcha | Turnstile |
|------|----------|-----------|
| 部署方式 | 自托管 | 云服务 |
| 费用 | 免费 | 免费 |
| 数据隐私 | 完全自控 | 第三方 |
| 定制化 | 高 | 低 |
| 维护成本 | 需自行维护 | 无需维护 |
| 安全性 | 取决于实现 | 专业团队 |

## License

Internal use only.
