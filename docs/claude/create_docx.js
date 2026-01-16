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
        children: [new TextRun({ text: "K-ERP SaaS Platform 개발 계획서", size: 18, color: "808080" })]
      })] })
    },
    footers: {
      default: new Footer({ children: [new Paragraph({
        alignment: AlignmentType.CENTER,
        children: [new TextRun({ text: "Page ", size: 18 }), new TextRun({ children: [PageNumber.CURRENT], size: 18 }), new TextRun({ text: " / ", size: 18 }), new TextRun({ children: [PageNumber.TOTAL_PAGES], size: 18 })]
      })] })
    },
    children: [
      // 표지
      new Paragraph({ spacing: { before: 2000 } }),
      new Paragraph({ heading: HeadingLevel.TITLE, children: [new TextRun("한국형 SaaS ERP 시스템")] }),
      new Paragraph({ alignment: AlignmentType.CENTER, spacing: { after: 600 }, children: [new TextRun({ text: "개발 계획서", size: 40, bold: true, color: "404040" })] }),
      new Paragraph({ alignment: AlignmentType.CENTER, spacing: { before: 1200 }, children: [new TextRun({ text: "K-ERP SaaS Platform", size: 28, italics: true, color: "808080" })] }),
      new Paragraph({ alignment: AlignmentType.CENTER, spacing: { before: 800 }, children: [new TextRun({ text: "문서 버전: 1.0", size: 22 })] }),
      new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "작성일: 2025년 1월", size: 22 })] }),

      // 페이지 브레이크
      new Paragraph({ children: [new PageBreak()] }),

      // 1. 프로젝트 개요
      new Paragraph({ heading: HeadingLevel.HEADING_1, children: [new TextRun("1. 프로젝트 개요")] }),

      new Paragraph({ heading: HeadingLevel.HEADING_2, children: [new TextRun("1.1 프로젝트명")] }),
      new Paragraph({ children: [new TextRun({ text: "K-ERP SaaS Platform", bold: true }), new TextRun(" - 한국 중소기업을 위한 클라우드 기반 통합 경영관리 시스템")] }),

      new Paragraph({ heading: HeadingLevel.HEADING_2, children: [new TextRun("1.2 프로젝트 목표")] }),
      new Paragraph({ numbering: { reference: "bullet-list", level: 0 }, children: [new TextRun("국세청 전자세금계산서 연동을 포함한 완전한 한국형 회계 시스템 구축")] }),
      new Paragraph({ numbering: { reference: "bullet-list", level: 0 }, children: [new TextRun("4대보험(국민연금, 건강보험, 고용보험, 산재보험) EDI 연동")] }),
      new Paragraph({ numbering: { reference: "bullet-list", level: 0 }, children: [new TextRun("급여/인사관리 시스템 통합")] }),
      new Paragraph({ numbering: { reference: "bullet-list", level: 0 }, children: [new TextRun("멀티테넌트 SaaS 아키텍처로 확장 가능한 플랫폼 구축")] }),

      new Paragraph({ heading: HeadingLevel.HEADING_2, children: [new TextRun("1.3 대상 고객")] }),
      new Table({
        columnWidths: [3000, 6000],
        rows: [
          new TableRow({ tableHeader: true, children: [
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "구분", bold: true, color: "FFFFFF" })] })] }),
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "설명", bold: true, color: "FFFFFF" })] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun({ text: "Primary", bold: true })] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("연매출 10억~100억 중소기업")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun({ text: "Secondary", bold: true })] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("스타트업 및 소규모 사업자")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun({ text: "Enterprise", bold: true })] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("중견기업 (향후 확장)")] })] })
          ]})
        ]
      }),

      // 페이지 브레이크
      new Paragraph({ children: [new PageBreak()] }),

      // 2. 시스템 범위
      new Paragraph({ heading: HeadingLevel.HEADING_1, children: [new TextRun("2. 시스템 범위")] }),

      new Paragraph({ heading: HeadingLevel.HEADING_2, children: [new TextRun("2.1 핵심 모듈 (MVP)")] }),
      new Table({
        columnWidths: [3000, 6000],
        rows: [
          new TableRow({ tableHeader: true, children: [
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "모듈", bold: true, color: "FFFFFF" })] })] }),
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "주요 기능", bold: true, color: "FFFFFF" })] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun({ text: "회계관리", bold: true })] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("전표입력, 분개장, 재무제표")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun({ text: "세금관리", bold: true })] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("세금계산서, 부가세신고, 원천징수")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun({ text: "인사급여", bold: true })] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("급여계산, 4대보험, 연말정산")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun({ text: "매입매출", bold: true })] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("거래처관리, 견적/주문, 청구/수금")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun({ text: "재고관리", bold: true })] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("입출고, 재고실사, 품목관리")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun({ text: "경영분석", bold: true })] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("대시보드, 리포트, KPI 분석")] })] })
          ]})
        ]
      }),

      new Paragraph({ heading: HeadingLevel.HEADING_2, spacing: { before: 400 }, children: [new TextRun("2.2 정부 기관 연동")] }),
      new Table({
        columnWidths: [2000, 4000, 3000],
        rows: [
          new TableRow({ tableHeader: true, children: [
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "기관", bold: true, color: "FFFFFF" })] })] }),
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "연동 내용", bold: true, color: "FFFFFF" })] })] }),
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "연동 방식", bold: true, color: "FFFFFF" })] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("국세청")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("전자세금계산서 발행/조회")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("ASP API (Popbill)")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("국민연금공단")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("자격취득/상실 신고")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("EDI 파일 전송")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("건강보험공단")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("보험료 고지/납부")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("EDI 파일 전송")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("고용산재보험")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("고용/산재보험 신고")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("4대보험 통합 EDI")] })] })
          ]})
        ]
      }),

      // 페이지 브레이크
      new Paragraph({ children: [new PageBreak()] }),

      // 3. 기술 스택
      new Paragraph({ heading: HeadingLevel.HEADING_1, children: [new TextRun("3. 기술 스택")] }),

      new Paragraph({ heading: HeadingLevel.HEADING_2, children: [new TextRun("3.1 권장 기술 스택")] }),
      new Table({
        columnWidths: [2500, 3000, 3500],
        rows: [
          new TableRow({ tableHeader: true, children: [
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "영역", bold: true, color: "FFFFFF" })] })] }),
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "기술", bold: true, color: "FFFFFF" })] })] }),
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "선정 이유", bold: true, color: "FFFFFF" })] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Core Backend")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Laravel (PHP)")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("기존 코드 활용, 빠른 개발")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Tax/HR Service")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Go (Gin)")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("고성능 API, 동시성 처리")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Frontend")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("React + TypeScript")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("컴포넌트 재사용, 타입 안정성")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Database")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("PostgreSQL 15")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("JSON 지원, 전문검색")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Cache")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Redis 7")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("세션, 캐시 관리")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Container")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Docker + Kubernetes")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("확장성, 배포 자동화")] })] })
          ]})
        ]
      }),

      new Paragraph({ heading: HeadingLevel.HEADING_2, spacing: { before: 400 }, children: [new TextRun("3.2 언어별 역할 분담")] }),
      new Paragraph({ numbering: { reference: "bullet-list", level: 0 }, children: [new TextRun({ text: "PHP/Laravel (50%): ", bold: true }), new TextRun("Core 회계, 사용자 관리, 기준정보, API Gateway")] }),
      new Paragraph({ numbering: { reference: "bullet-list", level: 0 }, children: [new TextRun({ text: "Go (35%): ", bold: true }), new TextRun("세금계산서, 4대보험 EDI, 급여 계산, 배치 처리")] }),
      new Paragraph({ numbering: { reference: "bullet-list", level: 0 }, children: [new TextRun({ text: "Python (15%): ", bold: true }), new TextRun("데이터 분석, 리포트, ML 기능")] }),

      // 페이지 브레이크
      new Paragraph({ children: [new PageBreak()] }),

      // 4. 세금계산서 연동
      new Paragraph({ heading: HeadingLevel.HEADING_1, children: [new TextRun("4. 전자세금계산서 연동")] }),

      new Paragraph({ heading: HeadingLevel.HEADING_2, children: [new TextRun("4.1 연동 방식")] }),
      new Paragraph({ children: [new TextRun("국세청은 일반 개발자에게 직접 API를 제공하지 않습니다. 따라서 등록된 ASP(Application Service Provider) 사업자를 통해 연동해야 합니다.")] }),

      new Paragraph({ heading: HeadingLevel.HEADING_3, spacing: { before: 200 }, children: [new TextRun("권장: ASP API 연동 (Popbill)")] }),
      new Paragraph({ numbering: { reference: "bullet-list", level: 0 }, children: [new TextRun("개발 기간: 1-2주")] }),
      new Paragraph({ numbering: { reference: "bullet-list", level: 0 }, children: [new TextRun("비용: 건당 100원, 월 기본료 없음")] }),
      new Paragraph({ numbering: { reference: "bullet-list", level: 0 }, children: [new TextRun("SDK 지원: PHP, Java, Go, Python, Node.js")] }),

      new Paragraph({ heading: HeadingLevel.HEADING_2, spacing: { before: 400 }, children: [new TextRun("4.2 주요 ASP 서비스 비교")] }),
      new Table({
        columnWidths: [2000, 2000, 2000, 3000],
        rows: [
          new TableRow({ tableHeader: true, children: [
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "서비스", bold: true, color: "FFFFFF" })] })] }),
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "발행 단가", bold: true, color: "FFFFFF" })] })] }),
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "월 기본료", bold: true, color: "FFFFFF" })] })] }),
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "특징", bold: true, color: "FFFFFF" })] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun({ text: "Popbill", bold: true })] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("100원/건")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("없음")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("가장 많은 SDK 지원")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun({ text: "Barobill", bold: true })] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("50-100원/건")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("없음")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("금융권 연동 강점")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun({ text: "Smartbill", bold: true })] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("80원/건")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("없음")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("간편한 REST API")] })] })
          ]})
        ]
      }),

      // 페이지 브레이크
      new Paragraph({ children: [new PageBreak()] }),

      // 5. 4대보험 EDI 연동
      new Paragraph({ heading: HeadingLevel.HEADING_1, children: [new TextRun("5. 4대보험 EDI 연동")] }),

      new Paragraph({ heading: HeadingLevel.HEADING_2, children: [new TextRun("5.1 4대보험 개요")] }),
      new Table({
        columnWidths: [2500, 2500, 2000, 2000],
        rows: [
          new TableRow({ tableHeader: true, children: [
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "보험종류", bold: true, color: "FFFFFF" })] })] }),
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "관리기관", bold: true, color: "FFFFFF" })] })] }),
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "총 요율", bold: true, color: "FFFFFF" })] })] }),
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "사업주분", bold: true, color: "FFFFFF" })] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("국민연금")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("국민연금공단")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("9.0%")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("4.5%")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("건강보험")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("건강보험공단")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("7.09%")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("3.545%")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("고용보험")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("근로복지공단")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("1.8%~")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("0.9%~")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("산재보험")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("근로복지공단")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("업종별")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("전액")] })] })
          ]})
        ]
      }),
      new Paragraph({ spacing: { before: 100 }, children: [new TextRun({ text: "※ 요율은 2024년 기준이며 매년 변경될 수 있음", size: 18, italics: true, color: "808080" })] }),

      new Paragraph({ heading: HeadingLevel.HEADING_2, spacing: { before: 400 }, children: [new TextRun("5.2 연동 방식")] }),
      new Paragraph({ children: [new TextRun("4대보험 정보연계센터를 통한 EDI 파일 전송 방식을 사용합니다.")] }),
      new Paragraph({ numbering: { reference: "bullet-list", level: 0 }, children: [new TextRun("파일 포맷: 고정길이 또는 파이프(|) 구분자")] }),
      new Paragraph({ numbering: { reference: "bullet-list", level: 0 }, children: [new TextRun("문자셋: EUC-KR (필수)")] }),
      new Paragraph({ numbering: { reference: "bullet-list", level: 0 }, children: [new TextRun("전송 방식: SFTP 또는 웹 업로드")] }),
      new Paragraph({ numbering: { reference: "bullet-list", level: 0 }, children: [new TextRun("인증: 공동인증서 필요")] }),

      // 페이지 브레이크
      new Paragraph({ children: [new PageBreak()] }),

      // 6. 개발 로드맵
      new Paragraph({ heading: HeadingLevel.HEADING_1, children: [new TextRun("6. 개발 로드맵")] }),

      new Table({
        columnWidths: [2000, 4500, 2500],
        rows: [
          new TableRow({ tableHeader: true, children: [
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "단계", bold: true, color: "FFFFFF" })] })] }),
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "주요 작업", bold: true, color: "FFFFFF" })] })] }),
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "산출물", bold: true, color: "FFFFFF" })] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, shading: { fill: "E8F4FD", type: ShadingType.CLEAR }, children: [new Paragraph({ children: [new TextRun({ text: "Phase 1", bold: true })] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("개발환경 구성, 인프라 설정, 인증/인가 시스템")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("기반 시스템")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, shading: { fill: "E8F4FD", type: ShadingType.CLEAR }, children: [new Paragraph({ children: [new TextRun({ text: "Phase 2", bold: true })] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("계정과목, 전표관리, 분개장/원장, 재무제표")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("핵심 회계")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, shading: { fill: "E8F4FD", type: ShadingType.CLEAR }, children: [new Paragraph({ children: [new TextRun({ text: "Phase 3", bold: true })] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("Popbill 연동, 정발행/역발행, 부가세 신고자료")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("세금계산서")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, shading: { fill: "E8F4FD", type: ShadingType.CLEAR }, children: [new Paragraph({ children: [new TextRun({ text: "Phase 4", bold: true })] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("직원관리, 급여계산, 4대보험 EDI 연동")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("급여/인사")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, shading: { fill: "E8F4FD", type: ShadingType.CLEAR }, children: [new Paragraph({ children: [new TextRun({ text: "Phase 5", bold: true })] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("거래처/품목 관리, 재고관리, 경영분석 대시보드")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("확장 기능")] })] })
          ]})
        ]
      }),

      // 페이지 브레이크
      new Paragraph({ children: [new PageBreak()] }),

      // 7. 예상 비용
      new Paragraph({ heading: HeadingLevel.HEADING_1, children: [new TextRun("7. 예상 인프라 비용")] }),

      new Table({
        columnWidths: [3000, 3500, 2500],
        rows: [
          new TableRow({ tableHeader: true, children: [
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "항목", bold: true, color: "FFFFFF" })] })] }),
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "스펙", bold: true, color: "FFFFFF" })] })] }),
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "월 비용", bold: true, color: "FFFFFF" })] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("AWS EKS")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("3 nodes (t3.medium)")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.RIGHT, children: [new TextRun("$150")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("RDS PostgreSQL")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("db.t3.medium")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.RIGHT, children: [new TextRun("$80")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("ElastiCache Redis")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("cache.t3.micro")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.RIGHT, children: [new TextRun("$15")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("S3 Storage")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("100GB")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.RIGHT, children: [new TextRun("$5")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("CloudFront CDN")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("100GB transfer")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.RIGHT, children: [new TextRun("$10")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("기타 (모니터링 등)")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("-")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ alignment: AlignmentType.RIGHT, children: [new TextRun("$40")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, shading: { fill: "F0F0F0", type: ShadingType.CLEAR }, children: [new Paragraph({ children: [new TextRun({ text: "합계", bold: true })] })] }),
            new TableCell({ borders: cellBorders, shading: { fill: "F0F0F0", type: ShadingType.CLEAR }, children: [new Paragraph({ children: [new TextRun("-")] })] }),
            new TableCell({ borders: cellBorders, shading: { fill: "F0F0F0", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.RIGHT, children: [new TextRun({ text: "약 $300/월", bold: true })] })] })
          ]})
        ]
      }),
      new Paragraph({ spacing: { before: 100 }, children: [new TextRun({ text: "※ 초기 개발 단계 기준이며, 사용량 증가에 따라 조정 필요", size: 18, italics: true, color: "808080" })] }),

      // 8. 성공 지표
      new Paragraph({ heading: HeadingLevel.HEADING_1, spacing: { before: 600 }, children: [new TextRun("8. 성공 지표 (KPI)")] }),
      new Table({
        columnWidths: [5000, 4000],
        rows: [
          new TableRow({ tableHeader: true, children: [
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "지표", bold: true, color: "FFFFFF" })] })] }),
            new TableCell({ borders: cellBorders, shading: { fill: "2E74B5", type: ShadingType.CLEAR }, children: [new Paragraph({ alignment: AlignmentType.CENTER, children: [new TextRun({ text: "목표", bold: true, color: "FFFFFF" })] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("시스템 가동률")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("99.9%")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("API 응답 시간 (p95)")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("< 200ms")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("세금계산서 발행 성공률")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("99.99%")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("고객 만족도 (NPS)")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("> 50")] })] })
          ]}),
          new TableRow({ children: [
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("월간 활성 사용자 (MAU) - 1년차")] })] }),
            new TableCell({ borders: cellBorders, children: [new Paragraph({ children: [new TextRun("1,000+")] })] })
          ]})
        ]
      }),

      // 마무리
      new Paragraph({ spacing: { before: 1200 }, alignment: AlignmentType.CENTER, children: [new TextRun({ text: "— End of Document —", size: 20, color: "808080", italics: true })] })
    ]
  }]
});

Packer.toBuffer(doc).then(buffer => {
  fs.writeFileSync("/Users/saint/01_DEV/SaaS_erp/docs/plan/K-ERP_계획서_통합.docx", buffer);
  console.log("Document created successfully!");
});
