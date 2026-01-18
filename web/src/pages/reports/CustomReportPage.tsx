import { useState, useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  FileSpreadsheet,
  FileText,
  Play,
  Settings2,
  Table2,
  RefreshCw,
  ChevronDown,
  ChevronUp,
  Filter,
  X,
  Search,
  ArrowUpDown,
  ArrowUp,
  ArrowDown,
} from "lucide-react";
import {
  Button,
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  Input,
  Select,
  Badge,
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from "@/components/ui";
import { DateRangePicker } from "@/components/common";
import { formatCurrency, cn } from "@/lib/utils";
import { reportsApi } from "@/api/reports";
import type { ReportTemplate, CustomReportData } from "@/api/reports";

// Mock templates
const mockTemplates: ReportTemplate[] = [
  {
    id: "sales-by-partner",
    name: "거래처별 매출 현황",
    description: "거래처별 매출액, 거래건수, 평균 거래액을 조회합니다.",
    category: "매출",
    columns: [
      { field: "partnerName", header: "거래처명", type: "string", sortable: true },
      { field: "businessNumber", header: "사업자번호", type: "string" },
      { field: "totalAmount", header: "매출액", type: "currency", sortable: true, aggregatable: true },
      { field: "transactionCount", header: "거래건수", type: "number", sortable: true, aggregatable: true },
      { field: "averageAmount", header: "평균거래액", type: "currency", sortable: true },
    ],
    filters: [
      { field: "period", label: "기간", type: "daterange", required: true },
      { field: "partnerType", label: "거래처유형", type: "select", options: [
        { value: "all", label: "전체" },
        { value: "customer", label: "고객" },
        { value: "supplier", label: "공급사" },
      ]},
    ],
    defaultSort: "totalAmount",
    defaultGroupBy: "partnerName",
  },
  {
    id: "expense-by-account",
    name: "계정과목별 비용 현황",
    description: "계정과목별 비용 발생액, 예산, 집행률을 조회합니다.",
    category: "비용",
    columns: [
      { field: "accountCode", header: "계정코드", type: "string" },
      { field: "accountName", header: "계정과목", type: "string", sortable: true },
      { field: "currentAmount", header: "당기비용", type: "currency", sortable: true, aggregatable: true },
      { field: "budgetAmount", header: "예산", type: "currency", aggregatable: true },
      { field: "executionRate", header: "집행률", type: "percentage", sortable: true },
      { field: "previousAmount", header: "전기비용", type: "currency", aggregatable: true },
    ],
    filters: [
      { field: "period", label: "기간", type: "daterange", required: true },
      { field: "accountCategory", label: "계정분류", type: "select", options: [
        { value: "all", label: "전체" },
        { value: "labor", label: "인건비" },
        { value: "operation", label: "운영비" },
        { value: "marketing", label: "마케팅비" },
      ]},
    ],
    defaultSort: "currentAmount",
  },
  {
    id: "employee-list",
    name: "직원 목록",
    description: "직원의 인적사항, 부서, 직급, 입사일 등을 조회합니다.",
    category: "인사",
    columns: [
      { field: "employeeId", header: "사번", type: "string" },
      { field: "name", header: "성명", type: "string", sortable: true },
      { field: "department", header: "부서", type: "string", sortable: true },
      { field: "position", header: "직급", type: "string", sortable: true },
      { field: "hireDate", header: "입사일", type: "date", sortable: true },
      { field: "employmentType", header: "고용형태", type: "string" },
    ],
    filters: [
      { field: "department", label: "부서", type: "select", options: [
        { value: "all", label: "전체" },
        { value: "dev", label: "개발부" },
        { value: "sales", label: "영업부" },
        { value: "marketing", label: "마케팅부" },
        { value: "support", label: "경영지원부" },
      ]},
      { field: "employmentType", label: "고용형태", type: "select", options: [
        { value: "all", label: "전체" },
        { value: "regular", label: "정규직" },
        { value: "contract", label: "계약직" },
        { value: "partTime", label: "파트타임" },
      ]},
    ],
    defaultSort: "name",
  },
  {
    id: "voucher-list",
    name: "전표 목록",
    description: "전표의 일자, 적요, 금액, 승인상태를 조회합니다.",
    category: "회계",
    columns: [
      { field: "voucherNumber", header: "전표번호", type: "string" },
      { field: "voucherDate", header: "전표일자", type: "date", sortable: true },
      { field: "description", header: "적요", type: "string" },
      { field: "debitAmount", header: "차변", type: "currency", aggregatable: true },
      { field: "creditAmount", header: "대변", type: "currency", aggregatable: true },
      { field: "status", header: "상태", type: "string" },
    ],
    filters: [
      { field: "period", label: "기간", type: "daterange", required: true },
      { field: "status", label: "상태", type: "select", options: [
        { value: "all", label: "전체" },
        { value: "draft", label: "임시저장" },
        { value: "pending", label: "승인대기" },
        { value: "approved", label: "승인완료" },
      ]},
    ],
    defaultSort: "voucherDate",
  },
  {
    id: "product-inventory",
    name: "품목별 재고 현황",
    description: "품목별 현재고, 입출고 현황을 조회합니다.",
    category: "재고",
    columns: [
      { field: "productCode", header: "품목코드", type: "string" },
      { field: "productName", header: "품목명", type: "string", sortable: true },
      { field: "category", header: "카테고리", type: "string" },
      { field: "currentStock", header: "현재고", type: "number", sortable: true, aggregatable: true },
      { field: "inbound", header: "입고", type: "number", aggregatable: true },
      { field: "outbound", header: "출고", type: "number", aggregatable: true },
      { field: "stockValue", header: "재고금액", type: "currency", sortable: true, aggregatable: true },
    ],
    filters: [
      { field: "period", label: "기간", type: "daterange", required: true },
      { field: "category", label: "카테고리", type: "select", options: [
        { value: "all", label: "전체" },
        { value: "electronics", label: "전자" },
        { value: "machinery", label: "기계" },
        { value: "consumables", label: "소모품" },
      ]},
    ],
    defaultSort: "stockValue",
  },
];

// Mock report data generator
const generateMockReportData = (template: ReportTemplate): CustomReportData => {
  const mockRows: Record<string, unknown>[] = [];

  // Generate sample data based on template
  for (let i = 0; i < 15; i++) {
    const row: Record<string, unknown> = {};
    template.columns.forEach((col) => {
      switch (col.type) {
        case "string":
          if (col.field.includes("name") || col.field.includes("Name")) {
            row[col.field] = `샘플 ${col.header} ${i + 1}`;
          } else if (col.field.includes("Number") || col.field.includes("Code") || col.field.includes("Id")) {
            row[col.field] = `${col.field.toUpperCase().slice(0, 3)}-${String(i + 1).padStart(4, "0")}`;
          } else {
            row[col.field] = `${col.header} ${i + 1}`;
          }
          break;
        case "number":
          row[col.field] = Math.floor(Math.random() * 1000) + 100;
          break;
        case "currency":
          row[col.field] = Math.floor(Math.random() * 50000000) + 1000000;
          break;
        case "date":
          const date = new Date();
          date.setDate(date.getDate() - Math.floor(Math.random() * 365));
          row[col.field] = date.toISOString().split("T")[0];
          break;
        case "percentage":
          row[col.field] = Math.floor(Math.random() * 100) + 1;
          break;
        default:
          row[col.field] = `값 ${i + 1}`;
      }
    });
    mockRows.push(row);
  }

  // Calculate totals for aggregatable columns
  const totals: Record<string, number> = {};
  template.columns
    .filter((col) => col.aggregatable)
    .forEach((col) => {
      totals[col.field] = mockRows.reduce((sum, row) => sum + (Number(row[col.field]) || 0), 0);
    });

  return {
    templateId: template.id,
    templateName: template.name,
    generatedAt: new Date().toISOString(),
    columns: template.columns,
    rows: mockRows,
    totals,
    pagination: {
      page: 1,
      pageSize: 20,
      totalCount: mockRows.length,
      totalPages: 1,
    },
  };
};

// Format cell value based on column type
function formatCellValue(value: unknown, type: string): string {
  if (value === null || value === undefined) return "-";

  switch (type) {
    case "currency":
      return formatCurrency(Number(value), { showSymbol: false });
    case "number":
      return Number(value).toLocaleString();
    case "percentage":
      return `${Number(value).toFixed(1)}%`;
    case "date":
      return new Date(String(value)).toLocaleDateString("ko-KR");
    default:
      return String(value);
  }
}

export function CustomReportPage() {
  const today = new Date();
  const firstDayOfMonth = new Date(today.getFullYear(), today.getMonth(), 1);

  const [selectedTemplateId, setSelectedTemplateId] = useState<string>("");
  const [startDate, setStartDate] = useState(firstDayOfMonth.toISOString().split("T")[0]);
  const [endDate, setEndDate] = useState(today.toISOString().split("T")[0]);
  const [filterValues, setFilterValues] = useState<Record<string, string>>({});
  const [reportData, setReportData] = useState<CustomReportData | null>(null);
  const [sortField, setSortField] = useState<string>("");
  const [sortDirection, setSortDirection] = useState<"asc" | "desc">("desc");
  const [showFilters, setShowFilters] = useState(true);
  const [searchQuery, setSearchQuery] = useState("");

  // Fetch templates
  const {
    data: templatesResponse,
    isLoading: isLoadingTemplates,
  } = useQuery({
    queryKey: ["reports", "templates"],
    queryFn: () => reportsApi.getTemplates(),
  });

  // Use mock templates if no real data
  const templates = templatesResponse?.data || mockTemplates;

  // Get selected template
  const selectedTemplate = templates.find((t) => t.id === selectedTemplateId);

  // Set default sort when template changes
  useEffect(() => {
    if (selectedTemplate?.defaultSort) {
      setSortField(selectedTemplate.defaultSort);
      setSortDirection("desc");
    }
  }, [selectedTemplate]);

  const handleDateChange = (start: string, end: string) => {
    setStartDate(start);
    setEndDate(end);
  };

  const handleFilterChange = (field: string, value: string) => {
    setFilterValues((prev) => ({ ...prev, [field]: value }));
  };

  const handleGenerateReport = () => {
    if (!selectedTemplate) return;

    // In real implementation, this would call the API
    const mockData = generateMockReportData(selectedTemplate);
    setReportData(mockData);
  };

  const handleSort = (field: string) => {
    if (sortField === field) {
      setSortDirection(sortDirection === "asc" ? "desc" : "asc");
    } else {
      setSortField(field);
      setSortDirection("desc");
    }
  };

  const handleExport = (format: "xlsx" | "pdf") => {
    // In real implementation, this would call the export API
    alert(`${format.toUpperCase()} 파일로 내보내기를 시작합니다.`);
  };

  // Sort and filter rows
  const processedRows = reportData?.rows
    .filter((row) => {
      if (!searchQuery) return true;
      return Object.values(row).some((value) =>
        String(value).toLowerCase().includes(searchQuery.toLowerCase())
      );
    })
    .sort((a, b) => {
      if (!sortField) return 0;
      const aVal = a[sortField];
      const bVal = b[sortField];
      const direction = sortDirection === "asc" ? 1 : -1;

      if (typeof aVal === "number" && typeof bVal === "number") {
        return (aVal - bVal) * direction;
      }
      return String(aVal).localeCompare(String(bVal)) * direction;
    }) || [];

  // Group templates by category
  const templatesByCategory = templates.reduce((acc, template) => {
    if (!acc[template.category]) {
      acc[template.category] = [];
    }
    acc[template.category].push(template);
    return acc;
  }, {} as Record<string, ReportTemplate[]>);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">맞춤 보고서</h1>
          <p className="text-muted-foreground">
            템플릿을 선택하고 조건을 설정하여 맞춤 보고서를 생성합니다.
          </p>
        </div>
        {reportData && (
          <div className="flex items-center space-x-2">
            <Button variant="outline" onClick={() => handleExport("xlsx")}>
              <FileSpreadsheet className="h-4 w-4 mr-2" />
              Excel
            </Button>
            <Button variant="outline" onClick={() => handleExport("pdf")}>
              <FileText className="h-4 w-4 mr-2" />
              PDF
            </Button>
          </div>
        )}
      </div>

      {/* Template Selection */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg flex items-center gap-2">
            <Table2 className="h-5 w-5" />
            보고서 템플릿
          </CardTitle>
        </CardHeader>
        <CardContent>
          {isLoadingTemplates ? (
            <div className="py-8 flex items-center justify-center">
              <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-primary"></div>
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {Object.entries(templatesByCategory).map(([category, categoryTemplates]) => (
                <div key={category} className="space-y-2">
                  <h4 className="text-sm font-medium text-muted-foreground">{category}</h4>
                  <div className="space-y-2">
                    {categoryTemplates.map((template) => (
                      <button
                        key={template.id}
                        onClick={() => {
                          setSelectedTemplateId(template.id);
                          setReportData(null);
                          setFilterValues({});
                        }}
                        className={cn(
                          "w-full text-left p-3 rounded-lg border transition-colors",
                          selectedTemplateId === template.id
                            ? "border-primary bg-primary/5"
                            : "border-border hover:border-primary/50 hover:bg-muted/50"
                        )}
                      >
                        <p className="font-medium">{template.name}</p>
                        <p className="text-xs text-muted-foreground mt-1">
                          {template.description}
                        </p>
                      </button>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Filters */}
      {selectedTemplate && (
        <Card>
          <CardHeader className="cursor-pointer" onClick={() => setShowFilters(!showFilters)}>
            <CardTitle className="text-lg flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Settings2 className="h-5 w-5" />
                조회 조건
              </div>
              {showFilters ? (
                <ChevronUp className="h-5 w-5" />
              ) : (
                <ChevronDown className="h-5 w-5" />
              )}
            </CardTitle>
          </CardHeader>
          {showFilters && (
            <CardContent>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                {/* Date range picker for period filter */}
                {selectedTemplate.filters.some((f) => f.type === "daterange") && (
                  <div className="col-span-1 md:col-span-2">
                    <DateRangePicker
                      startDate={startDate}
                      endDate={endDate}
                      onDateChange={handleDateChange}
                      label="조회기간"
                    />
                  </div>
                )}

                {/* Other filters */}
                {selectedTemplate.filters
                  .filter((f) => f.type !== "daterange")
                  .map((filter) => (
                    <div key={filter.field}>
                      <label className="block text-sm font-medium mb-1.5">
                        {filter.label}
                        {filter.required && <span className="text-destructive ml-1">*</span>}
                      </label>
                      {filter.type === "select" && filter.options && (
                        <Select
                          value={filterValues[filter.field] || "all"}
                          onChange={(value) => handleFilterChange(filter.field, value)}
                          options={filter.options.map(opt => ({ value: opt.value, label: opt.label }))}
                          className="w-full"
                        />
                      )}
                      {filter.type === "text" && (
                        <Input
                          value={filterValues[filter.field] || ""}
                          onChange={(e) => handleFilterChange(filter.field, e.target.value)}
                          placeholder={`${filter.label} 입력`}
                        />
                      )}
                    </div>
                  ))}
              </div>

              <div className="mt-4 flex justify-end">
                <Button onClick={handleGenerateReport}>
                  <Play className="h-4 w-4 mr-2" />
                  보고서 생성
                </Button>
              </div>
            </CardContent>
          )}
        </Card>
      )}

      {/* Report Results */}
      {reportData && (
        <Card>
          <CardHeader>
            <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
              <CardTitle className="text-lg">
                {reportData.templateName}
                <Badge variant="outline" className="ml-2 font-normal">
                  {processedRows.length}건
                </Badge>
              </CardTitle>
              <div className="flex items-center gap-2">
                <div className="relative">
                  <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                  <Input
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    placeholder="검색..."
                    className="pl-9 w-48"
                  />
                  {searchQuery && (
                    <button
                      onClick={() => setSearchQuery("")}
                      className="absolute right-3 top-1/2 transform -translate-y-1/2"
                    >
                      <X className="h-4 w-4 text-muted-foreground" />
                    </button>
                  )}
                </div>
                <Button variant="outline" size="icon" onClick={handleGenerateReport}>
                  <RefreshCw className="h-4 w-4" />
                </Button>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            <div className="border rounded-lg overflow-hidden">
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow className="bg-muted/50">
                      {reportData.columns.map((column) => (
                        <TableHead
                          key={column.field}
                          className={cn(
                            column.sortable && "cursor-pointer hover:bg-muted",
                            column.type === "currency" || column.type === "number" || column.type === "percentage"
                              ? "text-right"
                              : ""
                          )}
                          onClick={() => column.sortable && handleSort(column.field)}
                        >
                          <div className={cn(
                            "flex items-center gap-1",
                            column.type === "currency" || column.type === "number" || column.type === "percentage"
                              ? "justify-end"
                              : ""
                          )}>
                            {column.header}
                            {column.sortable && (
                              <span className="text-muted-foreground">
                                {sortField === column.field ? (
                                  sortDirection === "asc" ? (
                                    <ArrowUp className="h-3 w-3" />
                                  ) : (
                                    <ArrowDown className="h-3 w-3" />
                                  )
                                ) : (
                                  <ArrowUpDown className="h-3 w-3" />
                                )}
                              </span>
                            )}
                          </div>
                        </TableHead>
                      ))}
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {processedRows.map((row, rowIndex) => (
                      <TableRow key={rowIndex} className="hover:bg-muted/30">
                        {reportData.columns.map((column) => (
                          <TableCell
                            key={column.field}
                            className={cn(
                              column.type === "currency" || column.type === "number" || column.type === "percentage"
                                ? "text-right font-mono"
                                : ""
                            )}
                          >
                            {formatCellValue(row[column.field], column.type)}
                          </TableCell>
                        ))}
                      </TableRow>
                    ))}

                    {/* Totals row */}
                    {reportData.totals && Object.keys(reportData.totals).length > 0 && (
                      <TableRow className="bg-primary/10 font-semibold">
                        {reportData.columns.map((column, index) => (
                          <TableCell
                            key={column.field}
                            className={cn(
                              column.type === "currency" || column.type === "number" || column.type === "percentage"
                                ? "text-right font-mono"
                                : ""
                            )}
                          >
                            {index === 0 ? (
                              "합계"
                            ) : column.aggregatable && reportData.totals?.[column.field] !== undefined ? (
                              formatCellValue(reportData.totals[column.field], column.type)
                            ) : (
                              ""
                            )}
                          </TableCell>
                        ))}
                      </TableRow>
                    )}
                  </TableBody>
                </Table>
              </div>
            </div>

            {/* Report Info */}
            <div className="mt-4 flex items-center justify-between text-xs text-muted-foreground">
              <span>
                생성일시: {new Date(reportData.generatedAt).toLocaleString("ko-KR")}
              </span>
              <span>
                {reportData.pagination.totalCount}건 중 {processedRows.length}건 표시
              </span>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Empty state */}
      {!selectedTemplateId && (
        <Card>
          <CardContent className="py-16 text-center">
            <Table2 className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
            <h3 className="text-lg font-medium mb-2">보고서 템플릿을 선택해주세요</h3>
            <p className="text-muted-foreground">
              위의 템플릿 목록에서 원하는 보고서를 선택하면 조건 설정 후 보고서를 생성할 수 있습니다.
            </p>
          </CardContent>
        </Card>
      )}

      {selectedTemplateId && !reportData && (
        <Card>
          <CardContent className="py-16 text-center">
            <Filter className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
            <h3 className="text-lg font-medium mb-2">조회 조건을 설정하고 보고서를 생성하세요</h3>
            <p className="text-muted-foreground">
              조회 기간과 필터를 설정한 후 "보고서 생성" 버튼을 클릭해주세요.
            </p>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
