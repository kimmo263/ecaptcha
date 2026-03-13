<div align="center">

# 🛡️ ECaptcha

**轻量级、可自托管的人机验证解决方案**

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

</div>

---

## 📖 简介

ECaptcha 是一个**开源、可自托管**的人机验证 SDK，提供完整的前后端解决方案。无需依赖第三方云服务，数据完全自主可控，适合对数据隐私有要求的企业使用。

## ✨ 特性

| 特性 | 说明 |
|------|------|
| 🖼️ **图片验证码** | 经典的图片数字/字母验证码，支持自定义字体和干扰线 |
| 🧩 **滑动拼图** | 类似极验的滑动拼图验证，内置行为轨迹分析 |
| 🤖 **无感验证** | 基于用户行为分析的智能验证，对正常用户零打扰 |
| 🎯 **智能降级** | 先尝试无感验证，风险过高时自动降级到滑动拼图 |
| 🌍 **多语言** | 支持中文、英文，可扩展其他语言 |
| 🎨 **主题定制** | 支持亮色/暗色主题，可自定义样式 |
| 📦 **开箱即用** | 提供完整的前端 SDK 和后端服务 |

## 🏗️ 架构

```
┌─────────────────────────────────────────────────────────────┐
│                        前端应用                              │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              ECaptcha Frontend SDK                   │   │
│  │   • CDN 引入 / NPM 安装                              │   │
│  │   • 支持 React / Vue / 原生 JS                       │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      ECaptcha Server                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ Image 验证   │  │ Slider 验证  │  │ Behavior 验证│      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│                              │                               │
│                      ┌───────┴───────┐                      │
│                      │  Redis Store  │                      │
│                      └───────────────┘                      │
└─────────────────────────────────────────────────────────────┘
```

## 📁 项目结构

```
ecaptcha/
├── ecaptcha.go        # 核心验证服务
├── handler.go         # HTTP 处理器
├── store.go           # Redis 存储实现
├── image/             # 图片验证码提供者
├── slider/            # 滑动拼图提供者
├── behavior/          # 行为分析提供者
├── sdk/               # 前端 SDK
│   ├── ecaptcha.min.js
│   ├── ecaptcha.d.ts
│   └── README.md
└── example/           # 完整示例
    └── main.go
```

## 🚀 快速开始

### 环境要求

- Go 1.21+
- Redis 6.0+

### 后端集成 (Go)

#### 1. 安装

```bash
go get github.com/kimmo263/ecaptcha
```

#### 2. 初始化服务

```go
package main

import (
    "net/http"
    "time"

    "github.com/redis/go-redis/v9"
    "github.com/kimmo263/ecaptcha"
    "github.com/kimmo263/ecaptcha/image"
    "github.com/kimmo263/ecaptcha/slider"
    "github.com/kimmo263/ecaptcha/behavior"
)

func main() {
    // 1. 创建 Redis 存储
    redisClient := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    store := ecaptcha.NewRedisStore(redisClient)

    // 2. 配置验证服务
    config := ecaptcha.Config{
        Expire:            5 * time.Minute,  // 验证码过期时间
        MaxAttempts:       3,                // 最大尝试次数
        TokenExpire:       10 * time.Minute, // Token 过期时间
        BehaviorThreshold: 0.7,              // 行为分析阈值
    }

    // 3. 创建验证服务并注册提供者
    captcha := ecaptcha.New(config, store)
    captcha.RegisterProvider(image.New(config, store))
    captcha.RegisterProvider(slider.New(config, store))
    captcha.RegisterProvider(behavior.New(config, store, "your-secret-key"))

    // 4. 注册 HTTP 路由
    handler := ecaptcha.NewHandler(captcha)
    mux := http.NewServeMux()
    handler.RegisterRoutes(mux, "/ecaptcha")

    // 5. 启动服务
    http.ListenAndServe(":8080", mux)
}
```

#### 3. 在业务代码中验证 Token

```go
// 在敏感操作前验证 captcha token
func SendSMSHandler(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Phone        string `json:"phone"`
        CaptchaToken string `json:"captcha_token"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    // 验证 token
    valid, err := captcha.ValidateToken(r.Context(), req.CaptchaToken)
    if err != nil || !valid {
        http.Error(w, "验证码无效", http.StatusBadRequest)
        return
    }

    // 继续业务逻辑...
}
```

### 前端集成

> 详细文档请参考 [前端 SDK 文档](./sdk/README.md)

#### 方式一：CDN 引入

```html
<script src="https://your-domain/ecaptcha/sdk/ecaptcha.min.js"></script>
<script>
  // 初始化
  ECaptcha.init({ 
    server: 'https://api.example.com',
    lang: 'zh-CN',
    theme: 'light'
  });
  
  // 使用智能验证
  document.getElementById('submit-btn').onclick = async () => {
    try {
      const token = await ECaptcha.smart();
      // 将 token 发送给后端
      await fetch('/api/submit', {
        method: 'POST',
        body: JSON.stringify({ captcha_token: token })
      });
    } catch (e) {
      console.log('用户取消验证');
    }
  };
</script>
```

#### 方式二：NPM 安装

```bash
npm install @ectrack/ecaptcha
```

```typescript
import ECaptcha from '@ectrack/ecaptcha';

// 初始化 (在应用入口调用一次)
ECaptcha.init({ server: 'https://api.example.com' });

// 图片验证码
const token = await ECaptcha.verify('image');

// 滑动拼图
const token = await ECaptcha.verify('slider');

// 智能验证 (推荐) - 先无感验证，失败自动降级
const token = await ECaptcha.smart();
```

## 📡 API 接口

### POST /ecaptcha/generate

生成验证挑战。

**请求:**
```json
{
    "type": "image"  // image | slider | behavior
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
    "answer": { "code": "AB12" }
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

验证 token 是否有效（后端调用）。

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
    "data": { "success": true }
}
```

## 🔧 配置参数

```go
type Config struct {
    // 通用配置
    Expire       time.Duration // 验证码过期时间，默认 5 分钟
    MaxAttempts  int           // 最大尝试次数，默认 3 次
    TokenExpire  time.Duration // Token 过期时间，默认 10 分钟
    
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

## 🧠 行为分析指标

无感验证会分析以下用户行为：

| 指标 | 说明 | 风险判断 |
|------|------|----------|
| 鼠标轨迹 | 鼠标移动路径 | 过于线性、速度恒定 |
| 按键时间 | 键盘输入间隔 | 间隔过于均匀、输入过快 |
| 页面停留 | 在页面的时间 | 过短 (<1s) 或过长 (>5min) |
| 设备信息 | User-Agent | 包含自动化工具特征 |
| 滚动深度 | 页面滚动比例 | 没有滚动行为 |

风险评分 0-1，超过阈值 (默认 0.7) 需要二次验证。

## 📊 与其他方案对比

| 特性 | ECaptcha | Turnstile | reCAPTCHA |
|------|----------|-----------|-----------|
| 部署方式 | 自托管 | 云服务 | 云服务 |
| 费用 | 免费 | 免费 | 免费/付费 |
| 数据隐私 | ✅ 完全自控 | ❌ 第三方 | ❌ 第三方 |
| 定制化 | ✅ 高 | ❌ 低 | ❌ 低 |
| 维护成本 | 需自行维护 | 无需维护 | 无需维护 |
| 离线可用 | ✅ 支持 | ❌ 不支持 | ❌ 不支持 |

## 🔒 安全建议

1. **生产环境配置**
   - 使用强随机的 `secret-key`
   - 配置 Redis 密码认证
   - 启用 HTTPS

2. **防护策略**
   - 对同一 IP 限制请求频率
   - 监控异常验证行为
   - 定期更新行为分析规则

3. **Token 使用**
   - Token 仅可使用一次
   - 验证后立即失效
   - 设置合理的过期时间

## 📚 更多文档

- [前端 SDK 文档](./sdk/README.md) - 详细的前端集成指南
- [示例项目](./example/) - 完整的集成示例

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 License

[MIT License](LICENSE)
