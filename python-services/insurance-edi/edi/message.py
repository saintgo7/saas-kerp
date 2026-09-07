"""
EDI Message Structure

Defines the structure of EDI messages used for 4대보험 communication.

EDI Message Format:
+------------------+------------------+------------------+
|  Message Header  |   Data Header    |    Data Body     |
|   (Fixed 100B)   |   (Variable)     |   (Variable)     |
+------------------+------------------+------------------+
"""
from dataclasses import dataclass, field
from datetime import datetime
from typing import Optional, List
from enum import Enum


class MessageType(Enum):
    """EDI message types."""

    # Request types
    REQUEST_SUBMIT = "S"       # Submit document
    REQUEST_QUERY = "Q"        # Query status
    REQUEST_DOWNLOAD = "D"     # Download result
    REQUEST_CANCEL = "C"       # Cancel submission

    # Response types
    RESPONSE_SUCCESS = "0"     # Success
    RESPONSE_ERROR = "1"       # Error
    RESPONSE_PENDING = "2"     # Processing


class InsuranceType(Enum):
    """Insurance provider types."""

    NPS = "10"      # 국민연금 (National Pension Service)
    NHIS = "20"     # 건강보험 (National Health Insurance Service)
    EI = "30"       # 고용보험 (Employment Insurance)
    WCI = "40"      # 산재보험 (Workers' Compensation Insurance)


class DocumentType(Enum):
    """Document types for submissions."""

    # NPS documents
    NPS_ACQUISITION = "1001"           # 취득신고
    NPS_LOSS = "1002"                  # 상실신고
    NPS_CHANGE = "1003"                # 내용변경
    NPS_MONTHLY_REPORT = "1004"        # 월별납부내역

    # NHIS documents
    NHIS_ACQUISITION = "2001"          # 취득신고
    NHIS_LOSS = "2002"                 # 상실신고
    NHIS_CHANGE = "2003"               # 보수월액변경
    NHIS_DEPENDENT = "2004"            # 피부양자신고

    # EI/WCI documents
    EI_ACQUISITION = "3001"            # 고용취득신고
    EI_LOSS = "3002"                   # 고용상실신고
    WCI_ACQUISITION = "4001"           # 산재취득신고
    WCI_LOSS = "4002"                  # 산재상실신고


@dataclass
class EDIHeader:
    """
    EDI Message Header (100 bytes fixed).

    Contains metadata about the EDI message including sender/receiver
    information, message type, and timestamps.
    """

    # Message identification
    message_id: str = ""               # 전문ID (20 bytes)
    message_type: MessageType = MessageType.REQUEST_SUBMIT
    message_version: str = "1.0"       # 버전 (4 bytes)

    # Sender information
    sender_id: str = ""                # 송신자ID (13 bytes, 사업장번호)
    sender_name: str = ""              # 송신자명 (30 bytes)

    # Receiver information
    insurance_type: InsuranceType = InsuranceType.NPS
    receiver_code: str = ""            # 수신기관코드 (3 bytes)

    # Timestamps
    send_datetime: datetime = field(default_factory=datetime.now)
    sequence_no: int = 1               # 일련번호 (4 bytes)

    # Security
    encrypted: bool = True
    signed: bool = True

    # Field layout: (name, byte offset, byte width). The header is a fixed
    # 100-BYTE record, so every offset below is a byte offset, never a
    # character offset. euc-kr encodes each Hangul syllable as 2 bytes, so
    # padding in characters and slicing in bytes (or the reverse) shifts every
    # later field -- which is what broke every message with a Korean 송신자명.
    _LAYOUT = (
        ("message_id", 0, 20),
        ("message_type", 20, 1),
        ("message_version", 21, 4),
        ("sender_id", 25, 13),
        ("sender_name", 38, 30),
        ("insurance_type", 68, 2),
        ("receiver_code", 70, 3),
        ("send_datetime", 73, 14),
        ("sequence_no", 87, 4),
        ("encrypted", 91, 1),
        ("signed", 92, 1),
    )

    HEADER_SIZE = 100
    ENCODING = "euc-kr"

    @classmethod
    def _pad_field(cls, value: str, width: int, *, name: str) -> bytes:
        """
        Encode a text field to exactly `width` bytes, left-aligned, space-padded.

        Truncation happens on a character boundary so a multi-byte character is
        never cut in half (which would make the field undecodable).

        Args:
            value: Field text
            width: Field width in bytes
            name: Field name, used in the error message

        Returns:
            Exactly `width` bytes
        """
        encoded = value.encode(cls.ENCODING, errors="strict")

        if len(encoded) > width:
            # Drop trailing characters until the encoded form fits.
            truncated = value
            while truncated and len(truncated.encode(cls.ENCODING)) > width:
                truncated = truncated[:-1]
            encoded = truncated.encode(cls.ENCODING)

        return encoded.ljust(width, b" ")

    def to_bytes(self) -> bytes:
        """
        Serialize header to the 100-byte fixed format.

        Returns:
            Exactly 100 bytes

        Raises:
            UnicodeEncodeError: If a field contains characters euc-kr cannot represent
        """
        parts = [
            self._pad_field(self.message_id, 20, name="message_id"),
            self._pad_field(self.message_type.value, 1, name="message_type"),
            self._pad_field(self.message_version, 4, name="message_version"),
            self._pad_field(self.sender_id, 13, name="sender_id"),
            self._pad_field(self.sender_name, 30, name="sender_name"),
            self._pad_field(self.insurance_type.value, 2, name="insurance_type"),
            self._pad_field(self.receiver_code, 3, name="receiver_code"),
            self.send_datetime.strftime("%Y%m%d%H%M%S").encode("ascii"),  # 14 bytes
            str(self.sequence_no).zfill(4)[:4].encode("ascii"),
            b"Y" if self.encrypted else b"N",
            b"Y" if self.signed else b"N",
        ]

        header = b"".join(parts)
        # 93 bytes of fields, padded out to the fixed 100-byte record.
        return header.ljust(self.HEADER_SIZE, b" ")[:self.HEADER_SIZE]

    @classmethod
    def from_bytes(cls, data: bytes) -> "EDIHeader":
        """
        Parse header from 100-byte data.

        Args:
            data: 100-byte header data

        Returns:
            Parsed EDIHeader

        Raises:
            ValueError: If the header is short or a field is malformed
        """
        if len(data) < cls.HEADER_SIZE:
            raise ValueError(
                f"Header must be {cls.HEADER_SIZE} bytes, got {len(data)}"
            )

        def field(offset: int, width: int) -> str:
            # Slice BYTES, then decode. `errors="ignore"` is confined to a
            # single field so a damaged 송신자명 cannot shift the fields after it.
            return data[offset:offset + width].decode(cls.ENCODING, errors="ignore").strip()

        raw_type = field(20, 1)
        raw_insurance = field(68, 2)
        raw_datetime = field(73, 14)
        raw_sequence = field(87, 4)

        try:
            message_type = MessageType(raw_type) if raw_type else MessageType.REQUEST_SUBMIT
        except ValueError as exc:
            raise ValueError(f"Unknown message_type {raw_type!r} in header") from exc

        try:
            insurance_type = InsuranceType(raw_insurance) if raw_insurance else InsuranceType.NPS
        except ValueError as exc:
            raise ValueError(f"Unknown insurance_type {raw_insurance!r} in header") from exc

        if raw_datetime:
            try:
                send_datetime = datetime.strptime(raw_datetime, "%Y%m%d%H%M%S")
            except ValueError as exc:
                raise ValueError(
                    f"Malformed send_datetime {raw_datetime!r} in header"
                ) from exc
        else:
            send_datetime = datetime.now()

        return cls(
            message_id=field(0, 20),
            message_type=message_type,
            message_version=field(21, 4),
            sender_id=field(25, 13),
            sender_name=field(38, 30),
            insurance_type=insurance_type,
            receiver_code=field(70, 3),
            send_datetime=send_datetime,
            sequence_no=int(raw_sequence) if raw_sequence.isdigit() else 1,
            encrypted=data[91:92] == b"Y",
            signed=data[92:93] == b"Y",
        )


@dataclass
class EDIBody:
    """
    EDI Message Body.

    Contains the actual document data being transmitted.
    """

    # Document info
    document_type: DocumentType = DocumentType.NPS_ACQUISITION
    document_count: int = 1

    # Company info
    company_id: str = ""           # 사업장관리번호
    company_name: str = ""         # 사업장명
    business_no: str = ""          # 사업자등록번호 (10 digits)

    # Content
    records: List[dict] = field(default_factory=list)
    raw_data: bytes = b""

    FIELD_SEPARATOR = "|"
    RECORD_SEPARATOR = "\n"
    ESCAPE = "\\"

    # Escape map applied to every serialized value. Without it, a 직원 성명 of
    # "홍길동|9001011234567|20240101" splits one record into several fields, and
    # a newline in any value turns one 취득신고 into two -- silently changing what
    # is filed with the 공단.
    _ESCAPES = (
        (ESCAPE, ESCAPE + ESCAPE),  # must be first
        (FIELD_SEPARATOR, ESCAPE + "p"),
        ("\n", ESCAPE + "n"),
        ("\r", ESCAPE + "r"),
    )

    @classmethod
    def escape_value(cls, value: object) -> str:
        """
        Escape a field value so it cannot inject separators.

        Args:
            value: Field value of any type; coerced with str()

        Returns:
            Escaped string safe to place between separators
        """
        text = str(value)
        for raw, escaped in cls._ESCAPES:
            text = text.replace(raw, escaped)
        return text

    @classmethod
    def unescape_value(cls, text: str) -> str:
        """
        Reverse `escape_value`.

        Args:
            text: Escaped field text

        Returns:
            Original value
        """
        out: list[str] = []
        i = 0
        while i < len(text):
            ch = text[i]
            if ch == cls.ESCAPE and i + 1 < len(text):
                nxt = text[i + 1]
                if nxt == "p":
                    out.append(cls.FIELD_SEPARATOR)
                elif nxt == "n":
                    out.append("\n")
                elif nxt == "r":
                    out.append("\r")
                elif nxt == cls.ESCAPE:
                    out.append(cls.ESCAPE)
                else:
                    out.append(nxt)
                i += 2
                continue
            out.append(ch)
            i += 1
        return "".join(out)

    @classmethod
    def _split_escaped(cls, line: str) -> list[str]:
        """Split a line on unescaped field separators."""
        fields: list[str] = []
        current: list[str] = []
        i = 0
        while i < len(line):
            ch = line[i]
            if ch == cls.ESCAPE and i + 1 < len(line):
                current.append(ch)
                current.append(line[i + 1])
                i += 2
                continue
            if ch == cls.FIELD_SEPARATOR:
                fields.append("".join(current))
                current = []
                i += 1
                continue
            current.append(ch)
            i += 1
        fields.append("".join(current))
        return [cls.unescape_value(f) for f in fields]

    def to_bytes(self, encoding: str = "euc-kr") -> bytes:
        """
        Serialize body to bytes.

        Args:
            encoding: Character encoding (default: euc-kr for Korean)

        Returns:
            Serialized body bytes

        Raises:
            ValueError: If `document_count` disagrees with the record count
        """
        if self.raw_data:
            return self.raw_data

        # The declared count and the actual number of records must agree, or
        # the 공단 receives a different number of filings than we believe we sent.
        if self.records and self.document_count != len(self.records):
            raise ValueError(
                f"document_count ({self.document_count}) does not match the number "
                f"of records ({len(self.records)})"
            )

        lines = []

        # Document header
        doc_header = self.FIELD_SEPARATOR.join(
            self.escape_value(v)
            for v in (
                self.document_type.value,
                self.document_count,
                self.company_id,
                self.business_no,
            )
        )
        lines.append(doc_header)

        # Records
        for record in self.records:
            lines.append(
                self.FIELD_SEPARATOR.join(self.escape_value(v) for v in record.values())
            )

        content = self.RECORD_SEPARATOR.join(lines)
        return content.encode(encoding)

    @classmethod
    def from_bytes(cls, data: bytes, encoding: str = "euc-kr") -> "EDIBody":
        """
        Parse body from bytes.

        Args:
            data: Body data bytes
            encoding: Character encoding

        Returns:
            Parsed EDIBody

        Raises:
            ValueError: If the body cannot be decoded or its header is malformed.
                A parse failure must surface: returning an empty body with a
                default document_type made a broken response look like a valid
                NPS acquisition.
        """
        try:
            text = data.decode(encoding)
        except UnicodeDecodeError as exc:
            raise ValueError(f"EDI body is not valid {encoding}: {exc}") from exc

        stripped = text.strip()
        if not stripped:
            # Genuinely empty body -- distinct from a malformed one.
            return cls(raw_data=data, records=[])

        lines = stripped.split(cls.RECORD_SEPARATOR)
        header_parts = cls._split_escaped(lines[0])

        if len(header_parts) < 4:
            raise ValueError(
                f"EDI body header needs 4 fields, got {len(header_parts)}"
            )

        try:
            document_type = DocumentType(header_parts[0])
        except ValueError as exc:
            raise ValueError(
                f"Unknown document_type {header_parts[0]!r} in EDI body"
            ) from exc

        try:
            document_count = int(header_parts[1])
        except ValueError as exc:
            raise ValueError(
                f"Malformed document_count {header_parts[1]!r} in EDI body"
            ) from exc

        body = cls(
            document_type=document_type,
            document_count=document_count,
            company_id=header_parts[2],
            business_no=header_parts[3],
            raw_data=data,
        )

        for line in lines[1:]:
            if line.strip():
                values = cls._split_escaped(line)
                body.records.append({f"field_{i}": v for i, v in enumerate(values)})

        return body

    def response_status(self) -> tuple[str, str]:
        """
        Extract the response code and message from a parsed response body.

        Response bodies carry `RSP|<code>|<message>` as their first line. Before
        this existed, `EDIMessage.response_code` was never populated, so every
        submission was scored as "not success" no matter what came back.

        Returns:
            Tuple of (response_code, response_message); ("", "") when the body
            carries no response line.
        """
        if not self.raw_data:
            return "", ""

        try:
            text = self.raw_data.decode("euc-kr", errors="ignore")
        except Exception:
            return "", ""

        first_line = text.strip().split(self.RECORD_SEPARATOR)[0]
        parts = self._split_escaped(first_line)

        if len(parts) >= 2 and parts[0] == "RSP":
            return parts[1], parts[2] if len(parts) > 2 else ""

        return "", ""


@dataclass
class EDIMessage:
    """
    Complete EDI Message.

    Combines header and body for a complete EDI transmission.
    """

    header: EDIHeader = field(default_factory=EDIHeader)
    body: EDIBody = field(default_factory=EDIBody)

    # Response fields (populated on receive)
    response_code: str = ""
    response_message: str = ""
    response_data: Optional[bytes] = None

    def to_bytes(self) -> bytes:
        """
        Serialize complete message to bytes.

        Returns:
            Complete message bytes (header + body)
        """
        header_bytes = self.header.to_bytes()
        body_bytes = self.body.to_bytes()

        # Add body length prefix (4 bytes)
        body_len = len(body_bytes)
        length_prefix = body_len.to_bytes(4, byteorder="big")

        return header_bytes + length_prefix + body_bytes

    @classmethod
    def from_bytes(cls, data: bytes) -> "EDIMessage":
        """
        Parse complete message from bytes.

        Args:
            data: Complete message bytes

        Returns:
            Parsed EDIMessage
        """
        if len(data) < 104:  # 100 (header) + 4 (length)
            raise ValueError("Message too short")

        header = EDIHeader.from_bytes(data[:100])
        body_len = int.from_bytes(data[100:104], byteorder="big")

        body_bytes = data[104:104 + body_len]
        if len(body_bytes) != body_len:
            raise ValueError(
                f"Truncated message: declared {body_len} body bytes, "
                f"got {len(body_bytes)}"
            )

        body = EDIBody.from_bytes(body_bytes)

        return cls(header=header, body=body)

    @classmethod
    def create_submit_message(
        cls,
        sender_id: str,
        insurance_type: InsuranceType,
        document_type: DocumentType,
        records: List[dict],
        company_id: str,
        business_no: str,
        idempotency_key: Optional[str] = None,
    ) -> "EDIMessage":
        """
        Create a document submission message.

        Args:
            sender_id: Sender business ID
            insurance_type: Target insurance provider
            document_type: Type of document
            records: Document records
            company_id: Company management number
            business_no: Business registration number
            idempotency_key: Caller-supplied key reused as the message ID. Pass
                the same key when retrying a submission: a fresh uuid4 per
                attempt gives the 공단 no way to recognise a retry, which is how
                one 취득신고 becomes two (error code 2001, 중복 신고).

        Returns:
            Prepared EDIMessage
        """
        import uuid

        header = EDIHeader(
            message_id=(idempotency_key or str(uuid.uuid4()))[:20],
            message_type=MessageType.REQUEST_SUBMIT,
            sender_id=sender_id,
            insurance_type=insurance_type,
        )

        body = EDIBody(
            document_type=document_type,
            document_count=len(records),
            company_id=company_id,
            business_no=business_no,
            records=records,
        )

        return cls(header=header, body=body)

    @classmethod
    def create_query_message(
        cls,
        sender_id: str,
        insurance_type: InsuranceType,
        reference_id: str,
    ) -> "EDIMessage":
        """
        Create a status query message.

        Args:
            sender_id: Sender business ID
            insurance_type: Target insurance provider
            reference_id: Reference ID of previous submission

        Returns:
            Prepared query EDIMessage
        """
        import uuid

        header = EDIHeader(
            message_id=str(uuid.uuid4())[:20],
            message_type=MessageType.REQUEST_QUERY,
            sender_id=sender_id,
            insurance_type=insurance_type,
        )

        body = EDIBody(
            raw_data=f"REF|{reference_id}".encode("euc-kr"),
        )

        return cls(header=header, body=body)
