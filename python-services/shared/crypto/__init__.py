"""
Cryptographic utilities for Korean government system integration.

Provides SEED, ARIA encryption algorithms and PKCS#7 signing required for
communication with Korean government systems (Hometax, 4대보험 EDI).
"""
from .seed import AESCBCCipher, SEEDCipher, SEEDNotImplementedError, generate_aes_key
from .aria import ARIACipher, ARIAModeCBC
from .pkcs7 import PKCS7Padding, PKCS7Signature, generate_test_keypair
from .utils import (
    generate_key,
    derive_key,
    generate_iv,
    bytes_to_hex,
    hex_to_bytes,
    bytes_to_base64,
    base64_to_bytes,
    constant_time_compare,
    load_symmetric_key,
    assert_strong_key,
    WeakKeyError,
    KeyDerivation,
    MessageAuthentication,
)

__all__ = [
    # AES (honestly named) and the SEED placeholder that refuses to run
    "AESCBCCipher",
    "SEEDCipher",
    "SEEDNotImplementedError",
    "generate_aes_key",
    # ARIA
    "ARIACipher",
    "ARIAModeCBC",
    # PKCS7
    "PKCS7Padding",
    "PKCS7Signature",
    "generate_test_keypair",
    # Utilities
    "generate_key",
    "derive_key",
    "generate_iv",
    "bytes_to_hex",
    "hex_to_bytes",
    "bytes_to_base64",
    "base64_to_bytes",
    "constant_time_compare",
    "load_symmetric_key",
    "assert_strong_key",
    "WeakKeyError",
    "KeyDerivation",
    "MessageAuthentication",
]
