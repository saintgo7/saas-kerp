const { Document, Packer, Paragraph, TextRun, Table, TableRow, TableCell,
        Header, Footer, AlignmentType, LevelFormat, HeadingLevel,
        BorderStyle, WidthType, ShadingType, VerticalAlign, PageNumber, PageBreak } = require('docx');
const fs = require('fs');

const tableBorder = { style: BorderStyle.SINGLE, size: 1, color: "CCCCCC" };
const cellBorders = { top: tableBorder, bottom: tableBorder, left: tableBorder, right: tableBorder };

const doc = new Document({
  styles: {
    default: { document: { run: { font: "맑은 고딕", size: 22 } } },
    paragraphStyles: [
      { id: "Title", name: "Title", basedOn: "Normal",
        run: { size: 48, bold: true, color: "1F4E79", font: "맑은 고딕" },
        paragraph: { spacing: { before: 240, after: 240 }, alignment: AlignmentType.CENTER } },
      { id: "Heading1", name: "Heading 1", basedOn: "Normal", next: "Normal", quickFormat: true,
        run: { size: 32, bold: true, color: "2E74B5", font: "맑은 고딕" },
        paragraph: { spacing: { before: 360, after: 180 }, outlineLevel: 0 } },
      { id: "Heading2", name: "Heading 2", basedOn: "Normal", next: "Normal", quickFormat: true,
        run: { size: 26, bold: true, color: "404040", font: "맑은 고딕" },
        paragraph: { spacing: { before: 240, after: 120 }, outlineLevel: 1 } },
      { id: "Heading3", name: "Heading 3", basedOn: "Normal", next: "Normal", quickFormat: true,
        run: { size: 24, bold: true, color: "595959", font: "맑은 고딕" },
        paragraph: { spacing: { before: 180, after: 100 }, outlineLevel: 2 } }
    ]
  },
  numbering: {
    config: [
      { reference: "bullet-list",
        levels: [{ level: 0, format: LevelFormat.BULLET, text: "•", alignment: AlignmentType.LEFT,
          style: { paragraph: { indent: { left: 720, hanging: 360 } } } }] },
      { reference: "numbered-list",
        levels: [{ level: 0, format: LevelFormat.DECIMAL, text: "%1.", alignment: AlignmentType.LEFT,
          style: { paragraph: { indent: { left: 720, hanging: 360 } } } }] }
    ]
  },
  sections: [{
    properties: {
      page: { margin: { top: 1440, right: 1440, bottom: 1440, left: 1440 } }
    },
    headers: {
      default: new Header({ children: [new Paragraph({
        alignment: AlignmentType.RIGHT,
        children: [new TextRun({ text: "K-ERP SaaS v0.2 Development Plan", size: 18, color: "808080" })]
      })] })
    },
    footers: {
      default: new Footer({ children: [new Paragraph({
        alignment: AlignmentType.CENTER,
        children: [new TextRun({ text: "Page ", size: 18 }), new TextRun({ children: [PageNumber.CURRENT], size: 18 }), new TextRun({ text: " / ", size: 18 }), new TextRun({ children: [PageNumber.TOTAL_PAGES], size: 18 })]
      })] })
    },
    children: [
      // Cover
      new Paragraph({ spacing: { before: 2000 } }),
      new Paragraph({ heading: HeadingLevel.TITLE, children: [new TextRun("K-ERP SaaS Platform")] }),
      new Paragraph({ alignment: AlignmentType.CENTER, spacing: { after: 600 }, children: [new TextRun({ text: "8-Phase Parallel Development Plan", size: 40, bold: true, color: "404040" })] }),
      new Paragraph({ alignment: AlignmentType.CENTER, spacing: { before: 1200 }, children: [new TextRun({ text: "Go + Python Hybrid Architecture", size: 28, italics: true, color: "808080" })] }),
      new Paragraph({ alignment: AlignmentType.CENTER, spacing: { before: 800 }, children: [new TextRun({ text: "Version: 0.2.0", size: 22 })] }),
      new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "Date: 2026-01-17", size: 22 })] }),

      new Paragraph({ children: [new PageBreak()] }),

      // 1. Overview
      new Paragraph({ heading: HeadingLevel.HEADING_1, children: [new TextRun("1. Project Overview")] }),

      new Paragraph({ heading: HeadingLevel.HEADING_2, children: [new TextRun("1.1 Project Summary")] }),
      new Table({
        columnWidths: [3000, 6000],
        rows: [
          new TableRow({ tableHeader: true, children: [
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "Item", bold: true, color: "FFFFFF" })] })] }),
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "Description", bold: true, color: "FFFFFF" })] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun({ text: "Project", bold: true })] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("K-ERP SaaS Platform")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun({ text: "Target", bold: true })] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Korean SMBs (Revenue 1B~100B KRW)")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun({ text: "Architecture", bold: true })] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Go + Python Hybrid")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun({ text: "Duration", bold: true })] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("12 months (MVP: 6 months)")] })] })
          ]})
        ]
      }),

      new Paragraph({ heading: HeadingLevel.HEADING_2, spacing: { before: 400 }, children: [new TextRun("1.2 Tech Stack")] }),
      new Table({
        columnWidths: [3000, 6000],
        rows: [
          new TableRow({ tableHeader: true, children: [
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "Layer", bold: true, color: "FFFFFF" })] })] }),
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "Technology", bold: true, color: "FFFFFF" })] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Backend")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Go 1.22+ (Gin) + Python 3.11+ (gRPC)")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Frontend")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("React 18 + TypeScript + Vite")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Database")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("PostgreSQL 16 + Redis 7 + NATS JetStream")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Container")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Docker + Kubernetes")] })] })
          ]})
        ]
      }),

      new Paragraph({ children: [new PageBreak()] }),

      // 2. 8 Phase Development
      new Paragraph({ heading: HeadingLevel.HEADING_1, children: [new TextRun("2. 8 Phase Parallel Development")] }),

      new Table({
        columnWidths: [1200, 3000, 1500, 1500, 1800],
        rows: [
          new TableRow({ tableHeader: true, children: [
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "Phase", bold: true, color: "FFFFFF" })] })] }),
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "Name", bold: true, color: "FFFFFF" })] })] }),
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "Agent", bold: true, color: "FFFFFF" })] })] }),
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "Terminal", bold: true, color: "FFFFFF" })] })] }),
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "Duration", bold: true, color: "FFFFFF" })] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, shading: { fill: "E8F4FD", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "1", bold: true })] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Infrastructure & DevOps")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("@ops")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("T1")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("Week 1-2")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, shading: { fill: "E8F4FD", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "2", bold: true })] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Database & Schema")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("@db")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("T2")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("Week 1-3")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, shading: { fill: "E8F4FD", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "3", bold: true })] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Go Core API")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("@go")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("T3")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("Week 2-4")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, shading: { fill: "E8F4FD", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "4", bold: true })] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Go Business Modules")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("@acc")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("T4")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("Week 3-8")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, shading: { fill: "E8F4FD", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "5", bold: true })] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Python Tax Scraper")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("@tax")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("T5")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("Week 4-8")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, shading: { fill: "E8F4FD", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "6", bold: true })] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Python Insurance EDI")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("@hr")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("T6")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("Week 5-10")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, shading: { fill: "E8F4FD", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "7", bold: true })] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Frontend")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("@react")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("T7")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("Week 4-12")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, shading: { fill: "E8F4FD", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "8", bold: true })] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Integration & Testing")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("testing")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("T8")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("Week 6-12")] })] })
          ]})
        ]
      }),

      new Paragraph({ children: [new PageBreak()] }),

      // 3. Remote Server
      new Paragraph({ heading: HeadingLevel.HEADING_1, children: [new TextRun("3. Remote Development Server")] }),

      new Paragraph({ heading: HeadingLevel.HEADING_2, children: [new TextRun("3.1 Server Information")] }),
      new Table({
        columnWidths: [3000, 6000],
        rows: [
          new TableRow({ tableHeader: true, children: [
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "Item", bold: true, color: "FFFFFF" })] })] }),
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "Value", bold: true, color: "FFFFFF" })] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Host")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("wsl-48-246 (61.245.248.246:5022)")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("User")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("blackpc")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Path")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("/home/blackpc/saas-kerp")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Domain")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("erp.abada.kr")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("OS")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("WSL2 Ubuntu")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("RAM")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("94GB (91GB available)")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Disk")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("950GB available")] })] })
          ]})
        ]
      }),

      new Paragraph({ heading: HeadingLevel.HEADING_2, spacing: { before: 400 }, children: [new TextRun("3.2 Service Ports")] }),
      new Table({
        columnWidths: [4000, 2000, 3000],
        rows: [
          new TableRow({ tableHeader: true, children: [
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "Service", bold: true, color: "FFFFFF" })] })] }),
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "Port", bold: true, color: "FFFFFF" })] })] }),
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "Language", bold: true, color: "FFFFFF" })] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Go API Server")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("8080")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("Go")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Tax Scraper (gRPC)")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("50051")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("Python")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Insurance EDI (gRPC)")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("50052")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("Python")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("PostgreSQL")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("5432")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("-")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Redis")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("6379")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("-")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("NATS")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("4222")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun("-")] })] })
          ]})
        ]
      }),

      new Paragraph({ children: [new PageBreak()] }),

      // 4. Security
      new Paragraph({ heading: HeadingLevel.HEADING_1, children: [new TextRun("4. Security Checklist")] }),

      new Paragraph({ heading: HeadingLevel.HEADING_2, children: [new TextRun("4.1 Authentication & Authorization")] }),
      new Paragraph({ numbering: { reference: "bullet-list", level: 0 }, children: [new TextRun("JWT Token - Access: 15min, Refresh: 7days")] }),
      new Paragraph({ numbering: { reference: "bullet-list", level: 0 }, children: [new TextRun("MFA - TOTP support")] }),
      new Paragraph({ numbering: { reference: "bullet-list", level: 0 }, children: [new TextRun("Password - bcrypt hashing")] }),
      new Paragraph({ numbering: { reference: "bullet-list", level: 0 }, children: [new TextRun("RBAC - Role-based permissions")] }),

      new Paragraph({ heading: HeadingLevel.HEADING_2, spacing: { before: 300 }, children: [new TextRun("4.2 Data Security")] }),
      new Paragraph({ numbering: { reference: "bullet-list", level: 0 }, children: [new TextRun("TLS 1.3 for all connections")] }),
      new Paragraph({ numbering: { reference: "bullet-list", level: 0 }, children: [new TextRun("mTLS for Go-Python gRPC communication")] }),
      new Paragraph({ numbering: { reference: "bullet-list", level: 0 }, children: [new TextRun("AES-256 encryption for PII")] }),
      new Paragraph({ numbering: { reference: "bullet-list", level: 0 }, children: [new TextRun("Row-Level Security (RLS) for multi-tenancy")] }),

      new Paragraph({ heading: HeadingLevel.HEADING_2, spacing: { before: 300 }, children: [new TextRun("4.3 Government Integration Security")] }),
      new Paragraph({ numbering: { reference: "bullet-list", level: 0 }, children: [new TextRun("SEED-CBC - Hometax, NPS, EI, WCI")] }),
      new Paragraph({ numbering: { reference: "bullet-list", level: 0 }, children: [new TextRun("ARIA-CBC - NHIS (Health Insurance)")] }),
      new Paragraph({ numbering: { reference: "bullet-list", level: 0 }, children: [new TextRun("PKCS#7 - Digital signature")] }),
      new Paragraph({ numbering: { reference: "bullet-list", level: 0 }, children: [new TextRun("Certificate - Secure storage")] }),

      new Paragraph({ children: [new PageBreak()] }),

      // 5. Commands
      new Paragraph({ heading: HeadingLevel.HEADING_1, children: [new TextRun("5. Commands Reference")] }),

      new Paragraph({ heading: HeadingLevel.HEADING_2, children: [new TextRun("5.1 Skills (Slash Commands)")] }),
      new Table({
        columnWidths: [2500, 6500],
        rows: [
          new TableRow({ tableHeader: true, children: [
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "Command", bold: true, color: "FFFFFF" })] })] }),
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "Description", bold: true, color: "FFFFFF" })] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("/api {name}")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Go REST API scaffolding")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("/grpc {name}")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("gRPC Proto/code generation")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("/provider {name}")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Provider pattern implementation")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("/sqlc {name}")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("sqlc query generation")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("/py {name}")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Python gRPC service")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("/test {name}")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Test code generation")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("/db {name}")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("DB migration")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("/deploy {env}")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Deploy to environment")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("/status")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Show development status")] })] })
          ]})
        ]
      }),

      new Paragraph({ heading: HeadingLevel.HEADING_2, spacing: { before: 400 }, children: [new TextRun("5.2 Agents")] }),
      new Table({
        columnWidths: [2000, 7000],
        rows: [
          new TableRow({ tableHeader: true, children: [
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "Agent", bold: true, color: "FFFFFF" })] })] }),
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "Specialty", bold: true, color: "FFFFFF" })] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("@go")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Go development specialist")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("@py")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Python gRPC specialist")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("@react")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("React frontend specialist")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("@acc")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Accounting domain expert (K-IFRS)")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("@tax")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Tax invoice specialist (Hometax, Popbill)")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("@hr")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("HR/Payroll specialist (4대보험 EDI)")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("@db")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Database architect (PostgreSQL, RLS)")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("@ops")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("DevOps specialist (Docker, K8s)")] })] })
          ]})
        ]
      }),

      // End
      new Paragraph({ spacing: { before: 1200 }, alignment: AlignmentType.CENTER, children: [new TextRun({ text: "--- End of Document ---", size: 20, color: "808080", italics: true })] })
    ]
  }]
});

Packer.toBuffer(doc).then(buffer => {
  fs.writeFileSync("docs/DEVELOPMENT_PLAN.docx", buffer);
  console.log("DEVELOPMENT_PLAN.docx created successfully!");
});
