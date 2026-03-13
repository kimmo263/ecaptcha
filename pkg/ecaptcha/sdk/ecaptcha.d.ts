/**
 * ECaptcha - 人机验证前端 SDK 类型定义
 */

declare namespace ECaptcha {
  interface InitOptions {
    /** API 服务器地址 (必填) */
    server: string;
    /** API 前缀，默认 /ecaptcha */
    prefix?: string;
    /** 语言，zh-CN / en-US */
    lang?: 'zh-CN' | 'en-US';
    /** 主题，light / dark */
    theme?: 'light' | 'dark';
    /** 弹窗层级 */
    zIndex?: number;
  }

  type CaptchaType = 'image' | 'slider' | 'behavior';

  /** SDK 版本 */
  const version: string;

  /**
   * 初始化配置
   */
  function init(options: InitOptions): void;

  /**
   * 执行验证
   * @param type 验证类型
   * @returns 验证成功返回 token
   */
  function verify(type?: CaptchaType): Promise<string>;

  /**
   * 验证 token 是否有效
   */
  function validateToken(token: string): Promise<boolean>;

  /**
   * 智能验证 (先尝试无感，失败则降级到滑动)
   */
  function smart(): Promise<string>;
}

export = ECaptcha;
export as namespace ECaptcha;
