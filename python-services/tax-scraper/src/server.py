"""
gRPC Service Implementation for Tax Invoice Operations
"""
from typing import Any

import grpc
import structlog

# Generated proto imports (will be available after proto generation)
try:
    from src.grpc_gen import tax_pb2, tax_pb2_grpc
except ImportError:
    tax_pb2 = None
    tax_pb2_grpc = None

from src.hometax.scraper import HometaxScraper

logger = structlog.get_logger()


class TaxInvoiceServicer:
    """gRPC servicer for tax invoice operations."""

    def __init__(self) -> None:
        """Initialize the servicer."""
        self.scraper = HometaxScraper()
        self.log = logger.bind(component="TaxInvoiceServicer")

    async def Login(
        self,
        request: Any,
        context: grpc.aio.ServicerContext,
    ) -> Any:
        """
        Login to Hometax with certificate or credentials.

        Args:
            request: LoginRequest containing authentication info
            context: gRPC context

        Returns:
            LoginResponse with session info
        """
        self.log.info(
            "login_attempt",
            auth_type=request.auth_type if hasattr(request, "auth_type") else "unknown",
        )

        try:
            session = await self.scraper.login(
                business_number=request.business_number,
                auth_type=request.auth_type,
                cert_password=request.cert_password if hasattr(request, "cert_password") else None,
                user_id=request.user_id if hasattr(request, "user_id") else None,
                password=request.password if hasattr(request, "password") else None,
            )

            self.log.info("login_success", business_number=request.business_number[:6] + "****")

            if tax_pb2:
                return tax_pb2.LoginResponse(
                    success=True,
                    session_id=session.session_id,
                    expires_at=session.expires_at.isoformat(),
                )
            return None

        except Exception as e:
            self.log.error("login_failed", error=str(e))
            context.set_code(grpc.StatusCode.UNAUTHENTICATED)
            context.set_details(f"Login failed: {str(e)}")
            if tax_pb2:
                return tax_pb2.LoginResponse(success=False, error_message=str(e))
            return None

    async def GetTaxInvoices(
        self,
        request: Any,
        context: grpc.aio.ServicerContext,
    ) -> Any:
        """
        Retrieve tax invoices from Hometax.

        Args:
            request: GetTaxInvoicesRequest with search criteria
            context: gRPC context

        Returns:
            GetTaxInvoicesResponse with list of invoices
        """
        self.log.info(
            "get_tax_invoices",
            session_id=request.session_id[:8] + "..." if request.session_id else None,
            start_date=request.start_date,
            end_date=request.end_date,
        )

        try:
            invoices = await self.scraper.get_tax_invoices(
                session_id=request.session_id,
                start_date=request.start_date,
                end_date=request.end_date,
                invoice_type=request.invoice_type if hasattr(request, "invoice_type") else None,
            )

            self.log.info("get_tax_invoices_success", count=len(invoices))

            if tax_pb2:
                return tax_pb2.GetTaxInvoicesResponse(
                    success=True,
                    invoices=[self._to_proto_invoice(inv) for inv in invoices],
                    total_count=len(invoices),
                )
            return None

        except Exception as e:
            self.log.error("get_tax_invoices_failed", error=str(e))
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Failed to retrieve invoices: {str(e)}")
            if tax_pb2:
                return tax_pb2.GetTaxInvoicesResponse(success=False, error_message=str(e))
            return None

    async def IssueTaxInvoice(
        self,
        request: Any,
        context: grpc.aio.ServicerContext,
    ) -> Any:
        """
        Issue a new tax invoice via Hometax.

        Args:
            request: IssueTaxInvoiceRequest with invoice details
            context: gRPC context

        Returns:
            IssueTaxInvoiceResponse with issued invoice info
        """
        self.log.info(
            "issue_tax_invoice",
            session_id=request.session_id[:8] + "..." if request.session_id else None,
            supplier_business_number=request.invoice.supplier_business_number[:6] + "****"
            if hasattr(request, "invoice")
            else None,
        )

        try:
            result = await self.scraper.issue_tax_invoice(
                session_id=request.session_id,
                invoice_data=self._from_proto_invoice(request.invoice),
            )

            self.log.info(
                "issue_tax_invoice_success",
                invoice_number=result.invoice_number,
            )

            if tax_pb2:
                return tax_pb2.IssueTaxInvoiceResponse(
                    success=True,
                    invoice_number=result.invoice_number,
                    issue_date=result.issue_date.isoformat(),
                    nts_confirm_number=result.nts_confirm_number,
                )
            return None

        except Exception as e:
            self.log.error("issue_tax_invoice_failed", error=str(e))
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Failed to issue invoice: {str(e)}")
            if tax_pb2:
                return tax_pb2.IssueTaxInvoiceResponse(success=False, error_message=str(e))
            return None

    async def Logout(
        self,
        request: Any,
        context: grpc.aio.ServicerContext,
    ) -> Any:
        """
        Logout from Hometax and invalidate session.

        Args:
            request: LogoutRequest with session info
            context: gRPC context

        Returns:
            LogoutResponse
        """
        self.log.info(
            "logout",
            session_id=request.session_id[:8] + "..." if request.session_id else None,
        )

        try:
            await self.scraper.logout(session_id=request.session_id)
            self.log.info("logout_success")

            if tax_pb2:
                return tax_pb2.LogoutResponse(success=True)
            return None

        except Exception as e:
            self.log.error("logout_failed", error=str(e))
            if tax_pb2:
                return tax_pb2.LogoutResponse(success=False, error_message=str(e))
            return None

    def _to_proto_invoice(self, invoice: Any) -> Any:
        """Convert domain invoice to proto message."""
        if not tax_pb2:
            return None

        return tax_pb2.TaxInvoice(
            invoice_number=invoice.invoice_number,
            issue_date=invoice.issue_date.isoformat(),
            supplier_business_number=invoice.supplier_business_number,
            supplier_name=invoice.supplier_name,
            buyer_business_number=invoice.buyer_business_number,
            buyer_name=invoice.buyer_name,
            supply_amount=invoice.supply_amount,
            tax_amount=invoice.tax_amount,
            total_amount=invoice.total_amount,
            invoice_type=invoice.invoice_type,
            nts_confirm_number=invoice.nts_confirm_number or "",
        )

    def _from_proto_invoice(self, proto_invoice: Any) -> dict:
        """Convert proto message to domain invoice dict."""
        return {
            "supplier_business_number": proto_invoice.supplier_business_number,
            "supplier_name": proto_invoice.supplier_name,
            "buyer_business_number": proto_invoice.buyer_business_number,
            "buyer_name": proto_invoice.buyer_name,
            "supply_amount": proto_invoice.supply_amount,
            "tax_amount": proto_invoice.tax_amount,
            "total_amount": proto_invoice.total_amount,
            "items": [
                {
                    "description": item.description,
                    "quantity": item.quantity,
                    "unit_price": item.unit_price,
                    "amount": item.amount,
                }
                for item in proto_invoice.items
            ],
        }
