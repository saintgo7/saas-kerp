"""
Explicit failure types for the Hometax scraping paths.

These exist so that "we did not do the thing" can never be reported as "we did
the thing". Every one of them maps to a distinct gRPC status in
`src/server.py`, so a caller can tell an unimplemented path from a rejected
login from a genuine upstream failure.
"""


class HometaxError(Exception):
    """Base class for Hometax integration failures."""

    error_code = "HOMETAX_ERROR"


class HometaxNotEnabledError(HometaxError):
    """
    Raised when a Hometax path is invoked while live mode is off.

    Live mode is off by default because the scraping selectors have never been
    validated against the real site. Refusing here is the point: the previous
    behaviour was to return a fabricated success with a made-up 국세청 승인번호.
    """

    error_code = "HOMETAX_NOT_ENABLED"


class HometaxNotImplementedError(HometaxError):
    """Raised for a Hometax capability that has no implementation at all."""

    error_code = "NOT_IMPLEMENTED"


class HometaxLoginError(HometaxError):
    """Raised when authentication against Hometax did not succeed."""

    error_code = "LOGIN_FAILED"


class HometaxSessionError(HometaxError):
    """Raised when a session is unknown, expired, or belongs to another tenant."""

    error_code = "SESSION_INVALID"


class HometaxScrapeError(HometaxError):
    """
    Raised when a page did not yield the data we expected.

    Distinct from "no results": a search that fails must not be reported as a
    period with zero 세금계산서, because that number feeds 부가가치세 신고.
    """

    error_code = "SCRAPE_FAILED"


class HometaxResultUndeterminedError(HometaxError):
    """
    Raised when an operation's outcome could not be established.

    Applies to issuance in particular: if we cannot confirm the 국세청 승인번호,
    the filing must be treated as unknown and reconciled, never as successful.
    """

    error_code = "RESULT_UNDETERMINED"
