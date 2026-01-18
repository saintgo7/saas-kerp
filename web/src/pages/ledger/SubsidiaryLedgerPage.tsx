import { useState, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { Download, Printer, Search, Users } from "lucide-react";
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
} from "@/components/ui";
import { DateRangePicker } from "@/components/common";
import { formatCurrency, formatDate } from "@/lib/utils";
import { ledgerApi, accountsApi } from "@/api";
import type { SubsidiaryLedgerData } from "@/api/ledger";
import type { Account, AccountType } from "@/types";
import { ACCOUNT_TYPES } from "@/constants";

// Mock data for development
const mockSubsidiaryData: SubsidiaryLedgerData[] = [
  {
    account: {
      id: "4",
      companyId: "1",
      code: "109",
      name: "외상매출금",
      type: "asset",
      level: 1,
      isActive: true,
    },
    partnerId: "p1",
    partnerName: "A상사",
    openingBalance: 2000000,
    entries: [
      {
        date: "2024-01-05",
        voucherId: "v1",
        voucherNumber: "2024-0001",
        description: "상품매출",
        debitAmount: 1100000,
        creditAmount: 0,
        balance: 3100000,
        partnerId: "p1",
        partnerName: "A상사",
        partnerCode: "C001",
      },
      {
        date: "2024-01-20",
        voucherId: "v5",
        voucherNumber: "2024-0025",
        description: "대금입금",
        debitAmount: 0,
        creditAmount: 1000000,
        balance: 2100000,
        partnerId: "p1",
        partnerName: "A상사",
        partnerCode: "C001",
      },
    ],
    totalDebit: 1100000,
    totalCredit: 1000000,
    closingBalance: 2100000,
  },
  {
    account: {
      id: "4",
      companyId: "1",
      code: "109",
      name: "외상매출금",
      type: "asset",
      level: 1,
      isActive: true,
    },
    partnerId: "p2",
    partnerName: "B기업",
    openingBalance: 5000000,
    entries: [
      {
        date: "2024-01-10",
        voucherId: "v2",
        voucherNumber: "2024-0010",
        description: "상품매출",
        debitAmount: 3300000,
        creditAmount: 0,
        balance: 8300000,
        partnerId: "p2",
        partnerName: "B기업",
        partnerCode: "C002",
      },
    ],
    totalDebit: 3300000,
    totalCredit: 0,
    closingBalance: 8300000,
  },
  {
    account: {
      id: "4",
      companyId: "1",
      code: "109",
      name: "외상매출금",
      type: "asset",
      level: 1,
      isActive: true,
    },
    partnerId: "p3",
    partnerName: "C산업",
    openingBalance: 1500000,
    entries: [
      {
        date: "2024-01-15",
        voucherId: "v3",
        voucherNumber: "2024-0015",
        description: "대금입금",
        debitAmount: 0,
        creditAmount: 1500000,
        balance: 0,
        partnerId: "p3",
        partnerName: "C산업",
        partnerCode: "C003",
      },
    ],
    totalDebit: 0,
    totalCredit: 1500000,
    closingBalance: 0,
  },
];

const subsidiaryAccounts: Account[] = [
  { id: "4", companyId: "1", code: "109", name: "외상매출금", type: "asset", level: 1, isActive: true },
  { id: "6", companyId: "1", code: "202", name: "외상매입금", type: "liability", level: 1, isActive: true },
  { id: "9", companyId: "1", code: "251", name: "선수금", type: "liability", level: 1, isActive: true },
  { id: "10", companyId: "1", code: "131", name: "선급금", type: "asset", level: 1, isActive: true },
];

export function SubsidiaryLedgerPage() {
  const today = new Date();
  const firstDayOfMonth = new Date(today.getFullYear(), today.getMonth(), 1);

  const [startDate, setStartDate] = useState(firstDayOfMonth.toISOString().split("T")[0]);
  const [endDate, setEndDate] = useState(today.toISOString().split("T")[0]);
  const [selectedAccountId, setSelectedAccountId] = useState<string>("");
  const [searchTerm, setSearchTerm] = useState("");
  const [expandedPartners, setExpandedPartners] = useState<Set<string>>(new Set());

  // Fetch subsidiary ledger data grouped by partner
  const {
    data: ledgerResponse,
    isLoading,
    error,
  } = useQuery({
    queryKey: ["ledger", "subsidiary", selectedAccountId, startDate, endDate],
    queryFn: () =>
      ledgerApi.subsidiaryLedgerByPartner(selectedAccountId, { startDate, endDate }),
    enabled: !!selectedAccountId && !!startDate && !!endDate,
  });

  // Use mock data if no real data
  const ledgerDataList = ledgerResponse?.data || (selectedAccountId ? mockSubsidiaryData : []);

  // Filter by search term
  const filteredLedgerData = useMemo(() => {
    if (!searchTerm) return ledgerDataList;
    const term = searchTerm.toLowerCase();
    return ledgerDataList.filter(
      (d) =>
        d.partnerName?.toLowerCase().includes(term) ||
        d.partnerId?.toLowerCase().includes(term)
    );
  }, [ledgerDataList, searchTerm]);

  // Calculate totals
  const totals = useMemo(() => {
    return filteredLedgerData.reduce(
      (acc, data) => ({
        openingBalance: acc.openingBalance + data.openingBalance,
        totalDebit: acc.totalDebit + data.totalDebit,
        totalCredit: acc.totalCredit + data.totalCredit,
        closingBalance: acc.closingBalance + data.closingBalance,
      }),
      { openingBalance: 0, totalDebit: 0, totalCredit: 0, closingBalance: 0 }
    );
  }, [filteredLedgerData]);

  const handleDateChange = (start: string, end: string) => {
    setStartDate(start);
    setEndDate(end);
  };

  const togglePartnerExpand = (partnerId: string) => {
    setExpandedPartners((prev) => {
      const next = new Set(prev);
      if (next.has(partnerId)) {
        next.delete(partnerId);
      } else {
        next.add(partnerId);
      }
      return next;
    });
  };

  const expandAll = () => {
    setExpandedPartners(new Set(filteredLedgerData.map((d) => d.partnerId || "")));
  };

  const collapseAll = () => {
    setExpandedPartners(new Set());
  };

  const getAccountTypeLabel = (type: AccountType) => {
    return ACCOUNT_TYPES.find((t) => t.value === type)?.label || type;
  };

  const selectedAccount = subsidiaryAccounts.find((a) => a.id === selectedAccountId);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">보조원장</h1>
          <p className="text-muted-foreground">
            거래처별 채권/채무 내역을 조회합니다.
          </p>
        </div>
        <div className="flex items-center space-x-2">
          <Button variant="outline">
            <Download className="h-4 w-4 mr-2" />
            내보내기
          </Button>
          <Button variant="outline" onClick={() => window.print()}>
            <Printer className="h-4 w-4 mr-2" />
            인쇄
          </Button>
        </div>
      </div>

      {/* Filters */}
      <Card>
        <CardContent className="p-4">
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            {/* Account Select */}
            <div>
              <label className="block text-sm font-medium mb-1.5">계정과목</label>
              <select
                className="w-full h-9 rounded-md border border-input bg-background px-3 text-sm"
                value={selectedAccountId}
                onChange={(e) => setSelectedAccountId(e.target.value)}
              >
                <option value="">계정과목 선택</option>
                {subsidiaryAccounts.map((account) => (
                  <option key={account.id} value={account.id}>
                    {account.code} {account.name}
                  </option>
                ))}
              </select>
            </div>

            {/* Date Range */}
            <div className="md:col-span-2">
              <DateRangePicker
                startDate={startDate}
                endDate={endDate}
                onDateChange={handleDateChange}
                label="조회기간"
              />
            </div>

            {/* Search */}
            <div>
              <label className="block text-sm font-medium mb-1.5">거래처 검색</label>
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="거래처명 검색..."
                  className="pl-9"
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                />
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Content */}
      {!selectedAccountId ? (
        <Card>
          <CardContent className="py-16 text-center">
            <Users className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
            <p className="text-muted-foreground">계정과목을 선택하세요.</p>
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
      ) : (
        <>
          {/* Summary */}
          <Card>
            <CardHeader className="pb-3">
              <div className="flex items-center justify-between">
                <CardTitle className="text-lg">
                  {selectedAccount && (
                    <>
                      <span className="font-mono mr-2">{selectedAccount.code}</span>
                      {selectedAccount.name}
                    </>
                  )}
                </CardTitle>
                <div className="flex items-center space-x-2">
                  <Badge variant="secondary">
                    {filteredLedgerData.length}개 거래처
                  </Badge>
                  <Button variant="outline" size="sm" onClick={expandAll}>
                    전체 펼치기
                  </Button>
                  <Button variant="outline" size="sm" onClick={collapseAll}>
                    전체 접기
                  </Button>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-4 gap-4">
                <div className="text-center p-3 bg-muted rounded-lg">
                  <p className="text-sm text-muted-foreground">이월잔액 합계</p>
                  <p className="text-lg font-semibold font-mono">
                    {formatCurrency(totals.openingBalance, { showSymbol: false })}
                  </p>
                </div>
                <div className="text-center p-3 bg-blue-50 rounded-lg">
                  <p className="text-sm text-muted-foreground">차변 합계</p>
                  <p className="text-lg font-semibold font-mono text-blue-600">
                    {formatCurrency(totals.totalDebit, { showSymbol: false })}
                  </p>
                </div>
                <div className="text-center p-3 bg-red-50 rounded-lg">
                  <p className="text-sm text-muted-foreground">대변 합계</p>
                  <p className="text-lg font-semibold font-mono text-red-600">
                    {formatCurrency(totals.totalCredit, { showSymbol: false })}
                  </p>
                </div>
                <div className="text-center p-3 bg-primary/10 rounded-lg">
                  <p className="text-sm text-muted-foreground">현재잔액 합계</p>
                  <p className="text-lg font-semibold font-mono">
                    {formatCurrency(totals.closingBalance, { showSymbol: false })}
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Partner List */}
          <div className="space-y-4">
            {filteredLedgerData.length === 0 ? (
              <Card>
                <CardContent className="py-8 text-center text-muted-foreground">
                  해당 기간에 거래 내역이 없습니다.
                </CardContent>
              </Card>
            ) : (
              filteredLedgerData.map((data) => (
                <Card key={data.partnerId}>
                  <CardHeader
                    className="pb-3 cursor-pointer hover:bg-muted/30 transition-colors"
                    onClick={() => togglePartnerExpand(data.partnerId || "")}
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center space-x-3">
                        <span
                          className={`transform transition-transform ${
                            expandedPartners.has(data.partnerId || "") ? "rotate-90" : ""
                          }`}
                        >
                          ▶
                        </span>
                        <div>
                          <CardTitle className="text-base">{data.partnerName}</CardTitle>
                          <p className="text-sm text-muted-foreground">
                            {data.entries.length}건의 거래
                          </p>
                        </div>
                      </div>
                      <div className="flex items-center space-x-6 text-right">
                        <div>
                          <p className="text-xs text-muted-foreground">이월</p>
                          <p className="font-mono">
                            {formatCurrency(data.openingBalance, { showSymbol: false })}
                          </p>
                        </div>
                        <div>
                          <p className="text-xs text-muted-foreground">차변</p>
                          <p className="font-mono text-blue-600">
                            {formatCurrency(data.totalDebit, { showSymbol: false })}
                          </p>
                        </div>
                        <div>
                          <p className="text-xs text-muted-foreground">대변</p>
                          <p className="font-mono text-red-600">
                            {formatCurrency(data.totalCredit, { showSymbol: false })}
                          </p>
                        </div>
                        <div>
                          <p className="text-xs text-muted-foreground">잔액</p>
                          <p className="font-mono font-semibold">
                            {formatCurrency(data.closingBalance, { showSymbol: false })}
                          </p>
                        </div>
                      </div>
                    </div>
                  </CardHeader>

                  {expandedPartners.has(data.partnerId || "") && (
                    <CardContent className="pt-0">
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
                          {/* Opening Balance */}
                          <TableRow className="bg-muted/30">
                            <TableCell>{formatDate(startDate)}</TableCell>
                            <TableCell>-</TableCell>
                            <TableCell className="font-medium">이월잔액</TableCell>
                            <TableCell className="text-right font-mono">-</TableCell>
                            <TableCell className="text-right font-mono">-</TableCell>
                            <TableCell className="text-right font-mono font-semibold">
                              {formatCurrency(data.openingBalance, { showSymbol: false })}
                            </TableCell>
                          </TableRow>

                          {/* Entries */}
                          {data.entries.map((entry, idx) => (
                            <TableRow key={idx}>
                              <TableCell>{formatDate(entry.date)}</TableCell>
                              <TableCell>
                                <a
                                  href={`/accounting/voucher/${entry.voucherId}`}
                                  className="text-primary hover:underline"
                                >
                                  {entry.voucherNumber}
                                </a>
                              </TableCell>
                              <TableCell>{entry.description}</TableCell>
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
                        </TableBody>
                      </Table>
                    </CardContent>
                  )}
                </Card>
              ))
            )}
          </div>
        </>
      )}
    </div>
  );
}
