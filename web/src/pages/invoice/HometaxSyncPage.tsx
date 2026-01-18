import { useState, useMemo } from "react";
import {
  RefreshCw,
  XCircle,
  AlertCircle,
  Clock,
  Download,
  FileText,
  Link2,
  ShieldCheck,
  Activity,
  ArrowUpDown,
  Building2,
  TrendingUp,
  TrendingDown,
  Minus,
} from "lucide-react";
import {
  Button,
  Input,
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  Badge,
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
  Modal,
} from "@/components/ui";
import { formatCurrency, formatDate, cn } from "@/lib/utils";
import { toast } from "@/stores/ui";

// Auth status types
type AuthStatus = "connected" | "disconnected" | "expired" | "pending";

// Sync status types
type SyncStatus = "success" | "failed" | "in_progress" | "pending";

// Comparison status types
type ComparisonStatus = "matched" | "unmatched" | "erp_only" | "hometax_only";

// Mock auth state
const mockAuthState = {
  status: "connected" as AuthStatus,
  lastAuthDate: "2024-01-15T10:30:00",
  authMethod: "certificate", // 'certificate' | 'simple' | 'bio'
  businessNumber: "1234567890",
  companyName: "(주)우리회사",
  expiresAt: "2024-02-15T10:30:00",
};

// Mock sync history
const mockSyncHistory = [
  {
    id: "1",
    syncDate: "2024-01-15T14:30:00",
    syncType: "all" as const,
    status: "success" as SyncStatus,
    issuedCount: 25,
    receivedCount: 18,
    duration: 45,
    message: "동기화 완료",
  },
  {
    id: "2",
    syncDate: "2024-01-14T09:15:00",
    syncType: "issued" as const,
    status: "success" as SyncStatus,
    issuedCount: 12,
    receivedCount: 0,
    duration: 22,
    message: "매출 세금계산서 동기화 완료",
  },
  {
    id: "3",
    syncDate: "2024-01-13T16:45:00",
    syncType: "received" as const,
    status: "failed" as SyncStatus,
    issuedCount: 0,
    receivedCount: 0,
    duration: 0,
    message: "홈택스 서버 연결 실패",
  },
  {
    id: "4",
    syncDate: "2024-01-12T11:00:00",
    syncType: "all" as const,
    status: "success" as SyncStatus,
    issuedCount: 30,
    receivedCount: 22,
    duration: 58,
    message: "동기화 완료",
  },
  {
    id: "5",
    syncDate: "2024-01-11T08:30:00",
    syncType: "all" as const,
    status: "success" as SyncStatus,
    issuedCount: 15,
    receivedCount: 10,
    duration: 32,
    message: "동기화 완료",
  },
];

// Mock comparison data (ERP vs Hometax)
const mockComparisonData = {
  issued: {
    erpTotal: 150,
    hometaxTotal: 148,
    matched: 145,
    unmatched: 3,
    erpOnly: 5,
    hometaxOnly: 0,
    erpAmount: 1500000000,
    hometaxAmount: 1480000000,
  },
  received: {
    erpTotal: 120,
    hometaxTotal: 125,
    matched: 118,
    unmatched: 2,
    erpOnly: 0,
    hometaxOnly: 7,
    erpAmount: 980000000,
    hometaxAmount: 1020000000,
  },
};

// Mock unmatched invoices
const mockUnmatchedInvoices = [
  {
    id: "1",
    invoiceNumber: "20240115-001",
    issueDate: "2024-01-15",
    partnerName: "A상사",
    partnerBusinessNumber: "1234567890",
    direction: "issued" as const,
    erpAmount: 11000000,
    hometaxAmount: 10000000,
    status: "unmatched" as ComparisonStatus,
    difference: 1000000,
    note: "금액 불일치",
  },
  {
    id: "2",
    invoiceNumber: "20240114-002",
    issueDate: "2024-01-14",
    partnerName: "B기업",
    partnerBusinessNumber: "9876543210",
    direction: "issued" as const,
    erpAmount: 5500000,
    hometaxAmount: null,
    status: "erp_only" as ComparisonStatus,
    difference: 5500000,
    note: "홈택스 미수신",
  },
  {
    id: "3",
    invoiceNumber: "20240113-003",
    issueDate: "2024-01-13",
    partnerName: "C물류",
    partnerBusinessNumber: "5555666677",
    direction: "received" as const,
    erpAmount: null,
    hometaxAmount: 3300000,
    status: "hometax_only" as ComparisonStatus,
    difference: -3300000,
    note: "ERP 미등록",
  },
  {
    id: "4",
    invoiceNumber: "20240112-004",
    issueDate: "2024-01-12",
    partnerName: "D테크",
    partnerBusinessNumber: "1112223334",
    direction: "received" as const,
    erpAmount: 8800000,
    hometaxAmount: 8800000,
    status: "matched" as ComparisonStatus,
    difference: 0,
    note: "",
  },
];

// Auth status badge styles
const authStatusStyles: Record<
  AuthStatus,
  { variant: "default" | "secondary" | "destructive" | "success" | "warning"; label: string }
> = {
  connected: { variant: "success", label: "연결됨" },
  disconnected: { variant: "secondary", label: "미연결" },
  expired: { variant: "destructive", label: "만료됨" },
  pending: { variant: "warning", label: "대기중" },
};

// Sync status badge styles
const syncStatusStyles: Record<
  SyncStatus,
  { variant: "default" | "secondary" | "destructive" | "success" | "warning"; label: string }
> = {
  success: { variant: "success", label: "성공" },
  failed: { variant: "destructive", label: "실패" },
  in_progress: { variant: "warning", label: "진행중" },
  pending: { variant: "secondary", label: "대기" },
};

// Comparison status badge styles
const comparisonStatusStyles: Record<
  ComparisonStatus,
  { variant: "default" | "secondary" | "destructive" | "success" | "warning"; label: string }
> = {
  matched: { variant: "success", label: "일치" },
  unmatched: { variant: "destructive", label: "불일치" },
  erp_only: { variant: "warning", label: "ERP만 존재" },
  hometax_only: { variant: "default", label: "홈택스만 존재" },
};

// Auth method labels
const authMethodLabels: Record<string, string> = {
  certificate: "공인인증서",
  simple: "간편인증",
  bio: "생체인증",
};

// Sync type labels
const syncTypeLabels: Record<string, string> = {
  all: "전체",
  issued: "매출",
  received: "매입",
};

export function HometaxSyncPage() {
  // State
  const [authState] = useState(mockAuthState);
  const [syncHistory] = useState(mockSyncHistory);
  const [comparisonData] = useState(mockComparisonData);
  const [unmatchedInvoices] = useState(mockUnmatchedInvoices);

  const [isSyncing, setIsSyncing] = useState(false);
  const [syncType, setSyncType] = useState<"all" | "issued" | "received">("all");
  const [dateFrom, setDateFrom] = useState(
    new Date(new Date().setMonth(new Date().getMonth() - 1)).toISOString().split("T")[0]
  );
  const [dateTo, setDateTo] = useState(new Date().toISOString().split("T")[0]);

  const [showAuthModal, setShowAuthModal] = useState(false);
  const [selectedTab, setSelectedTab] = useState<"sync" | "compare">("sync");
  const [comparisonFilter, setComparisonFilter] = useState<ComparisonStatus | "all">("all");
  const [directionFilter, setDirectionFilter] = useState<"all" | "issued" | "received">("all");

  // Filtered unmatched invoices
  const filteredInvoices = useMemo(() => {
    return unmatchedInvoices.filter((invoice) => {
      const matchesStatus =
        comparisonFilter === "all" || invoice.status === comparisonFilter;
      const matchesDirection =
        directionFilter === "all" || invoice.direction === directionFilter;
      return matchesStatus && matchesDirection;
    });
  }, [unmatchedInvoices, comparisonFilter, directionFilter]);

  // Handle sync action
  const handleSync = async () => {
    if (authState.status !== "connected") {
      toast.warning("인증 필요", "홈택스 연동을 위해 먼저 인증해주세요.");
      setShowAuthModal(true);
      return;
    }

    setIsSyncing(true);
    try {
      // Simulate API call
      await new Promise((resolve) => setTimeout(resolve, 3000));
      toast.success("동기화 완료", `${syncTypeLabels[syncType]} 세금계산서 동기화가 완료되었습니다.`);
    } catch {
      toast.error("동기화 실패", "홈택스 연결 중 오류가 발생했습니다.");
    } finally {
      setIsSyncing(false);
    }
  };

  // Handle auth
  const handleAuth = () => {
    toast.info("인증 진행", "공인인증서 로그인 페이지로 이동합니다.");
    setShowAuthModal(false);
  };

  // Check if auth is about to expire (within 7 days)
  const isAuthExpiringSoon = useMemo(() => {
    if (authState.status !== "connected" || !authState.expiresAt) return false;
    const expiresAt = new Date(authState.expiresAt);
    const now = new Date();
    const daysUntilExpiry = (expiresAt.getTime() - now.getTime()) / (1000 * 60 * 60 * 24);
    return daysUntilExpiry <= 7 && daysUntilExpiry > 0;
  }, [authState]);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">홈택스 연동</h1>
          <p className="text-muted-foreground">
            국세청 홈택스와 세금계산서를 동기화하고 비교합니다.
          </p>
        </div>
        <div className="flex items-center space-x-2">
          <Button variant="outline" onClick={() => setShowAuthModal(true)}>
            <Link2 className="h-4 w-4 mr-2" />
            인증 관리
          </Button>
          <Button onClick={handleSync} isLoading={isSyncing} disabled={isSyncing}>
            <RefreshCw className={cn("h-4 w-4 mr-2", isSyncing && "animate-spin")} />
            동기화
          </Button>
        </div>
      </div>

      {/* Auth Status Card */}
      <Card>
        <CardContent className="p-4">
          <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
            <div className="flex items-center space-x-4">
              <div
                className={cn(
                  "p-3 rounded-full",
                  authState.status === "connected" ? "bg-green-100" : "bg-gray-100"
                )}
              >
                {authState.status === "connected" ? (
                  <ShieldCheck className="h-6 w-6 text-green-600" />
                ) : (
                  <XCircle className="h-6 w-6 text-gray-400" />
                )}
              </div>
              <div>
                <div className="flex items-center space-x-2">
                  <p className="font-semibold">홈택스 연동 상태</p>
                  <Badge variant={authStatusStyles[authState.status].variant}>
                    {authStatusStyles[authState.status].label}
                  </Badge>
                </div>
                {authState.status === "connected" ? (
                  <div className="text-sm text-muted-foreground mt-1">
                    <span>{authState.companyName}</span>
                    <span className="mx-2">|</span>
                    <span>{authMethodLabels[authState.authMethod]}</span>
                    <span className="mx-2">|</span>
                    <span>마지막 인증: {formatDate(authState.lastAuthDate, { format: "time" })}</span>
                  </div>
                ) : (
                  <p className="text-sm text-muted-foreground mt-1">
                    홈택스 연동을 위해 인증이 필요합니다.
                  </p>
                )}
              </div>
            </div>
            {isAuthExpiringSoon && (
              <div className="flex items-center space-x-2 text-warning">
                <AlertCircle className="h-4 w-4" />
                <span className="text-sm">인증 만료 예정</span>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Tab Navigation */}
      <div className="border-b">
        <nav className="flex space-x-8">
          <button
            onClick={() => setSelectedTab("sync")}
            className={cn(
              "py-3 px-1 border-b-2 font-medium text-sm transition-colors",
              selectedTab === "sync"
                ? "border-primary text-primary"
                : "border-transparent text-muted-foreground hover:text-foreground"
            )}
          >
            <RefreshCw className="h-4 w-4 inline-block mr-2" />
            동기화
          </button>
          <button
            onClick={() => setSelectedTab("compare")}
            className={cn(
              "py-3 px-1 border-b-2 font-medium text-sm transition-colors",
              selectedTab === "compare"
                ? "border-primary text-primary"
                : "border-transparent text-muted-foreground hover:text-foreground"
            )}
          >
            <ArrowUpDown className="h-4 w-4 inline-block mr-2" />
            매출/매입 비교
          </button>
        </nav>
      </div>

      {selectedTab === "sync" && (
        <>
          {/* Sync Options */}
          <Card>
            <CardHeader>
              <CardTitle className="text-lg flex items-center">
                <RefreshCw className="h-5 w-5 mr-2" />
                동기화 설정
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                <div>
                  <label className="text-sm font-medium mb-2 block">동기화 유형</label>
                  <select
                    className="w-full h-9 rounded-md border border-input bg-background px-3 text-sm"
                    value={syncType}
                    onChange={(e) =>
                      setSyncType(e.target.value as "all" | "issued" | "received")
                    }
                  >
                    <option value="all">전체 (매출 + 매입)</option>
                    <option value="issued">매출 세금계산서</option>
                    <option value="received">매입 세금계산서</option>
                  </select>
                </div>
                <div>
                  <label className="text-sm font-medium mb-2 block">시작일</label>
                  <Input
                    type="date"
                    value={dateFrom}
                    onChange={(e) => setDateFrom(e.target.value)}
                  />
                </div>
                <div>
                  <label className="text-sm font-medium mb-2 block">종료일</label>
                  <Input
                    type="date"
                    value={dateTo}
                    onChange={(e) => setDateTo(e.target.value)}
                  />
                </div>
                <div className="flex items-end">
                  <Button
                    className="w-full"
                    onClick={handleSync}
                    isLoading={isSyncing}
                    disabled={isSyncing}
                  >
                    <RefreshCw className={cn("h-4 w-4 mr-2", isSyncing && "animate-spin")} />
                    동기화 실행
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Sync Summary */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <Card>
              <CardContent className="p-4">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm text-muted-foreground">마지막 동기화</p>
                    <p className="text-lg font-semibold">
                      {syncHistory[0]
                        ? formatDate(syncHistory[0].syncDate, { format: "time" })
                        : "-"}
                    </p>
                  </div>
                  <Clock className="h-8 w-8 text-muted-foreground" />
                </div>
              </CardContent>
            </Card>
            <Card>
              <CardContent className="p-4">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm text-muted-foreground">매출 계산서</p>
                    <p className="text-lg font-semibold">
                      {syncHistory[0]?.issuedCount || 0}건
                    </p>
                  </div>
                  <TrendingUp className="h-8 w-8 text-green-500" />
                </div>
              </CardContent>
            </Card>
            <Card>
              <CardContent className="p-4">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm text-muted-foreground">매입 계산서</p>
                    <p className="text-lg font-semibold">
                      {syncHistory[0]?.receivedCount || 0}건
                    </p>
                  </div>
                  <TrendingDown className="h-8 w-8 text-red-500" />
                </div>
              </CardContent>
            </Card>
            <Card>
              <CardContent className="p-4">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm text-muted-foreground">동기화 상태</p>
                    {syncHistory[0] && (
                      <Badge variant={syncStatusStyles[syncHistory[0].status].variant}>
                        {syncStatusStyles[syncHistory[0].status].label}
                      </Badge>
                    )}
                  </div>
                  <Activity className="h-8 w-8 text-muted-foreground" />
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Sync History */}
          <Card>
            <CardHeader>
              <CardTitle className="text-lg flex items-center">
                <FileText className="h-5 w-5 mr-2" />
                동기화 이력
              </CardTitle>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>동기화 일시</TableHead>
                    <TableHead>유형</TableHead>
                    <TableHead className="text-right">매출</TableHead>
                    <TableHead className="text-right">매입</TableHead>
                    <TableHead className="text-right">소요시간</TableHead>
                    <TableHead>상태</TableHead>
                    <TableHead>메시지</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {syncHistory.map((sync) => (
                    <TableRow key={sync.id}>
                      <TableCell>{formatDate(sync.syncDate, { format: "time" })}</TableCell>
                      <TableCell>
                        <Badge variant="outline">{syncTypeLabels[sync.syncType]}</Badge>
                      </TableCell>
                      <TableCell className="text-right font-mono">
                        {sync.issuedCount}건
                      </TableCell>
                      <TableCell className="text-right font-mono">
                        {sync.receivedCount}건
                      </TableCell>
                      <TableCell className="text-right font-mono">
                        {sync.duration > 0 ? `${sync.duration}초` : "-"}
                      </TableCell>
                      <TableCell>
                        <Badge variant={syncStatusStyles[sync.status].variant}>
                          {syncStatusStyles[sync.status].label}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {sync.message}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </>
      )}

      {selectedTab === "compare" && (
        <>
          {/* Comparison Summary */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Issued (Sales) Comparison */}
            <Card>
              <CardHeader>
                <CardTitle className="text-lg flex items-center">
                  <TrendingUp className="h-5 w-5 mr-2 text-green-500" />
                  매출 세금계산서 비교
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  <div className="grid grid-cols-2 gap-4">
                    <div className="p-3 bg-muted rounded-lg">
                      <p className="text-sm text-muted-foreground">ERP 등록</p>
                      <p className="text-xl font-bold">{comparisonData.issued.erpTotal}건</p>
                      <p className="text-sm text-muted-foreground">
                        {formatCurrency(comparisonData.issued.erpAmount, { compact: true })}
                      </p>
                    </div>
                    <div className="p-3 bg-muted rounded-lg">
                      <p className="text-sm text-muted-foreground">홈택스</p>
                      <p className="text-xl font-bold">{comparisonData.issued.hometaxTotal}건</p>
                      <p className="text-sm text-muted-foreground">
                        {formatCurrency(comparisonData.issued.hometaxAmount, { compact: true })}
                      </p>
                    </div>
                  </div>
                  <div className="space-y-2">
                    <div className="flex justify-between items-center">
                      <span className="text-sm">일치</span>
                      <div className="flex items-center space-x-2">
                        <div className="w-32 bg-muted rounded-full h-2">
                          <div
                            className="bg-green-500 h-2 rounded-full"
                            style={{
                              width: `${
                                (comparisonData.issued.matched / comparisonData.issued.erpTotal) *
                                100
                              }%`,
                            }}
                          />
                        </div>
                        <span className="text-sm font-medium w-12 text-right">
                          {comparisonData.issued.matched}건
                        </span>
                      </div>
                    </div>
                    <div className="flex justify-between items-center">
                      <span className="text-sm">불일치</span>
                      <div className="flex items-center space-x-2">
                        <div className="w-32 bg-muted rounded-full h-2">
                          <div
                            className="bg-red-500 h-2 rounded-full"
                            style={{
                              width: `${
                                (comparisonData.issued.unmatched /
                                  comparisonData.issued.erpTotal) *
                                100
                              }%`,
                            }}
                          />
                        </div>
                        <span className="text-sm font-medium w-12 text-right">
                          {comparisonData.issued.unmatched}건
                        </span>
                      </div>
                    </div>
                    <div className="flex justify-between items-center">
                      <span className="text-sm">ERP만 존재</span>
                      <div className="flex items-center space-x-2">
                        <div className="w-32 bg-muted rounded-full h-2">
                          <div
                            className="bg-yellow-500 h-2 rounded-full"
                            style={{
                              width: `${
                                (comparisonData.issued.erpOnly / comparisonData.issued.erpTotal) *
                                100
                              }%`,
                            }}
                          />
                        </div>
                        <span className="text-sm font-medium w-12 text-right">
                          {comparisonData.issued.erpOnly}건
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>

            {/* Received (Purchase) Comparison */}
            <Card>
              <CardHeader>
                <CardTitle className="text-lg flex items-center">
                  <TrendingDown className="h-5 w-5 mr-2 text-red-500" />
                  매입 세금계산서 비교
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  <div className="grid grid-cols-2 gap-4">
                    <div className="p-3 bg-muted rounded-lg">
                      <p className="text-sm text-muted-foreground">ERP 등록</p>
                      <p className="text-xl font-bold">{comparisonData.received.erpTotal}건</p>
                      <p className="text-sm text-muted-foreground">
                        {formatCurrency(comparisonData.received.erpAmount, { compact: true })}
                      </p>
                    </div>
                    <div className="p-3 bg-muted rounded-lg">
                      <p className="text-sm text-muted-foreground">홈택스</p>
                      <p className="text-xl font-bold">
                        {comparisonData.received.hometaxTotal}건
                      </p>
                      <p className="text-sm text-muted-foreground">
                        {formatCurrency(comparisonData.received.hometaxAmount, { compact: true })}
                      </p>
                    </div>
                  </div>
                  <div className="space-y-2">
                    <div className="flex justify-between items-center">
                      <span className="text-sm">일치</span>
                      <div className="flex items-center space-x-2">
                        <div className="w-32 bg-muted rounded-full h-2">
                          <div
                            className="bg-green-500 h-2 rounded-full"
                            style={{
                              width: `${
                                (comparisonData.received.matched /
                                  comparisonData.received.erpTotal) *
                                100
                              }%`,
                            }}
                          />
                        </div>
                        <span className="text-sm font-medium w-12 text-right">
                          {comparisonData.received.matched}건
                        </span>
                      </div>
                    </div>
                    <div className="flex justify-between items-center">
                      <span className="text-sm">불일치</span>
                      <div className="flex items-center space-x-2">
                        <div className="w-32 bg-muted rounded-full h-2">
                          <div
                            className="bg-red-500 h-2 rounded-full"
                            style={{
                              width: `${
                                (comparisonData.received.unmatched /
                                  comparisonData.received.erpTotal) *
                                100
                              }%`,
                            }}
                          />
                        </div>
                        <span className="text-sm font-medium w-12 text-right">
                          {comparisonData.received.unmatched}건
                        </span>
                      </div>
                    </div>
                    <div className="flex justify-between items-center">
                      <span className="text-sm">홈택스만 존재</span>
                      <div className="flex items-center space-x-2">
                        <div className="w-32 bg-muted rounded-full h-2">
                          <div
                            className="bg-blue-500 h-2 rounded-full"
                            style={{
                              width: `${
                                (comparisonData.received.hometaxOnly /
                                  comparisonData.received.hometaxTotal) *
                                100
                              }%`,
                            }}
                          />
                        </div>
                        <span className="text-sm font-medium w-12 text-right">
                          {comparisonData.received.hometaxOnly}건
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Unmatched Invoices */}
          <Card>
            <CardHeader className="flex flex-row items-center justify-between">
              <CardTitle className="text-lg flex items-center">
                <AlertCircle className="h-5 w-5 mr-2 text-warning" />
                불일치/미매칭 세금계산서
              </CardTitle>
              <div className="flex items-center space-x-2">
                <select
                  className="h-9 rounded-md border border-input bg-background px-3 text-sm"
                  value={directionFilter}
                  onChange={(e) =>
                    setDirectionFilter(e.target.value as "all" | "issued" | "received")
                  }
                >
                  <option value="all">전체</option>
                  <option value="issued">매출</option>
                  <option value="received">매입</option>
                </select>
                <select
                  className="h-9 rounded-md border border-input bg-background px-3 text-sm"
                  value={comparisonFilter}
                  onChange={(e) =>
                    setComparisonFilter(e.target.value as ComparisonStatus | "all")
                  }
                >
                  <option value="all">전체 상태</option>
                  <option value="matched">일치</option>
                  <option value="unmatched">불일치</option>
                  <option value="erp_only">ERP만 존재</option>
                  <option value="hometax_only">홈택스만 존재</option>
                </select>
                <Button variant="outline">
                  <Download className="h-4 w-4 mr-2" />
                  내보내기
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>세금계산서 번호</TableHead>
                    <TableHead>발행일</TableHead>
                    <TableHead>구분</TableHead>
                    <TableHead>거래처</TableHead>
                    <TableHead className="text-right">ERP 금액</TableHead>
                    <TableHead className="text-right">홈택스 금액</TableHead>
                    <TableHead className="text-right">차이</TableHead>
                    <TableHead>상태</TableHead>
                    <TableHead>비고</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredInvoices.map((invoice) => (
                    <TableRow key={invoice.id}>
                      <TableCell className="font-medium">{invoice.invoiceNumber}</TableCell>
                      <TableCell>{formatDate(invoice.issueDate)}</TableCell>
                      <TableCell>
                        <Badge
                          variant={invoice.direction === "issued" ? "default" : "secondary"}
                        >
                          {invoice.direction === "issued" ? "매출" : "매입"}
                        </Badge>
                      </TableCell>
                      <TableCell>{invoice.partnerName}</TableCell>
                      <TableCell className="text-right font-mono">
                        {invoice.erpAmount !== null
                          ? formatCurrency(invoice.erpAmount, { showSymbol: false })
                          : "-"}
                      </TableCell>
                      <TableCell className="text-right font-mono">
                        {invoice.hometaxAmount !== null
                          ? formatCurrency(invoice.hometaxAmount, { showSymbol: false })
                          : "-"}
                      </TableCell>
                      <TableCell
                        className={cn(
                          "text-right font-mono",
                          invoice.difference > 0 && "text-red-600",
                          invoice.difference < 0 && "text-blue-600"
                        )}
                      >
                        {invoice.difference !== 0 ? (
                          <span className="flex items-center justify-end">
                            {invoice.difference > 0 ? (
                              <TrendingUp className="h-4 w-4 mr-1" />
                            ) : (
                              <TrendingDown className="h-4 w-4 mr-1" />
                            )}
                            {formatCurrency(Math.abs(invoice.difference), { showSymbol: false })}
                          </span>
                        ) : (
                          <Minus className="h-4 w-4 mx-auto text-muted-foreground" />
                        )}
                      </TableCell>
                      <TableCell>
                        <Badge variant={comparisonStatusStyles[invoice.status].variant}>
                          {comparisonStatusStyles[invoice.status].label}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {invoice.note}
                      </TableCell>
                    </TableRow>
                  ))}
                  {filteredInvoices.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={9} className="text-center text-muted-foreground py-8">
                        해당 조건에 맞는 세금계산서가 없습니다.
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </>
      )}

      {/* Auth Modal */}
      <Modal isOpen={showAuthModal} onClose={() => setShowAuthModal(false)} title="홈택스 인증">
        <div className="space-y-6">
          {/* Current Auth Status */}
          <div className="p-4 bg-muted rounded-lg">
            <div className="flex items-center justify-between mb-4">
              <span className="font-medium">현재 인증 상태</span>
              <Badge variant={authStatusStyles[authState.status].variant}>
                {authStatusStyles[authState.status].label}
              </Badge>
            </div>
            {authState.status === "connected" && (
              <div className="space-y-2 text-sm">
                <div className="flex justify-between">
                  <span className="text-muted-foreground">회사명</span>
                  <span>{authState.companyName}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">사업자번호</span>
                  <span className="font-mono">{authState.businessNumber}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">인증 방식</span>
                  <span>{authMethodLabels[authState.authMethod]}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">만료일</span>
                  <span>{formatDate(authState.expiresAt)}</span>
                </div>
              </div>
            )}
          </div>

          {/* Auth Methods */}
          <div className="space-y-4">
            <p className="font-medium">인증 방식 선택</p>
            <div className="grid grid-cols-1 gap-3">
              <button
                onClick={handleAuth}
                className="flex items-center p-4 border rounded-lg hover:bg-accent transition-colors text-left"
              >
                <div className="p-2 bg-primary/10 rounded-lg mr-4">
                  <ShieldCheck className="h-6 w-6 text-primary" />
                </div>
                <div>
                  <p className="font-medium">공인인증서</p>
                  <p className="text-sm text-muted-foreground">
                    USB 또는 하드디스크에 저장된 공인인증서로 로그인
                  </p>
                </div>
              </button>
              <button
                onClick={handleAuth}
                className="flex items-center p-4 border rounded-lg hover:bg-accent transition-colors text-left"
              >
                <div className="p-2 bg-green-100 rounded-lg mr-4">
                  <Building2 className="h-6 w-6 text-green-600" />
                </div>
                <div>
                  <p className="font-medium">간편인증 (네이버/카카오)</p>
                  <p className="text-sm text-muted-foreground">
                    네이버 또는 카카오 인증서로 간편 로그인
                  </p>
                </div>
              </button>
            </div>
          </div>

          {/* Warning */}
          <div className="flex items-start space-x-2 p-3 bg-warning/10 rounded-lg">
            <AlertCircle className="h-5 w-5 text-warning flex-shrink-0 mt-0.5" />
            <div className="text-sm">
              <p className="font-medium text-warning">주의사항</p>
              <p className="text-muted-foreground">
                홈택스 인증 정보는 서버에 저장되지 않으며, 동기화 시에만 사용됩니다. 보안을 위해
                브라우저를 닫으면 인증이 해제됩니다.
              </p>
            </div>
          </div>

          {/* Actions */}
          <div className="flex justify-end space-x-2">
            {authState.status === "connected" && (
              <Button variant="outline" onClick={() => toast.info("로그아웃", "인증이 해제되었습니다.")}>
                인증 해제
              </Button>
            )}
            <Button onClick={() => setShowAuthModal(false)}>닫기</Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
