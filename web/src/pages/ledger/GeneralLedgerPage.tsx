import { useState, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { Download, Printer, Search } from "lucide-react";
import {
  Button,
  Input,
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from "@/components/ui";
import { DateRangePicker } from "@/components/common";
import { formatCurrency, formatDate } from "@/lib/utils";
import { ledgerApi, accountsApi } from "@/api";
import type { GeneralLedgerData } from "@/api/ledger";
import type { Account, AccountType } from "@/types";
import { ACCOUNT_TYPES } from "@/constants";

// Mock data for development
const mockLedgerData: GeneralLedgerData = {
  account: {
    id: "1",
    companyId: "1",
    code: "101",
    name: "현금",
    type: "asset",
    level: 1,
    isActive: true,
  },
  openingBalance: 5000000,
  entries: [
    {
      date: "2024-01-05",
      voucherId: "v1",
      voucherNumber: "2024-0001",
      description: "상품매출 현금입금",
      debitAmount: 1100000,
      creditAmount: 0,
      balance: 6100000,
      entryDescription: "A상사 판매대금",
    },
    {
      date: "2024-01-10",
      voucherId: "v2",
      voucherNumber: "2024-0010",
      description: "사무용품 구입",
      debitAmount: 0,
      creditAmount: 150000,
      balance: 5950000,
      entryDescription: "문구류 구입",
    },
    {
      date: "2024-01-15",
      voucherId: "v3",
      voucherNumber: "2024-0015",
      description: "현금매출",
      debitAmount: 500000,
      creditAmount: 0,
      balance: 6450000,
      entryDescription: "소매매출",
    },
    {
      date: "2024-01-20",
      voucherId: "v4",
      voucherNumber: "2024-0020",
      description: "교통비 지출",
      debitAmount: 0,
      creditAmount: 50000,
      balance: 6400000,
      entryDescription: "출장 택시비",
    },
  ],
  totalDebit: 1600000,
  totalCredit: 200000,
  closingBalance: 6400000,
};

const mockAccounts: Account[] = [
  { id: "1", companyId: "1", code: "101", name: "현금", type: "asset", level: 1, isActive: true },
  { id: "2", companyId: "1", code: "102", name: "보통예금", type: "asset", level: 1, isActive: true },
  { id: "3", companyId: "1", code: "108", name: "받을어음", type: "asset", level: 1, isActive: true },
  { id: "4", companyId: "1", code: "109", name: "외상매출금", type: "asset", level: 1, isActive: true },
  { id: "5", companyId: "1", code: "201", name: "지급어음", type: "liability", level: 1, isActive: true },
  { id: "6", companyId: "1", code: "202", name: "외상매입금", type: "liability", level: 1, isActive: true },
  { id: "7", companyId: "1", code: "401", name: "상품매출", type: "revenue", level: 1, isActive: true },
  { id: "8", companyId: "1", code: "501", name: "상품매입", type: "expense", level: 1, isActive: true },
];

export function GeneralLedgerPage() {
  const today = new Date();
  const firstDayOfMonth = new Date(today.getFullYear(), today.getMonth(), 1);

  const [startDate, setStartDate] = useState(firstDayOfMonth.toISOString().split("T")[0]);
  const [endDate, setEndDate] = useState(today.toISOString().split("T")[0]);
  const [selectedAccountId, setSelectedAccountId] = useState<string>("");
  const [selectedAccountType, setSelectedAccountType] = useState<AccountType | "">("");
  const [searchTerm, setSearchTerm] = useState("");

  // Fetch accounts
  const { data: accountsResponse } = useQuery({
    queryKey: ["accounts", "list", selectedAccountType],
    queryFn: () =>
      accountsApi.list({
        type: selectedAccountType || undefined,
        isActive: true,
        pageSize: 100,
      }),
  });

  const accounts = accountsResponse?.data?.items || mockAccounts;

  // Filter accounts by search term
  const filteredAccounts = useMemo(() => {
    if (!searchTerm) return accounts;
    const term = searchTerm.toLowerCase();
    return accounts.filter(
      (a) =>
        a.code.toLowerCase().includes(term) ||
        a.name.toLowerCase().includes(term)
    );
  }, [accounts, searchTerm]);

  // Fetch ledger data
  const {
    data: ledgerResponse,
    isLoading,
    error,
  } = useQuery({
    queryKey: ["ledger", "general", selectedAccountId, startDate, endDate],
    queryFn: () =>
      ledgerApi.generalLedger(selectedAccountId, { startDate, endDate }),
    enabled: !!selectedAccountId && !!startDate && !!endDate,
  });

  // Use mock data if no real data
  const ledgerData = ledgerResponse?.data || (selectedAccountId ? mockLedgerData : null);

  const handleDateChange = (start: string, end: string) => {
    setStartDate(start);
    setEndDate(end);
  };

  const handleAccountSelect = (accountId: string) => {
    setSelectedAccountId(accountId);
  };

  const getAccountTypeLabel = (type: AccountType) => {
    return ACCOUNT_TYPES.find((t) => t.value === type)?.label || type;
  };

  const handleExport = () => {
    // TODO: Implement export
    console.log("Export general ledger");
  };

  const handlePrint = () => {
    window.print();
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">총계정원장</h1>
          <p className="text-muted-foreground">
            계정과목별 거래 내역을 조회합니다.
          </p>
        </div>
        <div className="flex items-center space-x-2">
          <Button variant="outline" onClick={handleExport}>
            <Download className="h-4 w-4 mr-2" />
            내보내기
          </Button>
          <Button variant="outline" onClick={handlePrint}>
            <Printer className="h-4 w-4 mr-2" />
            인쇄
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
        {/* Account List */}
        <Card className="lg:col-span-1">
          <CardHeader className="pb-3">
            <CardTitle className="text-lg">계정과목</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {/* Search */}
            <div className="relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="계정 검색..."
                className="pl-9"
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
              />
            </div>

            {/* Type Filter */}
            <select
              className="w-full h-9 rounded-md border border-input bg-background px-3 text-sm"
              value={selectedAccountType}
              onChange={(e) => setSelectedAccountType(e.target.value as AccountType | "")}
            >
              <option value="">전체 계정유형</option>
              {ACCOUNT_TYPES.map((type) => (
                <option key={type.value} value={type.value}>
                  {type.label}
                </option>
              ))}
            </select>

            {/* Account List */}
            <div className="max-h-[400px] overflow-y-auto border rounded-lg divide-y">
              {filteredAccounts.map((account) => (
                <button
                  key={account.id}
                  className={`w-full flex items-center justify-between px-3 py-2 text-sm hover:bg-muted text-left ${
                    selectedAccountId === account.id ? "bg-primary/10" : ""
                  }`}
                  onClick={() => handleAccountSelect(account.id)}
                >
                  <div className="flex items-center space-x-2">
                    <span className="font-mono text-xs bg-muted px-1.5 py-0.5 rounded">
                      {account.code}
                    </span>
                    <span className="truncate">{account.name}</span>
                  </div>
                  <span className="text-xs text-muted-foreground">
                    {getAccountTypeLabel(account.type)}
                  </span>
                </button>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Ledger Content */}
        <div className="lg:col-span-3 space-y-6">
          {/* Period Filter */}
          <Card>
            <CardContent className="p-4">
              <div className="flex flex-col md:flex-row md:items-center gap-4">
                <div className="flex-1">
                  <DateRangePicker
                    startDate={startDate}
                    endDate={endDate}
                    onDateChange={handleDateChange}
                    label="조회기간"
                  />
                </div>
                <Button
                  onClick={() => {
                    // Refresh data
                  }}
                  disabled={!selectedAccountId}
                >
                  조회
                </Button>
              </div>
            </CardContent>
          </Card>

          {/* Ledger Data */}
          {!selectedAccountId ? (
            <Card>
              <CardContent className="py-16 text-center">
                <p className="text-muted-foreground">
                  왼쪽에서 계정과목을 선택하세요.
                </p>
              </CardContent>
            </Card>
          ) : isLoading ? (
            <Card>
              <CardContent className="py-16 flex items-center justify-center">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
              </CardContent>
            </Card>
          ) : error ? (
            <Card>
              <CardContent className="py-16 text-center">
                <p className="text-destructive">데이터를 불러올 수 없습니다.</p>
              </CardContent>
            </Card>
          ) : ledgerData ? (
            <>
              {/* Account Summary */}
              <Card>
                <CardHeader className="pb-3">
                  <div className="flex items-center justify-between">
                    <CardTitle className="text-lg">
                      <span className="font-mono mr-2">{ledgerData.account.code}</span>
                      {ledgerData.account.name}
                    </CardTitle>
                    <span className="text-sm text-muted-foreground">
                      {getAccountTypeLabel(ledgerData.account.type)}
                    </span>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-4 gap-4">
                    <div className="text-center p-3 bg-muted rounded-lg">
                      <p className="text-sm text-muted-foreground">이월잔액</p>
                      <p className="text-lg font-semibold font-mono">
                        {formatCurrency(ledgerData.openingBalance, { showSymbol: false })}
                      </p>
                    </div>
                    <div className="text-center p-3 bg-blue-50 rounded-lg">
                      <p className="text-sm text-muted-foreground">차변합계</p>
                      <p className="text-lg font-semibold font-mono text-blue-600">
                        {formatCurrency(ledgerData.totalDebit, { showSymbol: false })}
                      </p>
                    </div>
                    <div className="text-center p-3 bg-red-50 rounded-lg">
                      <p className="text-sm text-muted-foreground">대변합계</p>
                      <p className="text-lg font-semibold font-mono text-red-600">
                        {formatCurrency(ledgerData.totalCredit, { showSymbol: false })}
                      </p>
                    </div>
                    <div className="text-center p-3 bg-primary/10 rounded-lg">
                      <p className="text-sm text-muted-foreground">현재잔액</p>
                      <p className="text-lg font-semibold font-mono">
                        {formatCurrency(ledgerData.closingBalance, { showSymbol: false })}
                      </p>
                    </div>
                  </div>
                </CardContent>
              </Card>

              {/* Transactions Table */}
              <Card>
                <CardHeader>
                  <CardTitle className="text-lg">거래내역</CardTitle>
                </CardHeader>
                <CardContent>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>일자</TableHead>
                        <TableHead>전표번호</TableHead>
                        <TableHead>적요</TableHead>
                        <TableHead className="text-right">차변</TableHead>
                        <TableHead className="text-right">대변</TableHead>
                        <TableHead className="text-right">잔액</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {/* Opening Balance Row */}
                      <TableRow className="bg-muted/50">
                        <TableCell>{formatDate(startDate)}</TableCell>
                        <TableCell>-</TableCell>
                        <TableCell className="font-semibold">이월잔액</TableCell>
                        <TableCell className="text-right font-mono">-</TableCell>
                        <TableCell className="text-right font-mono">-</TableCell>
                        <TableCell className="text-right font-mono font-semibold">
                          {formatCurrency(ledgerData.openingBalance, { showSymbol: false })}
                        </TableCell>
                      </TableRow>

                      {/* Transaction Rows */}
                      {ledgerData.entries.map((entry, index) => (
                        <TableRow key={index}>
                          <TableCell>{formatDate(entry.date)}</TableCell>
                          <TableCell>
                            <a
                              href={`/accounting/voucher/${entry.voucherId}`}
                              className="text-primary hover:underline"
                            >
                              {entry.voucherNumber}
                            </a>
                          </TableCell>
                          <TableCell>
                            <div>
                              <p>{entry.description}</p>
                              {entry.entryDescription && (
                                <p className="text-xs text-muted-foreground">
                                  {entry.entryDescription}
                                </p>
                              )}
                            </div>
                          </TableCell>
                          <TableCell className="text-right font-mono">
                            {entry.debitAmount > 0
                              ? formatCurrency(entry.debitAmount, { showSymbol: false })
                              : "-"}
                          </TableCell>
                          <TableCell className="text-right font-mono">
                            {entry.creditAmount > 0
                              ? formatCurrency(entry.creditAmount, { showSymbol: false })
                              : "-"}
                          </TableCell>
                          <TableCell className="text-right font-mono font-semibold">
                            {formatCurrency(entry.balance, { showSymbol: false })}
                          </TableCell>
                        </TableRow>
                      ))}

                      {/* Total Row */}
                      <TableRow className="bg-muted/50 font-semibold">
                        <TableCell colSpan={3} className="text-right">
                          합계
                        </TableCell>
                        <TableCell className="text-right font-mono text-blue-600">
                          {formatCurrency(ledgerData.totalDebit, { showSymbol: false })}
                        </TableCell>
                        <TableCell className="text-right font-mono text-red-600">
                          {formatCurrency(ledgerData.totalCredit, { showSymbol: false })}
                        </TableCell>
                        <TableCell className="text-right font-mono">
                          {formatCurrency(ledgerData.closingBalance, { showSymbol: false })}
                        </TableCell>
                      </TableRow>
                    </TableBody>
                  </Table>

                  {ledgerData.entries.length === 0 && (
                    <div className="py-8 text-center text-muted-foreground">
                      해당 기간에 거래 내역이 없습니다.
                    </div>
                  )}
                </CardContent>
              </Card>
            </>
          ) : null}
        </div>
      </div>
    </div>
  );
}
