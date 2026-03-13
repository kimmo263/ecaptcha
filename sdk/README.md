<div align="center">

# 🛡️ ECaptcha Frontend SDK

**轻量级人机验证前端 SDK**

[![npm version](https://img.shields.io/npm/v/@ectrack/ecaptcha.svg)](https://www.npmjs.com/package/@ectrack/ecaptcha)
[![bundle size](https://img.shields.io/badge/gzip-~8kb-brightgreen)](https://bundlephobia.com/package/@ectrack/ecaptcha)
[![TypeScript](https://img.shields.io/badge/TypeScript-Ready-blue)](https://www.typescriptlang.org/)

</div>

---

支持图片验证码、滑动拼图、无感验证，兼容 React / Vue / 原生 JS。

## 📦 安装

### 方式一：CDN 引入（推荐快速体验）

```html
<script src="https://your-domain/ecaptcha/sdk/ecaptcha.min.js"></script>
<script>
  // 初始化
  ECaptcha.init({
    server: 'https://api.example.com',  // 后端 API 地址
    lang: 'zh-CN',
    theme: 'light'
  });
  
  // 使用
  document.getElementById('send-code-btn').onclick = async () => {
    try {
      const token = await ECaptcha.verify('image');
      // 发送验证码
      await sendCode({ phone: '13800138000', captcha_token: token });
    } catch (e) {
      console.log('用户取消验证');
    }
  };
</script>
```

### 方式二：NPM 安装（推荐生产使用）

```bash
npm install @ectrack/ecaptcha
# 或
yarn add @ectrack/ecaptcha
# 或
pnpm add @ectrack/ecaptcha
```

```typescript
import ECaptcha from '@ectrack/ecaptcha';

ECaptcha.init({
  server: 'https://api.example.com'
});

// 图片验证码
const token = await ECaptcha.verify('image');

// 滑动拼图
const token = await ECaptcha.verify('slider');

// 无感验证 (行为分析)
const token = await ECaptcha.verify('behavior');

// 智能验证 (先无感，失败降级到滑动)
const token = await ECaptcha.smart();
```

## 📖 API

### ECaptcha.init(options)

初始化配置。

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| server | string | ✅ | - | 后端 API 地址 |
| prefix | string | - | /ecaptcha | API 路径前缀 |
| lang | string | - | zh-CN | 语言 (zh-CN / en-US) |
| theme | string | - | light | 主题 (light / dark) |
| zIndex | number | - | 10000 | 弹窗层级 |

### ECaptcha.verify(type)

执行验证，返回 Promise。

| 参数 | 说明 |
|------|------|
| image | 图片验证码 |
| slider | 滑动拼图 |
| behavior | 无感验证 (行为分析) |

```typescript
try {
  const token = await ECaptcha.verify('slider');
  console.log('验证成功:', token);
} catch (e) {
  console.log('用户取消或验证失败');
}
```

### ECaptcha.smart()

智能验证：先尝试无感验证，如果风险评分过高则自动降级到滑动拼图。

```typescript
const token = await ECaptcha.smart();
```

### ECaptcha.validateToken(token)

验证 token 是否有效。

```typescript
const isValid = await ECaptcha.validateToken(token);
```

## 💡 完整示例

### 登录页面

```html
<!DOCTYPE html>
<html>
<head>
  <title>登录</title>
  <script src="https://api.example.com/ecaptcha/sdk/ecaptcha.min.js"></script>
</head>
<body>
  <form id="login-form">
    <input type="text" id="phone" placeholder="手机号" />
    <button type="button" id="send-code">发送验证码</button>
    <input type="text" id="code" placeholder="验证码" />
    <button type="submit">登录</button>
  </form>

  <script>
    ECaptcha.init({ server: 'https://api.example.com' });

    document.getElementById('send-code').onclick = async () => {
      const phone = document.getElementById('phone').value;
      if (!phone) return alert('请输入手机号');

      try {
        // 1. 人机验证
        const captchaToken = await ECaptcha.smart();
        
        // 2. 发送验证码
        const res = await fetch('/user/v1/auth/send-code', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            phone,
            country_code: '86',
            scene: 'login',
            captcha_token: captchaToken
          })
        });
        
        const data = await res.json();
        if (data.success) {
          alert('验证码已发送');
        } else {
          alert(data.message);
        }
      } catch (e) {
        console.log('验证取消');
      }
    };
  </script>
</body>
</html>
```

### React 组件

```tsx
import { useCallback } from 'react';
import ECaptcha from '@ectrack/ecaptcha';

// 初始化 (在 App 入口调用一次)
ECaptcha.init({ server: import.meta.env.VITE_API_URL });

export function SendCodeButton({ phone, onSuccess }) {
  const handleClick = useCallback(async () => {
    try {
      // 人机验证
      const captchaToken = await ECaptcha.smart();
      
      // 发送验证码
      const res = await fetch('/user/v1/auth/send-code', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          phone,
          scene: 'register',
          captcha_token: captchaToken
        })
      });
      
      const data = await res.json();
      if (data.success) {
        onSuccess?.();
      }
    } catch (e) {
      // 用户取消
    }
  }, [phone, onSuccess]);

  return <button onClick={handleClick}>发送验证码</button>;
}
```

### Vue 组件

```vue
<template>
  <button @click="sendCode" :disabled="loading">
    {{ loading ? '发送中...' : '发送验证码' }}
  </button>
</template>

<script setup>
import { ref } from 'vue';
import ECaptcha from '@ectrack/ecaptcha';

ECaptcha.init({ server: import.meta.env.VITE_API_URL });

const props = defineProps(['phone']);
const emit = defineEmits(['success']);
const loading = ref(false);

async function sendCode() {
  if (!props.phone) return;
  
  try {
    loading.value = true;
    
    // 人机验证
    const captchaToken = await ECaptcha.smart();
    
    // 发送验证码
    const res = await fetch('/user/v1/auth/send-code', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        phone: props.phone,
        scene: 'register',
        captcha_token: captchaToken
      })
    });
    
    const data = await res.json();
    if (data.success) {
      emit('success');
    }
  } catch (e) {
    // 用户取消
  } finally {
    loading.value = false;
  }
}
</script>
```

## 🔗 后端集成

后端需要提供以下接口：

| 接口 | 方法 | 说明 |
|------|------|------|
| /ecaptcha/generate | POST | 生成验证挑战 |
| /ecaptcha/verify | POST | 验证用户答案 |
| /ecaptcha/validate | POST | 验证 token |

详见后端 SDK 文档：[pkg/ecaptcha/README.md](../README.md)

## 🎨 自定义样式

SDK 使用 CSS 类名，可通过覆盖样式自定义外观：

```css
/* 弹窗背景 */
.ecaptcha-modal { background: rgba(0,0,0,0.7); }

/* 容器 */
.ecaptcha-container { border-radius: 12px; }

/* 标题 */
.ecaptcha-title { color: #333; }

/* 输入框 */
.ecaptcha-input { border-color: #4a90d9; }

/* 滑动按钮 */
.ecaptcha-slider-btn { background: #4a90d9; }
```

## 🌐 浏览器支持

| 浏览器 | 最低版本 |
|--------|----------|
| Chrome | 60+ |
| Firefox | 55+ |
| Safari | 12+ |
| Edge | 79+ |
| iOS Safari | 12+ |
| Android Chrome | 60+ |

## 📄 License

[MIT License](../LICENSE)
