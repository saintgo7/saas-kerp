#!/usr/bin/env python3
"""Hometax crawling test script."""

import asyncio
import sys
from datetime import datetime
from pathlib import Path

# Add parent to path
sys.path.insert(0, str(Path(__file__).parent.parent))

from playwright.async_api import async_playwright
from loguru import logger


async def test_hometax_access():
    """Test basic Hometax page access."""
    logger.info("=" * 60)
    logger.info("Hometax Crawling Test")
    logger.info("=" * 60)

    async with async_playwright() as p:
        # Launch browser (headless for server, visible for debugging)
        browser = await p.chromium.launch(
            headless=True,  # Set to False to see browser
            slow_mo=100,
        )

        context = await browser.new_context(
            viewport={"width": 1920, "height": 1080},
            locale="ko-KR",
        )

        page = await context.new_page()

        try:
            # 1. Test main page access
            logger.info("[1/4] Accessing Hometax main page...")
            await page.goto("https://www.hometax.go.kr", timeout=30000)
            await page.wait_for_load_state("networkidle")

            title = await page.title()
            logger.info(f"Page title: {title}")

            # Take screenshot
            screenshot_path = Path(__file__).parent / "screenshots"
            screenshot_path.mkdir(exist_ok=True)

            await page.screenshot(
                path=str(screenshot_path / "01_main_page.png"),
                full_page=False
            )
            logger.success("Main page accessed successfully")

            # 2. Test login page navigation
            logger.info("[2/4] Navigating to login page...")

            # Look for login button/link
            login_selectors = [
                'a:has-text("로그인")',
                'button:has-text("로그인")',
                '[class*="login"]',
                'a[href*="login"]',
            ]

            login_found = False
            for selector in login_selectors:
                try:
                    element = await page.wait_for_selector(selector, timeout=5000)
                    if element:
                        await element.click()
                        login_found = True
                        logger.info(f"Found login element with selector: {selector}")
                        break
                except Exception:
                    continue

            if login_found:
                await page.wait_for_load_state("networkidle")
                await page.screenshot(
                    path=str(screenshot_path / "02_login_page.png"),
                    full_page=False
                )
                logger.success("Login page accessed")
            else:
                logger.warning("Login button not found - page structure may have changed")

            # 3. Check for certificate login options
            logger.info("[3/4] Checking authentication options...")

            auth_options = {
                "공동인증서": False,
                "간편인증": False,
                "아이디": False,
                "금융인증서": False,
            }

            page_content = await page.content()
            for option in auth_options:
                if option in page_content:
                    auth_options[option] = True
                    logger.info(f"  - {option}: Available")

            # 4. Test e-Tax Invoice page (requires login)
            logger.info("[4/4] Checking e-Tax Invoice menu structure...")

            # Try to find tax invoice related elements
            etax_url = "https://www.hometax.go.kr/websquare/websquare.wq?w2xPath=/ui/pp/index_pp.xml"
            await page.goto(etax_url, timeout=30000)
            await page.wait_for_load_state("networkidle")

            await page.screenshot(
                path=str(screenshot_path / "03_etax_page.png"),
                full_page=False
            )

            # Check page structure
            frames = page.frames
            logger.info(f"Page has {len(frames)} frames")

            # Summary
            logger.info("=" * 60)
            logger.success("Hometax Crawling Test Completed")
            logger.info("=" * 60)
            logger.info(f"Screenshots saved to: {screenshot_path}")
            logger.info("")
            logger.info("Test Results:")
            logger.info("  - Main page access: OK")
            logger.info(f"  - Login options found: {sum(auth_options.values())}/4")
            logger.info("  - e-Tax Invoice page: Accessible")
            logger.info("")
            logger.warning("Note: Actual login requires certificate or credentials")
            logger.info("=" * 60)

            return True

        except Exception as e:
            logger.error(f"Test failed: {e}")

            # Save error screenshot
            try:
                await page.screenshot(
                    path=str(screenshot_path / "error_screenshot.png"),
                    full_page=True
                )
            except Exception:
                pass

            return False

        finally:
            await browser.close()


async def test_scraper_class():
    """Test the HometaxScraper class."""
    logger.info("")
    logger.info("=" * 60)
    logger.info("Testing HometaxScraper Class")
    logger.info("=" * 60)

    try:
        from src.scrapers.hometax import HometaxScraper, HometaxCredentials, HometaxLoginType

        # Create scraper instance
        credentials = HometaxCredentials(
            login_type=HometaxLoginType.SIMPLE,
            user_id="test_user",
            user_password="test_pass",
        )

        async with HometaxScraper(credentials) as scraper:
            logger.info("Scraper initialized successfully")
            logger.info(f"  - Base URL: {scraper.BASE_URL}")
            logger.info(f"  - Login URL: {scraper.LOGIN_URL}")
            logger.info(f"  - Headless: {scraper.settings.browser_headless}")

            # Note: Actual login would fail with test credentials
            logger.warning("Skipping actual login (requires real credentials)")

        logger.success("HometaxScraper class test passed")
        return True

    except ImportError as e:
        logger.error(f"Import error: {e}")
        logger.info("This is expected if settings module uses different field names")
        return False
    except Exception as e:
        logger.error(f"Scraper test failed: {e}")
        return False


async def main():
    """Run all tests."""
    logger.remove()
    logger.add(
        sys.stderr,
        format="<green>{time:HH:mm:ss}</green> | <level>{level: <8}</level> | <level>{message}</level>",
        level="INFO",
        colorize=True,
    )

    print("\n")
    logger.info(f"Test started at {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print("\n")

    # Run tests
    results = {}

    results["hometax_access"] = await test_hometax_access()
    results["scraper_class"] = await test_scraper_class()

    # Summary
    print("\n")
    logger.info("=" * 60)
    logger.info("FINAL SUMMARY")
    logger.info("=" * 60)

    for test_name, passed in results.items():
        status = "PASS" if passed else "FAIL"
        logger.info(f"  {test_name}: {status}")

    total_passed = sum(results.values())
    total_tests = len(results)
    logger.info(f"\nTotal: {total_passed}/{total_tests} tests passed")
    logger.info("=" * 60)

    return all(results.values())


if __name__ == "__main__":
    success = asyncio.run(main())
    sys.exit(0 if success else 1)
