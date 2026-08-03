import type { NextConfig } from "next";

/**
 * AI-GOV 前端 Next.js 配置。
 *
 * 反向代理（rewrites）：
 *   - 浏览器相对路径 /v1/* 必须由 Next.js dev server 代理到后端 :8080。
 *   - /api/admin/* 同样代理到后端。
 *   - /livez /readyz /healthz /metrics 健康检查端点代理到后端。
 *   - 静态资源、Next.js 内部路径不代理。
 *
 * 严禁：
 *   - 不得把 /v1/* 路径写进前端 page.tsx，破坏 SSR 路由匹配。
 *   - 不得在客户端代码中硬编码 http://localhost:8080，全部走相对路径。
 */
const backendBaseURL =
  process.env.TOKENHUB_API_BASE_URL?.trim() ||
  process.env.NEXT_PUBLIC_API_BASE_URL?.trim() ||
  "http://localhost:8080";

const nextConfig: NextConfig = {
  reactStrictMode: true,
  output: "standalone",
  allowedDevOrigins: ["127.0.0.1", "localhost"],
  async rewrites() {
    return [
      // 治理 API 与数据面 API 一并代理到后端
      { source: "/v1/:path*", destination: `${backendBaseURL}/v1/:path*` },
      { source: "/api/admin/:path*", destination: `${backendBaseURL}/api/admin/:path*` },
      // 健康检查端点
      { source: "/livez", destination: `${backendBaseURL}/livez` },
      { source: "/readyz", destination: `${backendBaseURL}/readyz` },
      { source: "/healthz", destination: `${backendBaseURL}/healthz` },
      { source: "/metrics", destination: `${backendBaseURL}/metrics` },
    ];
  },
};

export default nextConfig;
