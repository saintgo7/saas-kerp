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

    def to_bytes(self) -> bytes:
        """
        Serialize header to 100-byte fixed format.

        Returns:
            100-byte header
        """
        # Format each field with proper padding
        parts = [
            self.message_id.ljust(20)[:20],
            self.message_type.value.ljust(1)[:1],
            self.message_version.ljust(4)[:4],
            self.sender_id.ljust(13)[:13],
            self.sender_name.encode("euc-kr").ljust(30)[:30].decode("euc-kr", errors="ignore"),
            self.insurance_type.value.ljust(2)[:2],
            self.receiver_code.ljust(3)[:3],
            self.send_datetime.strftime("%Y%m%d%H%M%S"),  # 14 bytes
            str(self.sequence_no).zfill(4)[:4],
            "Y" if self.encrypted else "N",
            "Y" if self.signed else "N",
        ]

        header_str = "".join(parts)
        # Pad to exactly 100 bytes
        return header_str.encode("euc-kr").ljust(100)[:100]

    @classmethod
    def from_bytes(cls, data: bytes) -> "EDIHeader":
        """
        Parse header from 100-byte data.

        Args:
            data: 100-byte header data

        Returns:
            Parsed EDIHeader
        """
        text = data.decode("euc-kr", errors="ignore")

        return cls(
            message_id=text[0:20].strip(),
            message_type=MessageType(text[20:21].strip() or "S"),
            message_version=text[21:25].strip(),
            sender_id=text[25:38].strip(),
            sender_name=text[38:68].strip(),
            insurance_type=InsuranceType(text[68:70].strip() or "10"),
            receiver_code=text[70:73].strip(),
            send_datetime=datetime.strptime(text[73:87], "%Y%m%d%H%M%S") if text[73:87].strip() else datetime.now(),
            sequence_no=int(text[87:91]) if text[87:91].strip().isdigit() else 1,
            encrypted=text[91:92] == "Y",
            signed=text[92:93] == "Y",
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

    def to_bytes(self, encoding: str = "euc-kr") -> bytes:
        """
        Serialize body to bytes.

        Args:
            encoding: Character encoding (default: euc-kr for Korean)

        Returns:
            Serialized body bytes
        """
        if self.raw_data:
            return self.raw_data

        # Build body content
        lines = []

        # Document header
        doc_header = f"{self.document_type.value}|{self.document_count}|{self.company_id}|{self.business_no}"
        lines.append(doc_header)

        # Records
        for record in self.records:
            record_line = "|".join(str(v) for v in record.values())
            lines.append(record_line)

        content = "\n".join(lines)
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
        """
        try:
            text = data.decode(encoding)
            lines = text.strip().split("\n")

            if not lines:
                return cls(raw_data=data)

            # Parse document header
            header_parts = lines[0].split("|")

            body = cls(
                document_type=DocumentType(header_parts[0]) if len(header_parts) > 0 else DocumentType.NPS_ACQUISITION,
                document_count=int(header_parts[1]) if len(header_parts) > 1 else 1,
                company_id=header_parts[2] if len(header_parts) > 2 else "",
                business_no=header_parts[3] if len(header_parts) > 3 else "",
                raw_data=data,
            )

            # Parse records
            for line in lines[1:]:
                if line.strip():
                    values = line.split("|")
                    body.records.append({f"field_{i}": v for i, v in enumerate(values)})

            return body

        except Exception:
            return cls(raw_data=data)


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
        body = EDIBody.from_bytes(data[104:104 + body_len])

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

        Returns:
            Prepared EDIMessage
        """
        import uuid

        header = EDIHeader(
            message_id=str(uuid.uuid4())[:20],
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
