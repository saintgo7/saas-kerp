"""
gRPC transport security and caller authentication for the Python services.

Both Python services previously called `add_insecure_port("0.0.0.0:5005x")` and
registered no interceptor at all. Anything able to reach the port could invoke
`IssueTaxInvoice`, `CancelTaxInvoice` or `SubmitAcquisition` -- the last of
which carries a plaintext 주민등록번호 -- with no credential and no transport
encryption.

This module supplies:

- `ServerSecurity.from_settings`: reads TLS material and a shared auth token
  from the environment and decides how the port is opened.
- `TokenAuthInterceptor`: rejects any RPC without a valid bearer token.
- `bind_port`: opens a TLS port when certificates are configured, otherwise an
  insecure one -- and refuses to open an unauthenticated insecure port outside
  development.

Environment variables (prefix defaults to `GRPC`):
    GRPC_TLS_CERT_FILE     server certificate (PEM)
    GRPC_TLS_KEY_FILE      server private key (PEM)
    GRPC_TLS_CLIENT_CA     client CA for mTLS; when set, client certs required
    GRPC_AUTH_TOKEN        shared bearer token callers must present
    GRPC_ALLOW_INSECURE    "true" to permit an unauthenticated plaintext port
"""
import os
import secrets
from dataclasses import dataclass
from typing import Optional, Sequence

import grpc
import structlog

logger = structlog.get_logger(__name__)

# Metadata key carrying the caller's bearer token.
AUTH_METADATA_KEY = "authorization"
_BEARER_PREFIX = "bearer "

# RPCs reachable without a token: health probes must work for the orchestrator.
DEFAULT_PUBLIC_METHODS = (
    "/grpc.health.v1.Health/Check",
    "/grpc.health.v1.Health/Watch",
)


class InsecureConfigurationError(RuntimeError):
    """Raised when a server would start without transport security or auth."""


def _env_flag(name: str, default: bool = False) -> bool:
    """Read a boolean environment variable."""
    raw = os.getenv(name)
    if raw is None:
        return default
    return raw.strip().lower() in ("1", "true", "yes", "on")


@dataclass
class ServerSecurity:
    """Resolved transport-security and authentication settings for a server."""

    tls_cert_file: Optional[str] = None
    tls_key_file: Optional[str] = None
    tls_client_ca_file: Optional[str] = None
    auth_token: Optional[str] = None
    allow_insecure: bool = False
    environment: str = "development"

    @classmethod
    def from_env(
        cls,
        environment: str = "development",
        prefix: str = "GRPC",
    ) -> "ServerSecurity":
        """
        Build the security configuration from environment variables.

        Args:
            environment: Deployment environment name
            prefix: Environment variable prefix

        Returns:
            Resolved ServerSecurity
        """
        return cls(
            tls_cert_file=os.getenv(f"{prefix}_TLS_CERT_FILE") or None,
            tls_key_file=os.getenv(f"{prefix}_TLS_KEY_FILE") or None,
            tls_client_ca_file=os.getenv(f"{prefix}_TLS_CLIENT_CA") or None,
            auth_token=os.getenv(f"{prefix}_AUTH_TOKEN") or None,
            allow_insecure=_env_flag(f"{prefix}_ALLOW_INSECURE", default=False),
            environment=environment,
        )

    @property
    def tls_enabled(self) -> bool:
        """Whether a TLS key pair is configured."""
        return bool(self.tls_cert_file and self.tls_key_file)

    @property
    def mtls_enabled(self) -> bool:
        """Whether client certificates are required."""
        return bool(self.tls_enabled and self.tls_client_ca_file)

    @property
    def is_production(self) -> bool:
        """Whether this looks like a non-development deployment."""
        return self.environment.lower() not in ("development", "dev", "local", "test")

    def build_server_credentials(self) -> Optional[grpc.ServerCredentials]:
        """
        Build gRPC server credentials from the configured TLS material.

        Returns:
            ServerCredentials, or None when TLS is not configured
        """
        if not self.tls_enabled:
            return None

        with open(self.tls_key_file, "rb") as f:
            private_key = f.read()
        with open(self.tls_cert_file, "rb") as f:
            certificate_chain = f.read()

        root_certificates = None
        if self.tls_client_ca_file:
            with open(self.tls_client_ca_file, "rb") as f:
                root_certificates = f.read()

        return grpc.ssl_server_credentials(
            [(private_key, certificate_chain)],
            root_certificates=root_certificates,
            # With a client CA configured, an unauthenticated client is rejected
            # at the transport layer.
            require_client_auth=bool(root_certificates),
        )

    def validate(self) -> None:
        """
        Refuse configurations that would expose the service without protection.

        Raises:
            InsecureConfigurationError: If a production deployment has neither
                TLS nor a shared token, and has not explicitly opted in
        """
        if self.tls_enabled or self.auth_token:
            return

        if self.allow_insecure:
            logger.warning(
                "grpc_insecure_port_explicitly_allowed",
                environment=self.environment,
                message=(
                    "Serving without TLS and without caller authentication "
                    "because GRPC_ALLOW_INSECURE is set."
                ),
            )
            return

        if self.is_production:
            raise InsecureConfigurationError(
                f"Refusing to start in environment '{self.environment}' without "
                f"transport security or caller authentication. Set "
                f"GRPC_TLS_CERT_FILE/GRPC_TLS_KEY_FILE, or GRPC_AUTH_TOKEN, or "
                f"set GRPC_ALLOW_INSECURE=true to accept the risk deliberately."
            )

        logger.warning(
            "grpc_insecure_development_port",
            environment=self.environment,
            message=(
                "No TLS and no auth token. Acceptable for local development "
                "only -- this port accepts 세금계산서 발급 and 4대보험 취득신고 "
                "from any caller that can reach it."
            ),
        )


class TokenAuthInterceptor(grpc.aio.ServerInterceptor):
    """
    Reject RPCs that do not present the expected bearer token.

    A shared token is a floor, not a ceiling: prefer mTLS where the deployment
    supports it. Without either, any pod on the container network can file a
    취득신고 or cancel someone's 세금계산서.
    """

    def __init__(
        self,
        expected_token: str,
        public_methods: Sequence[str] = DEFAULT_PUBLIC_METHODS,
    ) -> None:
        """
        Initialize the interceptor.

        Args:
            expected_token: Token callers must present
            public_methods: Fully-qualified methods exempt from authentication
        """
        if not expected_token:
            raise ValueError("expected_token must not be empty")

        self._expected_token = expected_token
        self._public_methods = set(public_methods)

    async def intercept_service(self, continuation, handler_call_details):
        """
        Check the caller's token before dispatching to the handler.

        Args:
            continuation: Next interceptor / handler factory
            handler_call_details: Call metadata

        Returns:
            The RPC handler, or a handler that aborts with UNAUTHENTICATED
        """
        method = handler_call_details.method

        if method in self._public_methods:
            return await continuation(handler_call_details)

        metadata = dict(handler_call_details.invocation_metadata or ())
        presented = metadata.get(AUTH_METADATA_KEY, "")

        if presented.lower().startswith(_BEARER_PREFIX):
            presented = presented[len(_BEARER_PREFIX):]

        # Constant-time comparison: a token check that short-circuits on the
        # first differing byte leaks the token to a patient caller.
        if not presented or not secrets.compare_digest(presented, self._expected_token):
            logger.warning("grpc_unauthenticated_call", method=method)
            return _abort_handler(
                grpc.StatusCode.UNAUTHENTICATED,
                "Missing or invalid authentication token",
            )

        return await continuation(handler_call_details)


def _abort_handler(code: grpc.StatusCode, details: str) -> grpc.RpcMethodHandler:
    """Build an RPC handler that immediately aborts with the given status."""

    async def abort(request, context):
        await context.abort(code, details)

    return grpc.unary_unary_rpc_method_handler(abort)


def build_interceptors(security: ServerSecurity) -> list:
    """
    Build the interceptor chain for a server.

    Args:
        security: Resolved security settings

    Returns:
        List of interceptors (empty when no token is configured)
    """
    if security.auth_token:
        logger.info("grpc_token_auth_enabled")
        return [TokenAuthInterceptor(security.auth_token)]
    return []


def bind_port(
    server: grpc.aio.Server,
    address: str,
    security: ServerSecurity,
) -> int:
    """
    Bind the server's listening port, using TLS when configured.

    Args:
        server: The gRPC server
        address: host:port to bind
        security: Resolved security settings

    Returns:
        The bound port

    Raises:
        InsecureConfigurationError: If the configuration is unsafe for the
            declared environment
    """
    security.validate()

    credentials = security.build_server_credentials()
    if credentials is not None:
        port = server.add_secure_port(address, credentials)
        logger.info(
            "grpc_secure_port_bound",
            address=address,
            mutual_tls=security.mtls_enabled,
        )
        return port

    port = server.add_insecure_port(address)
    logger.warning("grpc_insecure_port_bound", address=address)
    return port


def reflection_enabled(environment: str, configured: bool) -> bool:
    """
    Decide whether to expose server reflection.

    Reflection hands a caller the full service schema. That is a convenience in
    development and a head start for an attacker in production, so production
    requires an explicit opt-in through GRPC_ENABLE_REFLECTION.

    Args:
        environment: Deployment environment name
        configured: The service's own reflection setting

    Returns:
        Whether reflection should be registered
    """
    if not configured:
        return False

    is_production = environment.lower() not in ("development", "dev", "local", "test")
    if is_production and not _env_flag("GRPC_ENABLE_REFLECTION", default=False):
        logger.info(
            "grpc_reflection_suppressed",
            environment=environment,
            message="Reflection is off in production; set GRPC_ENABLE_REFLECTION=true to override.",
        )
        return False

    return True
