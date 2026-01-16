"""
Login page object for Hometax authentication.
"""
import asyncio
from typing import Optional

import structlog
from playwright.async_api import Page, TimeoutError as PlaywrightTimeout

from ..constants import (
    AUTH_CERT,
    AUTH_ID_PW,
    ERROR_MESSAGES,
    HOMETAX_MAIN_URL,
    SELECTORS,
    TIMEOUTS,
)

logger = structlog.get_logger()


class LoginPage:
    """
    Page object for Hometax login operations.

    Supports certificate-based and ID/password authentication.
    """

    def __init__(self, page: Page) -> None:
        """Initialize login page with Playwright page instance."""
        self.page = page
        self.log = logger.bind(component="LoginPage")

    async def navigate(self) -> None:
        """Navigate to Hometax main page."""
        self.log.info("navigating_to_hometax")
        await self.page.goto(HOMETAX_MAIN_URL, timeout=TIMEOUTS["page_load"])
        await self.page.wait_for_load_state("networkidle")
        self.log.info("navigation_complete")

    async def login_with_certificate(
        self,
        cert_password: str,
        cert_index: int = 0,
    ) -> bool:
        """
        Login using public key certificate (NPKI).

        Args:
            cert_password: Certificate password
            cert_index: Index of certificate to select (0-based)

        Returns:
            True if login successful, False otherwise
        """
        self.log.info("certificate_login_started")

        try:
            # Click certificate login button
            await self.page.click(SELECTORS["login_cert_btn"])
            await self._wait_for_loading()

            # Wait for certificate selection popup
            await self.page.wait_for_selector(
                SELECTORS["cert_select_popup"],
                timeout=TIMEOUTS["element_wait"],
            )

            # Select certificate (by index)
            cert_list = await self.page.query_selector_all(".cert_list li")
            if cert_index < len(cert_list):
                await cert_list[cert_index].click()

            # Enter certificate password
            await self.page.fill(SELECTORS["cert_pw_input"], cert_password)

            # Confirm
            await self.page.click(SELECTORS["cert_confirm_btn"])
            await self._wait_for_loading()

            # Check for login success
            success = await self._check_login_success()
            if success:
                self.log.info("certificate_login_success")
            else:
                self.log.warning("certificate_login_failed")

            return success

        except PlaywrightTimeout as e:
            self.log.error("certificate_login_timeout", error=str(e))
            return False
        except Exception as e:
            self.log.error("certificate_login_error", error=str(e))
            return False

    async def login_with_credentials(
        self,
        user_id: str,
        password: str,
    ) -> bool:
        """
        Login using ID and password.

        Args:
            user_id: Hometax user ID
            password: Password

        Returns:
            True if login successful, False otherwise
        """
        self.log.info("credential_login_started", user_id=user_id[:3] + "***")

        try:
            # Fill in credentials
            await self.page.fill(SELECTORS["login_id_input"], user_id)
            await self.page.fill(SELECTORS["login_pw_input"], password)

            # Submit login form
            await self.page.click(SELECTORS["login_submit"])
            await self._wait_for_loading()

            # Check for login success
            success = await self._check_login_success()
            if success:
                self.log.info("credential_login_success")
            else:
                self.log.warning("credential_login_failed")

            return success

        except PlaywrightTimeout as e:
            self.log.error("credential_login_timeout", error=str(e))
            return False
        except Exception as e:
            self.log.error("credential_login_error", error=str(e))
            return False

    async def _wait_for_loading(self) -> None:
        """Wait for loading indicator to disappear."""
        try:
            # Wait for loading indicator to appear
            await self.page.wait_for_selector(
                SELECTORS["loading_indicator"],
                state="visible",
                timeout=1000,
            )
        except PlaywrightTimeout:
            # Loading indicator may not appear for fast operations
            pass

        try:
            # Wait for loading indicator to disappear
            await self.page.wait_for_selector(
                SELECTORS["loading_indicator"],
                state="hidden",
                timeout=TIMEOUTS["navigation"],
            )
        except PlaywrightTimeout:
            pass

        # Additional wait for animations
        await asyncio.sleep(TIMEOUTS["animation"] / 1000)

    async def _check_login_success(self) -> bool:
        """Check if login was successful."""
        try:
            # Check for error alert
            alert = await self.page.query_selector(SELECTORS["alert_popup"])
            if alert:
                alert_text = await alert.text_content()
                if any(msg in alert_text for msg in ERROR_MESSAGES.values()):
                    self.log.warning("login_error_alert", message=alert_text)
                    return False

            # Check for logged-in state (user info visible)
            # This selector varies based on Hometax UI version
            user_info = await self.page.query_selector(".user_info, #userInfo, .login_user")
            return user_info is not None

        except Exception as e:
            self.log.error("check_login_error", error=str(e))
            return False

    async def logout(self) -> None:
        """Logout from Hometax."""
        self.log.info("logout_started")
        try:
            logout_btn = await self.page.query_selector(".logout_btn, #logout, [title='로그아웃']")
            if logout_btn:
                await logout_btn.click()
                await self._wait_for_loading()
                self.log.info("logout_success")
        except Exception as e:
            self.log.error("logout_error", error=str(e))

    async def get_user_info(self) -> Optional[dict]:
        """Get logged-in user information."""
        try:
            user_info_el = await self.page.query_selector(".user_info, #userInfo")
            if not user_info_el:
                return None

            text = await user_info_el.text_content()
            return {"display_name": text.strip() if text else None}

        except Exception as e:
            self.log.error("get_user_info_error", error=str(e))
            return None

    async def is_logged_in(self) -> bool:
        """Check if currently logged in."""
        user_info = await self.page.query_selector(".user_info, #userInfo, .login_user")
        return user_info is not None
