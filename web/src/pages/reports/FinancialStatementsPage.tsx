import { useState } from "react";
import { FileText, Scale, TrendingUp, PieChart, Download, Printer } from "lucide-react";
import { Button, Card, CardContent } from "@/components/ui";
import { cn } from "@/lib/utils";

// Import actual report components
import { BalanceSheetPage } from "./BalanceSheetPage";
import { IncomeStatementPage } from "./IncomeStatementPage";
import { CashFlowStatementPage } from "./CashFlowStatementPage";
import { EquityChangesStatementPage } from "./EquityChangesStatementPage";

type StatementTab = "balance-sheet" | "income-statement" | "cash-flow" | "equity-changes";

interface TabConfig {
  id: StatementTab;
  label: string;
  icon: React.ElementType;
  description: string;
  component: React.ComponentType | null;
}

const tabs: TabConfig[] = [
  {
    id: "balance-sheet",
    label: "재무상태표",
    icon: Scale,
    description: "자산, 부채, 자본의 현황",
    component: BalanceSheetPage,
  },
  {
    id: "income-statement",
    label: "손익계산서",
    icon: TrendingUp,
    description: "수익과 비용의 발생 현황",
    component: IncomeStatementPage,
  },
  {
    id: "cash-flow",
    label: "현금흐름표",
    icon: FileText,
    description: "현금의 유입과 유출 현황",
    component: CashFlowStatementPage,
  },
  {
    id: "equity-changes",
    label: "자본변동표",
    icon: PieChart,
    description: "자본의 변동 내역",
    component: EquityChangesStatementPage,
  },
];

export function FinancialStatementsPage() {
  const [activeTab, setActiveTab] = useState<StatementTab>("balance-sheet");

  const activeTabConfig = tabs.find((tab) => tab.id === activeTab);
  const ActiveComponent = activeTabConfig?.component;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">재무제표</h1>
          <p className="text-muted-foreground">
            기업의 재무 상태와 경영 성과를 확인합니다.
          </p>
        </div>
        <div className="flex items-center space-x-2">
          <Button variant="outline">
            <Download className="h-4 w-4 mr-2" />
            전체 내보내기
          </Button>
          <Button variant="outline" onClick={() => window.print()}>
            <Printer className="h-4 w-4 mr-2" />
            인쇄
          </Button>
        </div>
      </div>

      {/* Tab Navigation */}
      <Card>
        <CardContent className="p-2">
          <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
            {tabs.map((tab) => {
              const Icon = tab.icon;
              const isActive = activeTab === tab.id;
              const isAvailable = tab.component !== null;

              return (
                <button
                  key={tab.id}
                  onClick={() => isAvailable && setActiveTab(tab.id)}
                  disabled={!isAvailable}
                  className={cn(
                    "relative flex flex-col items-center p-4 rounded-lg transition-all",
                    "hover:bg-muted/50 focus:outline-none focus:ring-2 focus:ring-primary/50",
                    isActive && "bg-primary/10 ring-1 ring-primary",
                    !isAvailable && "opacity-50 cursor-not-allowed"
                  )}
                >
                  <Icon
                    className={cn(
                      "h-6 w-6 mb-2",
                      isActive ? "text-primary" : "text-muted-foreground"
                    )}
                  />
                  <span
                    className={cn(
                      "font-medium text-sm",
                      isActive ? "text-primary" : "text-foreground"
                    )}
                  >
                    {tab.label}
                  </span>
                  <span className="text-xs text-muted-foreground mt-1 hidden md:block">
                    {tab.description}
                  </span>
                  {!isAvailable && (
                    <span className="absolute top-2 right-2 text-[10px] bg-muted px-1.5 py-0.5 rounded text-muted-foreground">
                      준비중
                    </span>
                  )}
                </button>
              );
            })}
          </div>
        </CardContent>
      </Card>

      {/* Active Tab Content */}
      <div className="mt-6">
        {ActiveComponent ? (
          <ActiveComponent />
        ) : (
          <Card>
            <CardContent className="py-16 text-center">
              <FileText className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
              <h3 className="text-lg font-semibold mb-2">
                {activeTabConfig?.label} 준비중
              </h3>
              <p className="text-muted-foreground">
                이 기능은 현재 개발 중입니다. 곧 이용하실 수 있습니다.
              </p>
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  );
}
