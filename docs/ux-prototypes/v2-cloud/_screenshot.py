"""Playwright Python 截图脚本:将 v2-cloud/index.html 截成单图 + 每页单独截"""
import os
from pathlib import Path
from playwright.sync_api import sync_playwright

OUT_DIR = Path(__file__).parent
HTML_PATH = OUT_DIR / "index.html"


def main():
    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page(viewport={"width": 1440, "height": 900})
        file_url = "file:///" + str(HTML_PATH).replace("\\", "/")
        page.goto(file_url, wait_until="networkidle")
        # 等待 Tailwind CDN 注入完成
        page.wait_for_timeout(2000)

        # 1) 全页长截图
        full_path = OUT_DIR / "screenshot-full.png"
        page.screenshot(path=str(full_path), full_page=True)
        print(f"Saved: {full_path}")

        # 2) 首屏截图(1440x900 视口)
        vp_path = OUT_DIR / "screenshot.png"
        page.screenshot(path=str(vp_path), full_page=False)
        print(f"Saved: {vp_path}")

        # 3) 各 section 单独截图
        sections = ["dashboard", "sidebar", "sidebar-v2", "account", "model", "audit"]
        for sid in sections:
            el = page.query_selector(f"#{sid}")
            if el:
                p_out = OUT_DIR / f"screenshot-{sid}.png"
                el.screenshot(path=str(p_out))
                print(f"Saved: {p_out}")

        browser.close()
    print("Done.")


if __name__ == "__main__":
    main()
