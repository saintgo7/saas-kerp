import {
  TrendingUp,
  TrendingDown,
  DollarSign,
  FileText,
  Users,
  AlertCircle,
  ArrowRight,
  Calendar,
} from "lucide-react";
import { Link } from "react-router-dom";
import { Card, CardHeader, CardTitle, CardContent, Badge, Button } from "@/components/ui";
import { useAuthStore } from "@/stores";
import { formatCurrency, formatDate } from "@/lib/utils";
import { cn } from "@/lib/utils";

// Mock data for dashboard
const mockStats = {
  revenue: { current: 125000000, previous: 98000000, change: 27.55 },
  expenses: { current: 45000000, previous: 52000000, change: -13.46 },
  profit: { current: 80000000, previous: 46000000, change: 73.91 },
  invoicesPending: 12,
  payrollPending: 5,
};

const mockRecentVouchers = [
  { id: "1", number: "2024-0125", date: "2024-01-15", description: "거래처 매입", amount: 5500000, type: "expense" },
  { id: "2", number: "2024-0124", date: "2024-01-14", description: "제품 판매", amount: 12000000, type: "revenue" },
  { id: "3", number: "2024-0123", date: "2024-01-13", description: "급여 지급", amount: 8500000, type: "expense" },
  { id: "4", number: "2024-0122", date: "2024-01-12", description: "서비스 수익", amount: 3200000, type: "revenue" },
];

const mockPendingTasks = [
  { id: "1", title: "세금계산서 12건 발행 대기", type: "invoice", count: 12, urgent: true },
  { id: "2", title: "전표 승인 대기", type: "voucher", count: 8, urgent: false },
  { id: "3", title: "1월 급여 처리", type: "payroll", count: 1, urgent: true },
  { id: "4", title: "4대보험 신고", type: "insurance", count: 1, urgent: false },
];

interface StatCardProps {
  title: string;
  value: number;
  change: number;
  icon: React.ReactNode;
  prefix?: string;
}

function StatCard({ title, value, change, icon, prefix = "" }: StatCardProps) {
  const isPositive = change >= 0;

  return (
    <Card>
      <CardContent className="p-6">
        <div className="flex items-center justify-between">
          <div className="space-y-1">
            <p className="text-sm text-muted-foreground">{title}</p>
            <p className="text-2xl font-bold">
              {prefix}
              {formatCurrency(value, { compact: true })}
            </p>
            <div className="flex items-center space-x-1">
              {isPositive ? (
                <TrendingUp className="h-4 w-4 text-success" />
              ) : (
                <TrendingDown className="h-4 w-4 text-destructive" />
              )}
              <span
                className={cn(
                  "text-sm font-medium",
                  isPositive ? "text-success" : "text-destructive"
                )}
              >
                {isPositive ? "+" : ""}
                {change.toFixed(1)}%
              </span>
              <span className="text-xs text-muted-foreground">전월 대비</span>
            </div>
          </div>
          <div className="p-3 bg-primary/10 rounded-xl">{icon}</div>
        </div>
      </CardContent>
    </Card>
  );
}

export function DashboardPage() {
  const { user } = useAuthStore();
  const today = new Date();

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between">
        <div>
          <h1 className="text-2xl font-bold">
            안녕하세요, {user?.name || "사용자"}님
          </h1>
          <p className="text-muted-foreground">
            {formatDate(today, { format: "full" })} 기준 현황입니다.
          </p>
        </div>
        <div className="flex items-center space-x-2 mt-4 md:mt-0">
          <Button variant="outline" size="sm">
            <Calendar className="h-4 w-4 mr-2" />
            {formatDate(today, { format: "month" })}
          </Button>
        </div>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          title="이번 달 매출"
          value={mockStats.revenue.current}
          change={mockStats.revenue.change}
          icon={<DollarSign className="h-6 w-6 text-primary" />}
        />
        <StatCard
          title="이번 달 비용"
          value={mockStats.expenses.current}
          change={mockStats.expenses.change}
          icon={<TrendingDown className="h-6 w-6 text-primary" />}
        />
        <StatCard
          title="이번 달 순이익"
          value={mockStats.profit.current}
          change={mockStats.profit.change}
          icon={<TrendingUp className="h-6 w-6 text-primary" />}
        />
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <div className="space-y-1">
                <p className="text-sm text-muted-foreground">대기 중 작업</p>
                <p className="text-2xl font-bold">
                  {mockStats.invoicesPending + mockStats.payrollPending}건
                </p>
                <div className="flex items-center space-x-2">
                  <Badge variant="outline" className="text-xs">
                    세금계산서 {mockStats.invoicesPending}
                  </Badge>
                  <Badge variant="outline" className="text-xs">
                    급여 {mockStats.payrollPending}
                  </Badge>
                </div>
              </div>
              <div className="p-3 bg-warning/10 rounded-xl">
                <AlertCircle className="h-6 w-6 text-warning" />
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Main Content */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Recent Vouchers */}
        <Card className="lg:col-span-2">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-lg">최근 전표</CardTitle>
            <Link to="/accounting/voucher">
              <Button variant="ghost" size="sm">
                전체보기
                <ArrowRight className="h-4 w-4 ml-1" />
              </Button>
            </Link>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {mockRecentVouchers.map((voucher) => (
                <div
                  key={voucher.id}
                  className="flex items-center justify-between p-3 rounded-lg hover:bg-muted/50 transition-colors"
                >
                  <div className="flex items-center space-x-4">
                    <div
                      className={cn(
                        "w-10 h-10 rounded-lg flex items-center justify-center",
                        voucher.type === "revenue"
                          ? "bg-success/10"
                          : "bg-destructive/10"
                      )}
                    >
                      {voucher.type === "revenue" ? (
                        <TrendingUp
                          className={cn(
                            "h-5 w-5",
                            voucher.type === "revenue"
                              ? "text-success"
                              : "text-destructive"
                          )}
                        />
                      ) : (
                        <TrendingDown className="h-5 w-5 text-destructive" />
                      )}
                    </div>
                    <div>
                      <p className="font-medium">{voucher.description}</p>
                      <p className="text-sm text-muted-foreground">
                        {voucher.number} | {formatDate(voucher.date)}
                      </p>
                    </div>
                  </div>
                  <p
                    className={cn(
                      "font-semibold",
                      voucher.type === "revenue"
                        ? "text-success"
                        : "text-destructive"
                    )}
                  >
                    {voucher.type === "revenue" ? "+" : "-"}
                    {formatCurrency(voucher.amount)}
                  </p>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Pending Tasks */}
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">처리 대기 작업</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {mockPendingTasks.map((task) => (
                <div
                  key={task.id}
                  className="flex items-start space-x-3 p-3 rounded-lg hover:bg-muted/50 transition-colors cursor-pointer"
                >
                  <div
                    className={cn(
                      "w-8 h-8 rounded-lg flex items-center justify-center mt-0.5",
                      task.urgent ? "bg-destructive/10" : "bg-muted"
                    )}
                  >
                    {task.type === "invoice" && (
                      <FileText
                        className={cn(
                          "h-4 w-4",
                          task.urgent
                            ? "text-destructive"
                            : "text-muted-foreground"
                        )}
                      />
                    )}
                    {task.type === "voucher" && (
                      <DollarSign
                        className={cn(
                          "h-4 w-4",
                          task.urgent
                            ? "text-destructive"
                            : "text-muted-foreground"
                        )}
                      />
                    )}
                    {task.type === "payroll" && (
                      <Users
                        className={cn(
                          "h-4 w-4",
                          task.urgent
                            ? "text-destructive"
                            : "text-muted-foreground"
                        )}
                      />
                    )}
                    {task.type === "insurance" && (
                      <AlertCircle
                        className={cn(
                          "h-4 w-4",
                          task.urgent
                            ? "text-destructive"
                            : "text-muted-foreground"
                        )}
                      />
                    )}
                  </div>
                  <div className="flex-1">
                    <p className="text-sm font-medium">{task.title}</p>
                    <div className="flex items-center space-x-2 mt-1">
                      {task.urgent && (
                        <Badge variant="destructive" className="text-xs">
                          긴급
                        </Badge>
                      )}
                      <span className="text-xs text-muted-foreground">
                        {task.count}건
                      </span>
                    </div>
                  </div>
                  <ArrowRight className="h-4 w-4 text-muted-foreground" />
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Quick Actions */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">빠른 작업</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <Link to="/accounting/voucher/new">
              <Button variant="outline" className="w-full h-auto py-4 flex-col">
                <DollarSign className="h-6 w-6 mb-2" />
                <span>전표 작성</span>
              </Button>
            </Link>
            <Link to="/invoice/issue">
              <Button variant="outline" className="w-full h-auto py-4 flex-col">
                <FileText className="h-6 w-6 mb-2" />
                <span>세금계산서 발행</span>
              </Button>
            </Link>
            <Link to="/hr/payroll">
              <Button variant="outline" className="w-full h-auto py-4 flex-col">
                <Users className="h-6 w-6 mb-2" />
                <span>급여 관리</span>
              </Button>
            </Link>
            <Link to="/reports/sales">
              <Button variant="outline" className="w-full h-auto py-4 flex-col">
                <TrendingUp className="h-6 w-6 mb-2" />
                <span>매출 분석</span>
              </Button>
            </Link>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
