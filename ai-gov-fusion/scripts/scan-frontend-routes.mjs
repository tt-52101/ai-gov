#!/usr/bin/env node
/**
 * scan-frontend-routes.mjs
 *
 * 扫描前端治理控制台（app/(console)/gov/）的真实路由、菜单与按钮授权声明，
 * 生成 UI 权限 manifest（JSON），供后端 bootstrap 回灌到数据库。
 *
 * 设计原则（满足"禁止恶意注入"要求）：
 *  1. 纯静态文本扫描，绝不执行任何前端代码（无 eval / 无动态 import 执行）。
 *  2. 只接受 /gov/ 前缀的路由；拒绝任何包含 ".."、绝对路径或越界前缀的路径。
 *  3. 菜单定义只来自 layout.tsx 的 allNavItems（前端真实声明的导航）。
 *  4. 路由权限只来自 _route_permissions.ts 的 govRoutePermissions（前端真实声明）。
 *  5. 按钮授权只来自页面中的 data-permission / data-action 标记（前端真实声明）。
 *  6. 不推断、不伪造任何菜单/路由/按钮——未在前端声明的项一律不回灌。
 *
 * 输出：backend/internal/server/ui_permission/frontend_manifest.json
 *   {
 *     "scannedAt": "...",
 *     "sourceRoot": "...",
 *     "menus":   [{ code, label, icon, href }],
 *     "routes":  [{ path, menuCode, requiredAction }],
 *     "buttons": [{ code, page, requiredAction }]
 *   }
 */

import { readFileSync, writeFileSync, existsSync, readdirSync, statSync } from "node:fs";
import { join, resolve, relative, sep } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = fileURLToPath(new URL(".", import.meta.url));
const repoRoot = resolve(__dirname, ".."); // ai-gov-fusion/
const govDir = join(repoRoot, "frontend", "app", "(console)", "gov");
const manifestOut = join(
  repoRoot,
  "backend",
  "internal",
  "server",
  "ui_permission",
  "frontend_manifest.json"
);

/** 安全校验：路径必须位于 govDir 内且路由前缀为 /gov/ */
function assertSafeRoute(routePath) {
  if (!routePath.startsWith("/gov/")) {
    throw new Error(`拒绝回灌非 /gov/ 前缀路由: ${routePath}（防止恶意注入越界路由）`);
  }
  if (routePath.includes("..") || routePath.startsWith("//")) {
    throw new Error(`拒绝回灌包含非法片段的路由: ${routePath}`);
  }
  return true;
}

/** 递归查找所有 page.tsx */
function findPages(dir) {
  const out = [];
  for (const entry of readdirSync(dir)) {
    if (entry === "node_modules" || entry === ".next") continue;
    const full = join(dir, entry);
    const st = statSync(full);
    if (st.isDirectory()) {
      out.push(...findPages(full));
    } else if (entry === "page.tsx") {
      out.push(full);
    }
  }
  return out;
}

/** 将 page.tsx 文件路径转换为前端路由 */
function pageToRoute(file) {
  const rel = relative(govDir, file).split(sep).join("/");
  // govDir 已指向 app/(console)/gov，rel 形如 "dashboard/page.tsx"
  // 统一加 /gov/ 前缀，确保路由挂载在治理控制台命名空间下。
  const route = "/gov/" + rel.replace(/\/page\.tsx$/, "");
  return route;
}

/** 从 layout.tsx 提取 allNavItems 数组 */
function parseNavItems(layoutPath) {
  const src = readFileSync(layoutPath, "utf-8");
  const items = [];
  // 匹配每个 { href: "...", label: "...", icon: ..., code: "..." }
  const re = /\{\s*href:\s*"([^"]+)"\s*,\s*label:\s*"([^"]+)"\s*,\s*icon:\s*(\w+)\s*,\s*code:\s*"([^"]+)"\s*\}/g;
  let m;
  while ((m = re.exec(src)) !== null) {
    items.push({ href: m[1], label: m[2], icon: m[3], code: m[4] });
  }
  if (items.length === 0) {
    throw new Error("未能从 layout.tsx 解析出 allNavItems（前端导航声明缺失？）");
  }
  return items;
}

/** 从 _route_permissions.ts 提取 govRoutePermissions */
function parseRoutePermissions(permPath) {
  const src = readFileSync(permPath, "utf-8");
  const map = new Map();
  // 匹配 { route: "...", action: "...", menuCode: "..." }
  const re = /\{\s*route:\s*"([^"]+)"\s*,\s*action:\s*"([^"]+)"(?:\s*,\s*menuCode:\s*"([^"]+)")?\s*\}/g;
  let m;
  while ((m = re.exec(src)) !== null) {
    map.set(m[1], { action: m[2], menuCode: m[3] || null });
  }
  if (map.size === 0) {
    throw new Error("未能从 _route_permissions.ts 解析出 govRoutePermissions（前端路由权限声明缺失？）");
  }
  return map;
}

/** 从 page.tsx 提取按钮授权标记：data-permission="btn-x" data-action="a.b" */
function parseButtons(pagePath, route) {
  const src = readFileSync(pagePath, "utf-8");
  const buttons = [];
  const re = /data-permission="([^"]+)"(?:\s+data-action="([^"]+)")?/g;
  let m;
  while ((m = re.exec(src)) !== null) {
    buttons.push({ code: m[1], requiredAction: m[2] || null, page: route });
  }
  return buttons;
}

function main() {
  if (!existsSync(govDir)) {
    throw new Error(`治理控制台目录不存在: ${govDir}`);
  }

  const layoutPath = join(govDir, "layout.tsx");
  const permPath = join(govDir, "_route_permissions.ts");
  if (!existsSync(layoutPath)) throw new Error(`layout.tsx 不存在: ${layoutPath}`);
  if (!existsSync(permPath)) throw new Error(`_route_permissions.ts 不存在: ${permPath}`);

  // 1) 扫描真实路由（page.tsx）
  const pages = findPages(govDir);
  const routes = [];
  for (const p of pages) {
    const route = pageToRoute(p);
    assertSafeRoute(route); // 防恶意注入：只接受 /gov/ 前缀
    routes.push({ file: relative(repoRoot, p), route });
  }

  // 2) 解析菜单（前端 layout.tsx 真实声明）
  const navItems = parseNavItems(layoutPath);
  const menus = navItems.map((n) => ({
    code: n.code,
    label: n.label,
    icon: n.icon,
    href: n.href,
  }));

  // 3) 解析路由权限（前端 _route_permissions.ts 真实声明）
  const permMap = parseRoutePermissions(permPath);

  // 4) 组装 routes manifest，并校验每个真实路由都有权限声明
  const routeManifest = [];
  const orphanRoutes = [];
  for (const r of routes) {
    const perm = permMap.get(r.route);
    if (!perm) {
      // 前端有 page 但未在 _route_permissions.ts 声明 → 不回灌（防恶意注入）
      orphanRoutes.push(r.route);
      continue;
    }
    routeManifest.push({
      path: r.route,
      menuCode: perm.menuCode,
      requiredAction: perm.action,
    });
  }

  // 5) 扫描按钮授权标记
  const buttons = [];
  for (const r of routes) {
    const p = join(govDir, r.route.slice("/gov/".length) + "/page.tsx");
    if (existsSync(p)) {
      buttons.push(...parseButtons(p, r.route));
    }
  }

  // 6) 反向校验：_route_permissions.ts 声明的路由是否都有真实 page.tsx
  const realRoutes = new Set(routes.map((r) => r.route));
  const danglingPerms = [];
  for (const [route, perm] of permMap.entries()) {
    if (!realRoutes.has(route)) {
      danglingPerms.push({ route, action: perm.action });
    }
  }

  const manifest = {
    scannedAt: new Date().toISOString(),
    sourceRoot: relative(repoRoot, govDir),
    menus,
    routes: routeManifest,
    buttons,
    warnings: {
      orphanRoutes: orphanRoutes, // 有 page 但无权限声明 → 已跳过回灌
      danglingPermissions: danglingPerms, // 有权限声明但无 page → 可能路由挂载缺失
    },
  };

  writeFileSync(manifestOut, JSON.stringify(manifest, null, 2) + "\n", "utf-8");

  console.log(`✅ 扫描完成，manifest 已写入: ${relative(repoRoot, manifestOut)}`);
  console.log(`   菜单: ${menus.length} 项`);
  console.log(`   路由: ${routeManifest.length} 项（已回灌）`);
  console.log(`   按钮: ${buttons.length} 项`);
  if (orphanRoutes.length) {
    console.warn(`⚠️  有 ${orphanRoutes.length} 个真实路由未声明权限，已跳过回灌:`, orphanRoutes);
  }
  if (danglingPerms.length) {
    console.warn(`⚠️  有 ${danglingPerms.length} 个权限声明无对应页面（路由未正式挂载）:`, danglingPerms);
  }
}

main();
