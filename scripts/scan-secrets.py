#!/usr/bin/env python3
"""Repository secret scanner.

Replaces the previous check, which excluded ``*.example`` files outright and so
could never see the production SSH password committed in
``docs/.env_ssh.example``.  Here ``.example`` files ARE scanned; what makes a
hit acceptable is the *value* looking like a placeholder, not the file name.

Usage::

    scripts/scan-secrets.py            # scan every git-tracked file
    scripts/scan-secrets.py --staged   # scan only staged files
    scripts/scan-secrets.py --self-test

Exit codes: 0 = clean, 1 = candidate secret found, 2 = self-test failure.

Written in Python rather than shell because the shell version needed ``grep -P``
and ``mapfile``, neither of which exists on the macOS default toolchain the
pre-push hook runs under.
"""

from __future__ import annotations

import os
import re
import subprocess
import sys

# Binary blobs and generated output carry no reviewable secrets and only add
# noise.  Office documents are skipped here on purpose: a zipped OOXML body
# defeats every line-oriented scanner, so they are blocked from the repository
# entirely by .gitignore instead.
SKIP_RE = re.compile(
    r"(^|/)(node_modules|vendor|dist|build|coverage|graphify-out|\.git)/"
    r"|\.(png|jpe?g|gif|svg|ico|woff2?|ttf|eot|pdf|zip|gz|tgz|jar|class|so|"
    r"dylib|exe|lock|sum|docx|xlsx|pptx)$"
    r"|package-lock\.json$"
)

# Test fixtures legitimately hold throwaway credentials.  This exclusion is
# narrow and by file role - unlike the `.example` exclusion it replaces, which
# hid the very files where deployment credentials are pasted.
TEST_RE = re.compile(
    r"(^|/)(tests?|__tests__|testdata|e2e)/"
    r"|_test\.(go|py|ts|tsx|js)$"
    r"|\.(test|spec)\.(ts|tsx|js|jsx)$"
    r"|(^|/)test_[^/]+\.py$"
)

# Credential-ish keyword, matched anywhere inside an identifier so that
# ID_PASSWORD, MINIO_ROOT_PASSWORD and dbPassword are all covered.
KEYWORD = (
    r"pass(?:word|wd)|secret|api[_-]?key|apikey|"
    r"access[_-]?token|auth[_-]?token|private[_-]?key|client[_-]?secret|"
    r"_pass\b"
)

IDENT = r"[A-Za-z0-9_.\-]"

# A: key assigned a QUOTED string literal.
QUOTED_RE = re.compile(
    rf"(?i)(?P<key>{IDENT}*(?:{KEYWORD}){IDENT}*)\s*[:=]\s*"
    r"(?P<q>[\"'])(?P<value>[^\"'\n]{6,})(?P=q)"
)

# B: env/shell/dotenv style assignment - key at the start of the line.
# This is the shape that leaked the SSH password.  Applied ONLY to
# configuration-shaped files; in .go/.ts/.py sources the same shape is a struct
# field or a function parameter, not a credential.
ENV_RE = re.compile(
    rf"(?im)^\s*(?:export\s+|-\s+)?(?P<key>[A-Za-z0-9_]*(?:{KEYWORD})[A-Za-z0-9_]*)"
    r"\s*[:=]\s*(?P<value>[^\s#]{6,})\s*(?:#.*)?$"
)

CONFIG_FILE_RE = re.compile(
    r"(^|/)\.env(\.|$)|(^|/)\.env_[A-Za-z0-9_]+"
    r"|\.(sh|bash|zsh|ya?ml|conf|cfg|ini|toml|properties|md|env|tf|tfvars)$"
    r"|(^|/)(Dockerfile|Makefile)[^/]*$"
)

# A value is acceptable when it is obviously a placeholder, an environment
# lookup, or a reference to another symbol - rather than a literal credential.
# Extend this list; never exclude a whole file.
PLACEHOLDER_RE = re.compile(
    r"(?i)("
    r"change[-_ ]?me|change[-_ ]?this|change[-_ ]?in[-_ ]?production|"
    r"your[-_. ]|<[^>]*>|\$\{|\$\(|\$[A-Za-z_]|%s|%v|\{\{|"
    r"x{3,}|placeholder|example|dummy|sample|redacted|todo|fixme|"
    r"^(null|none|nil|true|false|empty|str|bytes|int|bool)$|"
    r"fake|test[-_]?only|not[-_]?a[-_]?real|ci[-_]placeholder|verify[-_]only|"
    r"localhost|process\.env|os\.getenv|getenv|viper\.|"
    r"^[A-Za-z_][A-Za-z0-9_]*\.[A-Za-z_]"      # obj.Field - a reference
    r"|\("                                      # function call / cast
    r"|^[a-z][a-z_]*$"                           # bare snake_case identifier or enum value
    r"|(^|[-_])test([-_]|$)"                     # explicitly-named test credential
    r")"
)

# Self-test corpus: (text, expect_finding).  `scan-secrets.py --self-test`
# proves the scanner still catches the class of defect it was written for.
SELF_TEST = [
    # (path, line, expect_finding)
    ("docs/.env_ssh.example", "WSL_PASS=tel3phone9088\n", True),
    ("docs/.env_ssh.example", "WSL_PASS=your-password-here\n", False),
    ("docs/dev-log/PLAN.md", "WSL_PASS=<your-ssh-password>\n", False),
    ("deployments/docker/.env.example", "JWT_SECRET=change-me-in-production\n", False),
    ("deployments/docker/.env.example", "JWT_SECRET=Wq8s7Xn2LmZ4pR6tYb1Vc3Ed5Gf7Hj9K\n", True),
    ("deployments/docker/docker-compose.yml", '  password: "hunter2secret"\n', True),
    ("deployments/docker/docker-compose.yml", '  password: "${DB_PASSWORD}"\n', False),
    ("deployments/docker/docker-compose.yml", "  - MINIO_ROOT_PASSWORD=S3cur3xMinioRoot\n", True),
    ("deployments/docker/.env.example", "MINIO_ROOT_PASSWORD=minio123_change_in_production\n", False),
    (".github/workflows/ci.yml", "POSTGRES_PASSWORD: kerp_test_password\n", False),
    ("python-services/shared/crypto/utils.py", "def f(password: str = None):\n", False),
    ("python-services/shared/crypto/pkcs7.py", "    password=password,\n", False),
    ("web/src/services/api.ts", "accessToken: getAccessToken(),\n", False),
    ("python-services/tax-scraper/src/hometax/models.py", 'ID_PASSWORD = "id_password"\n', False),
    ("internal/domain/user.go", "PasswordHash: string(hash),\n", False),
    ("web/src/constants/index.ts", 'passwordMatch: "비밀번호가 일치하지 않습니다.",\n', False),
]


def looks_like_prose(value: str) -> bool:
    """Prose, not a credential.

    UI strings and comments routinely sit next to a `password`-ish key.  A
    credential never contains a space or a non-ASCII character; a Korean error
    message always does.
    """
    return any(ord(ch) > 127 for ch in value) or " " in value


def scan_text(text: str, path: str = "<mem>") -> list[str]:
    findings = []
    rules = [QUOTED_RE]
    if CONFIG_FILE_RE.search(path):
        rules.append(ENV_RE)
    for lineno, line in enumerate(text.splitlines(), 1):
        if len(line) > 4000:
            continue
        for rx in rules:
            for match in rx.finditer(line):
                value = match.group("value")
                if PLACEHOLDER_RE.search(value) or looks_like_prose(value):
                    continue
                findings.append(f"{path}:{lineno}: {line.strip()[:160]}")
                break
    return findings


def self_test() -> int:
    bad = 0
    for path, text, expect in SELF_TEST:
        got = bool(scan_text(text, path))
        status = "ok " if got == expect else "FAIL"
        if got != expect:
            bad += 1
        print(f"  [{status}] expect={expect!s:5} got={got!s:5} {path} :: {text.strip()!r}")
    if bad:
        print(f"self-test: {bad} case(s) failed")
        return 2
    print(f"self-test: all {len(SELF_TEST)} cases passed")
    return 0


def tracked_files(staged: bool) -> list[str]:
    cmd = (
        ["git", "diff", "--cached", "--name-only", "--diff-filter=ACM"]
        if staged
        else ["git", "ls-files"]
    )
    out = subprocess.run(cmd, capture_output=True, text=True, check=True).stdout
    return [line for line in out.splitlines() if line]


def main() -> int:
    args = sys.argv[1:]
    if "--self-test" in args:
        return self_test()

    staged = "--staged" in args

    root = subprocess.run(
        ["git", "rev-parse", "--show-toplevel"],
        capture_output=True, text=True, check=True,
    ).stdout.strip()
    os.chdir(root)

    self_path = os.path.relpath(os.path.abspath(__file__), root)

    findings: list[str] = []
    considered = 0

    for path in tracked_files(staged):
        if SKIP_RE.search(path) or TEST_RE.search(path) or path == self_path:
            continue
        if not os.path.isfile(path):
            continue
        considered += 1
        try:
            with open(path, "r", encoding="utf-8", errors="ignore") as fh:
                findings.extend(scan_text(fh.read(), path))
        except OSError:
            continue

    if findings:
        print("ERROR: candidate secrets found (value does not look like a placeholder):")
        print()
        for f in findings:
            print(f"  {f}")
        print()
        print("If a hit is a false positive, make the value obviously a placeholder")
        print("(change-me-in-production, <your-value-here>, ${VAR}) rather than")
        print("excluding the file from the scan.")
        return 1

    print(f"Secret scan clean ({considered} files considered).")
    return 0


if __name__ == "__main__":
    sys.exit(main())
