/**
 * 治理 API 客户端封装 —— 统一注入认证头、错误处理、请求 ID。
 *
 * 设计要点：
 *   1. 浏览器侧 GOV 页面使用相对路径 /v1/gov/*，由 Next.js dev server 代理到后端 :8080。
 *   2. GOV 后端只接受 Authorization: Bearer <token>，不直接接受浏览器 cookie 会话。
 *   3. dev 环境统一从 NEXT_PUBLIC_GOV_API_TOKEN 注入管理 token；生产环境必须替换为
 *      与后端 AdminToken 一致的密钥，并由部署环境注入。
 *   4. 所有响应统一提取 error 字段并抛出可读消息。
 *
 * 严禁：
 *   - 在 .tsx 组件中直接调用 fetch(/v1/gov/...)，必须通过此模块。
 *   - 硬编码 token 字符串，token 必须来自环境变量。
 */

const API_BASE = "/v1/gov";

/** 从 NEXT_PUBLIC_GOV_API_TOKEN 读取治理 API Token；空字符串表示不附加认证头。 */
function readAdminToken(): string {
  // 浏览器侧通过 NEXT_PUBLIC_* 注入
  const token = process.env.NEXT_PUBLIC_GOV_API_TOKEN?.trim();
  return token ?? "";
}

/** 发起治理 API 请求。 */
export async function govFetch(
  path: string,
  init: RequestInit = {},
): Promise<Response> {
  const url = path.startsWith("http") ? path : `${API_BASE}${path}`;
  const headers = new Headers(init.headers);
  // 1. 注入 Bearer Token
  const token = readAdminToken();
  if (token && !headers.has("Authorization")) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  // 2. 注入全链路请求 ID
  if (!headers.has("X-Request-ID")) {
    headers.set("X-Request-ID", generateRequestID());
  }
  // 3. 缺省 Content-Type
  if (init.body && !headers.has("Content-Type") && !(init.body instanceof FormData)) {
    headers.set("Content-Type", "application/json");
  }
  // 4. 透传 credentials
  const finalInit: RequestInit = { ...init, headers, credentials: init.credentials ?? "include" };
  return fetch(url, finalInit);
}

/** 发起治理 API 请求并解析 JSON。 */
export async function govFetchJSON<T = unknown>(
  path: string,
  init: RequestInit = {},
): Promise<T> {
  const res = await govFetch(path, init);
  if (!res.ok) {
    const text = await res.text();
    let msg = `${res.status} ${res.statusText}`;
    try {
      const obj = JSON.parse(text);
      if (obj?.error?.message) msg = obj.error.message;
      else if (obj?.message) msg = obj.message;
    } catch {
      if (text) msg = text.slice(0, 200);
    }
    throw new Error(`[gov-api] ${msg}`);
  }
  // 204 No Content
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}

/** 生成请求 ID —— 16 位随机 hex 字符串。 */
function generateRequestID(): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  // 兜底：基于 Math.random 的简易 ID（仅 dev 兜底用，不用于生产追踪）
  return `req_${Math.random().toString(16).slice(2, 10)}${Date.now().toString(16)}`;
}

/** 暴露的常量 —— 供页面使用 API 路径前缀 */
export const GOV_API_BASE = API_BASE;
