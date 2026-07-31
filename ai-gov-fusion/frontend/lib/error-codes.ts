/**
 * 错误码中文映射表。
 * 将后端返回的 error.code 映射为用户可读的中文提示文案。
 * 覆盖 PRD §6 全部错误码。
 */
const ERROR_MESSAGES: Record<string, string> = {
  // === 预算相关 ===
  BUDGET_CAP_EXCEEDED: '预算已达上限',
  MODEL_BUDGET_EXCEEDED: '模型配额已用尽',
  BUDGET_WARN_THRESHOLD: '预算接近告警阈值',

  // === 余额相关 ===
  INSUFFICIENT_BALANCE: '可用余额不足',

  // === 模型权限 ===
  MODEL_ACCESS_DENIED: '无权访问该模型',

  // === 认证相关 ===
  AUTH_INVALID_KEY: 'API Key 无效',
  AUTH_USER_DISABLED: '用户已被禁用',
  AUTH_TOKEN_EXPIRED: '登录凭证已过期，请重新登录',

  // === 授权相关 ===
  AUTHZ_DENIED: '权限不足',

  // === 资源相关 ===
  RESOURCE_NOT_FOUND: '资源不存在',
  RESOURCE_CONFLICT: '资源冲突，可能已存在',

  // === 校验相关 ===
  VALIDATION_ERROR: '请求参数校验失败',
  INVALID_TRANSITION: '状态流转不合法',

  // === 幂等相关 ===
  IDEMPOTENCY_CONFLICT: '幂等键冲突，请求可能已处理',

  // === 限流相关 ===
  RATE_LIMITED: '请求频率过高，请稍后重试',

  // === 系统相关 ===
  INTERNAL_ERROR: '服务内部错误',
  SERVICE_UNAVAILABLE: '服务暂时不可用',
  UPSTREAM_ERROR: '上游服务异常',

  // === 资金相关 ===
  FUND_FROZEN: '账户已冻结，无法操作',
  FUND_ALLOCATION_FAILED: '资金划拨失败',
  FUND_LIQUIDATION_FAILED: '清算操作失败',

  // === 路由相关 ===
  ROUTE_PROFILE_IN_USE: '路由档案正在使用中，无法删除',
  DELTA_CAP_EXCEEDED: 'delta_cap 超出硬上限 20%',

  // === UI 权限相关 ===
  UI_ACTION_DENIED: '无此操作权限',
  UI_MENU_NOT_VISIBLE: '菜单不可见',
};

/**
 * 根据错误码获取中文提示文案。
 * @param code 后端返回的 error.code，如 "BUDGET_CAP_EXCEEDED"
 * @returns 中文提示文案，若未匹配则返回 null
 */
export function getErrorMessage(code: string | undefined | null): string | null {
  if (!code) return null;
  return ERROR_MESSAGES[code] ?? null;
}

/**
 * 从 fetch Response 中提取错误码并映射为中文提示。
 * 使用方式：
 *   if (!res.ok) {
 *     const message = await extractErrorMessage(res);
 *     throw new Error(message);
 *   }
 *
 * @param res fetch Response 对象
 * @returns 中文错误提示文案
 */
export async function extractErrorMessage(res: Response): Promise<string> {
  try {
    const body = await res.json();
    const code: string | undefined = body?.error?.code;
    const mapped = getErrorMessage(code);
    if (mapped) return mapped;
    // 回退：使用服务端返回的 message，再回退到 HTTP 状态码
    return body?.error?.message ?? `请求失败 (HTTP ${res.status})`;
  } catch {
    // JSON 解析失败或非 JSON 响应体
    return `请求失败 (HTTP ${res.status})`;
  }
}
