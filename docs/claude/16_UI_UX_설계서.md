# K-ERP UI/UX 설계서

> 작성일: 2026-01-16
> 버전: 1.0
> UI Framework: shadcn/ui + Tailwind CSS

---

## 1. 기술 스택

### 1.1 프론트엔드 스택

| 기술 | 버전 | 용도 |
|------|------|------|
| **React** | 18.x | UI 라이브러리 |
| **TypeScript** | 5.x | 타입 안전성 |
| **Vite** | 5.x | 빌드 도구 |
| **shadcn/ui** | latest | UI 컴포넌트 |
| **Tailwind CSS** | 3.x | 스타일링 |
| **TanStack Table** | 8.x | 데이터 테이블 |
| **TanStack Query** | 5.x | 서버 상태 관리 |
| **React Hook Form** | 7.x | 폼 관리 |
| **Zod** | 3.x | 스키마 검증 |
| **Recharts** | 2.x | 차트 |
| **Lucide React** | latest | 아이콘 |

### 1.2 shadcn/ui 선택 이유

| 장점 | 설명 |
|------|------|
| **커스터마이징** | 소스 코드 완전 소유, 자유로운 수정 |
| **접근성** | Radix UI 기반 WCAG 2.1 준수 |
| **타입 안전성** | TypeScript 완벽 지원 |
| **경량화** | 필요한 컴포넌트만 설치 |
| **디자인 일관성** | 통일된 디자인 시스템 |
| **다크 모드** | 기본 지원 |

---

## 2. 디자인 시스템

### 2.1 컬러 팔레트

```css
/* globals.css */
@layer base {
  :root {
    /* 브랜드 컬러 */
    --primary: 222.2 47.4% 11.2%;      /* 메인 파란색 */
    --primary-foreground: 210 40% 98%;

    /* 시맨틱 컬러 */
    --success: 142.1 76.2% 36.3%;      /* 성공/긍정 - 녹색 */
    --warning: 37.7 92.1% 50.2%;       /* 경고 - 주황색 */
    --destructive: 0 84.2% 60.2%;      /* 위험/오류 - 빨간색 */

    /* 중립 컬러 */
    --background: 0 0% 100%;
    --foreground: 222.2 84% 4.9%;
    --muted: 210 40% 96.1%;
    --muted-foreground: 215.4 16.3% 46.9%;

    /* 테두리/카드 */
    --border: 214.3 31.8% 91.4%;
    --card: 0 0% 100%;
    --card-foreground: 222.2 84% 4.9%;

    /* 사이드바 */
    --sidebar-background: 0 0% 98%;
    --sidebar-foreground: 240 5.3% 26.1%;
    --sidebar-primary: 240 5.9% 10%;
    --sidebar-accent: 240 4.8% 95.9%;

    /* 차트 컬러 */
    --chart-1: 12 76% 61%;             /* 매출 */
    --chart-2: 173 58% 39%;            /* 비용 */
    --chart-3: 197 37% 24%;            /* 이익 */
    --chart-4: 43 74% 66%;             /* 기타1 */
    --chart-5: 27 87% 67%;             /* 기타2 */
  }

  .dark {
    --background: 222.2 84% 4.9%;
    --foreground: 210 40% 98%;
    --primary: 217.2 91.2% 59.8%;
    --muted: 217.2 32.6% 17.5%;
    --muted-foreground: 215 20.2% 65.1%;
    --border: 217.2 32.6% 17.5%;
    --card: 222.2 84% 4.9%;
  }
}
```

### 2.2 회계/ERP 특화 컬러

```typescript
// lib/colors.ts
export const erpColors = {
  // 재무 상태
  profit: "hsl(142.1 76.2% 36.3%)",    // 이익 - 녹색
  loss: "hsl(0 84.2% 60.2%)",          // 손실 - 빨간색
  neutral: "hsl(215.4 16.3% 46.9%)",   // 중립 - 회색

  // 계정 유형
  asset: "hsl(217 91% 60%)",           // 자산 - 파란색
  liability: "hsl(0 72% 51%)",         // 부채 - 빨간색
  equity: "hsl(142 71% 45%)",          // 자본 - 녹색
  revenue: "hsl(262 83% 58%)",         // 수익 - 보라색
  expense: "hsl(25 95% 53%)",          // 비용 - 주황색

  // 상태 표시
  approved: "hsl(142.1 76.2% 36.3%)",  // 승인됨
  pending: "hsl(37.7 92.1% 50.2%)",    // 대기중
  rejected: "hsl(0 84.2% 60.2%)",      // 반려됨
  draft: "hsl(215.4 16.3% 46.9%)",     // 임시저장
}
```

### 2.3 타이포그래피

```css
/* 폰트 설정 */
@layer base {
  :root {
    --font-sans: "Pretendard", -apple-system, BlinkMacSystemFont, system-ui, sans-serif;
    --font-mono: "JetBrains Mono", "Fira Code", monospace;
  }
}

/* 텍스트 스타일 */
.text-display { @apply text-4xl font-bold tracking-tight; }
.text-heading-1 { @apply text-3xl font-semibold tracking-tight; }
.text-heading-2 { @apply text-2xl font-semibold; }
.text-heading-3 { @apply text-xl font-medium; }
.text-body { @apply text-base; }
.text-body-sm { @apply text-sm; }
.text-caption { @apply text-xs text-muted-foreground; }
.text-mono { @apply font-mono text-sm; }  /* 금액, 코드 표시용 */
```

### 2.4 간격 시스템

```typescript
// 8px 기반 간격 시스템
const spacing = {
  0: "0px",
  1: "4px",    // 0.5
  2: "8px",    // 1
  3: "12px",   // 1.5
  4: "16px",   // 2
  5: "20px",   // 2.5
  6: "24px",   // 3
  8: "32px",   // 4
  10: "40px",  // 5
  12: "48px",  // 6
  16: "64px",  // 8
}
```

---

## 3. 레이아웃 구조

### 3.1 전체 레이아웃

```
┌─────────────────────────────────────────────────────────────────┐
│                         Header (64px)                           │
│  [Logo] [Company] [Search...          ] [Noti] [User] [Theme]  │
├────────────┬────────────────────────────────────────────────────┤
│            │                                                    │
│  Sidebar   │                    Main Content                    │
│  (256px)   │                                                    │
│            │  ┌──────────────────────────────────────────────┐  │
│  [메뉴1]   │  │  Breadcrumb: 홈 > 회계 > 전표입력           │  │
│  [메뉴2]   │  ├──────────────────────────────────────────────┤  │
│  [메뉴3]   │  │  Page Header                                 │  │
│    ...     │  │  [제목]                    [액션 버튼들]     │  │
│            │  ├──────────────────────────────────────────────┤  │
│            │  │                                              │  │
│            │  │              Page Content                    │  │
│            │  │                                              │  │
│            │  │                                              │  │
│            │  └──────────────────────────────────────────────┘  │
│            │                                                    │
├────────────┴────────────────────────────────────────────────────┤
│                       Footer (선택적)                            │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 레이아웃 컴포넌트

```tsx
// components/layout/app-layout.tsx
import { SidebarProvider, SidebarTrigger } from "@/components/ui/sidebar"
import { AppSidebar } from "@/components/layout/app-sidebar"
import { AppHeader } from "@/components/layout/app-header"

export function AppLayout({ children }: { children: React.ReactNode }) {
  return (
    <SidebarProvider>
      <AppSidebar />
      <main className="flex-1 flex flex-col min-h-screen">
        <AppHeader />
        <div className="flex-1 p-6">
          {children}
        </div>
      </main>
    </SidebarProvider>
  )
}
```

### 3.3 사이드바 구조

```tsx
// components/layout/app-sidebar.tsx
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuItem,
  SidebarMenuButton,
  SidebarMenuSub,
  SidebarMenuSubItem,
  SidebarFooter,
} from "@/components/ui/sidebar"
import {
  LayoutDashboard,
  Receipt,
  FileText,
  Users,
  Package,
  BarChart3,
  Settings,
  Building2,
  Calculator,
  Wallet,
  FileSpreadsheet,
  UserCog,
  Boxes,
  ShoppingCart,
  TrendingUp,
} from "lucide-react"

const menuItems = [
  {
    title: "대시보드",
    icon: LayoutDashboard,
    url: "/dashboard",
  },
  {
    title: "회계관리",
    icon: Calculator,
    items: [
      { title: "전표입력", url: "/accounting/voucher" },
      { title: "전표조회", url: "/accounting/voucher/list" },
      { title: "계정원장", url: "/accounting/ledger" },
      { title: "재무제표", url: "/accounting/statements" },
      { title: "결산관리", url: "/accounting/closing" },
    ],
  },
  {
    title: "세금관리",
    icon: Receipt,
    items: [
      { title: "세금계산서 발행", url: "/tax/invoice/issue" },
      { title: "세금계산서 조회", url: "/tax/invoice/list" },
      { title: "매입매출장", url: "/tax/purchase-sales" },
      { title: "부가세 신고", url: "/tax/vat" },
    ],
  },
  {
    title: "인사/급여",
    icon: Users,
    items: [
      { title: "사원관리", url: "/hr/employees" },
      { title: "급여계산", url: "/hr/payroll" },
      { title: "급여명세서", url: "/hr/payslip" },
      { title: "4대보험", url: "/hr/insurance" },
      { title: "연말정산", url: "/hr/year-end" },
    ],
  },
  {
    title: "재고/물류",
    icon: Package,
    items: [
      { title: "품목관리", url: "/inventory/items" },
      { title: "입고관리", url: "/inventory/inbound" },
      { title: "출고관리", url: "/inventory/outbound" },
      { title: "재고현황", url: "/inventory/status" },
      { title: "재고실사", url: "/inventory/count" },
    ],
  },
  {
    title: "구매/판매",
    icon: ShoppingCart,
    items: [
      { title: "거래처관리", url: "/trade/partners" },
      { title: "견적관리", url: "/trade/quotes" },
      { title: "발주관리", url: "/trade/orders" },
      { title: "매출관리", url: "/trade/sales" },
    ],
  },
  {
    title: "경영분석",
    icon: BarChart3,
    items: [
      { title: "매출분석", url: "/analytics/sales" },
      { title: "비용분석", url: "/analytics/expenses" },
      { title: "손익분석", url: "/analytics/profit" },
      { title: "KPI 대시보드", url: "/analytics/kpi" },
    ],
  },
  {
    title: "설정",
    icon: Settings,
    items: [
      { title: "회사정보", url: "/settings/company" },
      { title: "계정과목", url: "/settings/accounts" },
      { title: "사용자관리", url: "/settings/users" },
      { title: "권한관리", url: "/settings/permissions" },
    ],
  },
]

export function AppSidebar() {
  return (
    <Sidebar>
      <SidebarContent>
        {menuItems.map((menu) => (
          <SidebarGroup key={menu.title}>
            {menu.items ? (
              <>
                <SidebarGroupLabel>
                  <menu.icon className="mr-2 h-4 w-4" />
                  {menu.title}
                </SidebarGroupLabel>
                <SidebarGroupContent>
                  <SidebarMenu>
                    {menu.items.map((item) => (
                      <SidebarMenuItem key={item.title}>
                        <SidebarMenuButton asChild>
                          <a href={item.url}>{item.title}</a>
                        </SidebarMenuButton>
                      </SidebarMenuItem>
                    ))}
                  </SidebarMenu>
                </SidebarGroupContent>
              </>
            ) : (
              <SidebarMenu>
                <SidebarMenuItem>
                  <SidebarMenuButton asChild>
                    <a href={menu.url}>
                      <menu.icon className="mr-2 h-4 w-4" />
                      {menu.title}
                    </a>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              </SidebarMenu>
            )}
          </SidebarGroup>
        ))}
      </SidebarContent>
      <SidebarFooter>
        <div className="p-4 text-xs text-muted-foreground">
          K-ERP v1.0.0
        </div>
      </SidebarFooter>
    </Sidebar>
  )
}
```

---

## 4. 핵심 화면 설계

### 4.1 대시보드 (Dashboard)

```
┌─────────────────────────────────────────────────────────────────┐
│  대시보드                                    [기간: 이번 달 ▼]  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌───────────┐ │
│  │ 💰 매출     │ │ 📉 비용     │ │ 📈 순이익   │ │ 📄 미수금  │ │
│  │             │ │             │ │             │ │           │ │
│  │ ₩125,000,000│ │ ₩98,000,000 │ │ ₩27,000,000 │ │₩15,000,000│ │
│  │ ▲ 12.5%    │ │ ▼ 3.2%     │ │ ▲ 25.1%    │ │ ▼ 8.3%   │ │
│  └─────────────┘ └─────────────┘ └─────────────┘ └───────────┘ │
│                                                                 │
│  ┌────────────────────────────────┐ ┌─────────────────────────┐ │
│  │        매출/비용 추이          │ │     계정별 비중         │ │
│  │  [Area Chart]                  │ │   [Pie Chart]          │ │
│  │                                │ │                         │ │
│  │  ████████████████████████     │ │      ████               │ │
│  │  ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓     │ │    ████████             │ │
│  │  1월 2월 3월 4월 5월 6월      │ │      ████               │ │
│  └────────────────────────────────┘ └─────────────────────────┘ │
│                                                                 │
│  ┌────────────────────────────────┐ ┌─────────────────────────┐ │
│  │      최근 전표                 │ │    알림/할 일           │ │
│  │  ┌──────────────────────────┐ │ │  ⚠️ 세금계산서 3건 미발행│ │
│  │  │ 2024-01-15 매입 1,000,000│ │ │  📅 부가세 신고 D-5     │ │
│  │  │ 2024-01-14 매출 2,500,000│ │ │  💰 미수금 회수 필요    │ │
│  │  │ 2024-01-13 급여 5,000,000│ │ │  📋 결산 마감 예정      │ │
│  │  └──────────────────────────┘ │ │                         │ │
│  │  [더보기]                      │ │  [전체보기]             │ │
│  └────────────────────────────────┘ └─────────────────────────┘ │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 대시보드 컴포넌트

```tsx
// app/dashboard/page.tsx
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent
} from "@/components/ui/chart"
import { Area, AreaChart, XAxis, YAxis, CartesianGrid } from "recharts"
import { TrendingUp, TrendingDown, Wallet, Receipt, PiggyBank, AlertCircle } from "lucide-react"

// KPI 카드 컴포넌트
function KPICard({
  title,
  value,
  change,
  icon: Icon,
  trend
}: {
  title: string
  value: string
  change: string
  icon: React.ElementType
  trend: "up" | "down"
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">
          {title}
        </CardTitle>
        <Icon className="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold font-mono">{value}</div>
        <div className={`flex items-center text-xs ${
          trend === "up" ? "text-green-600" : "text-red-600"
        }`}>
          {trend === "up" ? <TrendingUp className="h-3 w-3 mr-1" /> : <TrendingDown className="h-3 w-3 mr-1" />}
          {change} 전월 대비
        </div>
      </CardContent>
    </Card>
  )
}

export default function DashboardPage() {
  return (
    <div className="space-y-6">
      {/* KPI 카드 그리드 */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <KPICard
          title="매출"
          value="₩125,000,000"
          change="+12.5%"
          icon={Wallet}
          trend="up"
        />
        <KPICard
          title="비용"
          value="₩98,000,000"
          change="-3.2%"
          icon={Receipt}
          trend="down"
        />
        <KPICard
          title="순이익"
          value="₩27,000,000"
          change="+25.1%"
          icon={PiggyBank}
          trend="up"
        />
        <KPICard
          title="미수금"
          value="₩15,000,000"
          change="-8.3%"
          icon={AlertCircle}
          trend="down"
        />
      </div>

      {/* 차트 영역 */}
      <div className="grid gap-4 md:grid-cols-7">
        <Card className="col-span-4">
          <CardHeader>
            <CardTitle>매출/비용 추이</CardTitle>
          </CardHeader>
          <CardContent>
            <RevenueChart />
          </CardContent>
        </Card>
        <Card className="col-span-3">
          <CardHeader>
            <CardTitle>계정별 비중</CardTitle>
          </CardHeader>
          <CardContent>
            <AccountPieChart />
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
```

### 4.2 전표 입력 (Voucher Entry)

```
┌─────────────────────────────────────────────────────────────────┐
│  전표입력                          [임시저장] [전표등록] [닫기] │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─── 전표 기본정보 ───────────────────────────────────────────┐ │
│  │                                                             │ │
│  │  전표일자        전표유형           전표번호               │ │
│  │  [2024-01-15]   [일반전표    ▼]    [자동생성]             │ │
│  │                                                             │ │
│  │  적요                                                       │ │
│  │  [1월 사무용품 구매                                    ]   │ │
│  │                                                             │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ┌─── 분개 내역 ───────────────────────────────────────────────┐ │
│  │  [+ 차변 추가] [+ 대변 추가]                    대차 균형 ✓│ │
│  │                                                             │ │
│  │  ┌───────────────────────────────────────────────────────┐ │ │
│  │  │ 구분 │ 계정과목      │ 거래처    │ 차변금액  │ 대변금액│ │ │
│  │  ├───────────────────────────────────────────────────────┤ │ │
│  │  │ 차변 │ 소모품비 [▼] │ [선택]    │ 100,000  │         │ │ │
│  │  │ 대변 │ 보통예금 [▼] │ 국민은행  │          │ 100,000 │ │ │
│  │  └───────────────────────────────────────────────────────┘ │ │
│  │                                                             │ │
│  │  차변 합계: ₩100,000    대변 합계: ₩100,000    차액: ₩0    │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ┌─── 첨부파일 ────────────────────────────────────────────────┐ │
│  │  [📎 파일 추가]                                             │ │
│  │  📄 영수증_20240115.pdf  (125KB)  [×]                      │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 전표 입력 컴포넌트

```tsx
// app/accounting/voucher/page.tsx
import { useForm, useFieldArray } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Calendar } from "@/components/ui/calendar"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { CalendarIcon, Plus, Trash2, Check, X } from "lucide-react"
import { cn } from "@/lib/utils"
import { format } from "date-fns"
import { ko } from "date-fns/locale"

// 스키마 정의
const voucherEntrySchema = z.object({
  entryType: z.enum(["debit", "credit"]),
  accountCode: z.string().min(1, "계정과목을 선택하세요"),
  partnerId: z.string().optional(),
  amount: z.number().min(1, "금액을 입력하세요"),
  description: z.string().optional(),
})

const voucherSchema = z.object({
  voucherDate: z.date(),
  voucherType: z.enum(["general", "sales", "purchase", "payment", "receipt"]),
  description: z.string().min(1, "적요를 입력하세요"),
  entries: z.array(voucherEntrySchema).min(2, "최소 2개 이상의 분개가 필요합니다"),
}).refine((data) => {
  const debitSum = data.entries
    .filter(e => e.entryType === "debit")
    .reduce((sum, e) => sum + e.amount, 0)
  const creditSum = data.entries
    .filter(e => e.entryType === "credit")
    .reduce((sum, e) => sum + e.amount, 0)
  return debitSum === creditSum
}, { message: "차변과 대변의 합계가 일치해야 합니다" })

export function VoucherEntryForm() {
  const form = useForm<z.infer<typeof voucherSchema>>({
    resolver: zodResolver(voucherSchema),
    defaultValues: {
      voucherDate: new Date(),
      voucherType: "general",
      description: "",
      entries: [
        { entryType: "debit", accountCode: "", amount: 0 },
        { entryType: "credit", accountCode: "", amount: 0 },
      ],
    },
  })

  const { fields, append, remove } = useFieldArray({
    control: form.control,
    name: "entries",
  })

  // 차대변 합계 계산
  const entries = form.watch("entries")
  const debitTotal = entries
    .filter(e => e.entryType === "debit")
    .reduce((sum, e) => sum + (e.amount || 0), 0)
  const creditTotal = entries
    .filter(e => e.entryType === "credit")
    .reduce((sum, e) => sum + (e.amount || 0), 0)
  const isBalanced = debitTotal === creditTotal && debitTotal > 0

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
        {/* 전표 기본정보 */}
        <Card>
          <CardHeader>
            <CardTitle>전표 기본정보</CardTitle>
          </CardHeader>
          <CardContent className="grid gap-4 md:grid-cols-3">
            {/* 전표일자 */}
            <FormField
              control={form.control}
              name="voucherDate"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>전표일자</FormLabel>
                  <Popover>
                    <PopoverTrigger asChild>
                      <FormControl>
                        <Button
                          variant="outline"
                          className={cn(
                            "w-full pl-3 text-left font-normal",
                            !field.value && "text-muted-foreground"
                          )}
                        >
                          {field.value ? (
                            format(field.value, "yyyy-MM-dd", { locale: ko })
                          ) : (
                            <span>날짜 선택</span>
                          )}
                          <CalendarIcon className="ml-auto h-4 w-4 opacity-50" />
                        </Button>
                      </FormControl>
                    </PopoverTrigger>
                    <PopoverContent className="w-auto p-0" align="start">
                      <Calendar
                        mode="single"
                        selected={field.value}
                        onSelect={field.onChange}
                        locale={ko}
                      />
                    </PopoverContent>
                  </Popover>
                </FormItem>
              )}
            />

            {/* 전표유형 */}
            <FormField
              control={form.control}
              name="voucherType"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>전표유형</FormLabel>
                  <Select onValueChange={field.onChange} defaultValue={field.value}>
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder="전표유형 선택" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value="general">일반전표</SelectItem>
                      <SelectItem value="sales">매출전표</SelectItem>
                      <SelectItem value="purchase">매입전표</SelectItem>
                      <SelectItem value="payment">지급전표</SelectItem>
                      <SelectItem value="receipt">입금전표</SelectItem>
                    </SelectContent>
                  </Select>
                </FormItem>
              )}
            />

            {/* 적요 */}
            <FormField
              control={form.control}
              name="description"
              render={({ field }) => (
                <FormItem className="md:col-span-3">
                  <FormLabel>적요</FormLabel>
                  <FormControl>
                    <Input placeholder="전표 설명을 입력하세요" {...field} />
                  </FormControl>
                </FormItem>
              )}
            />
          </CardContent>
        </Card>

        {/* 분개 내역 */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle>분개 내역</CardTitle>
            <div className="flex items-center gap-2">
              {isBalanced ? (
                <span className="flex items-center text-sm text-green-600">
                  <Check className="h-4 w-4 mr-1" /> 대차 균형
                </span>
              ) : (
                <span className="flex items-center text-sm text-red-600">
                  <X className="h-4 w-4 mr-1" /> 대차 불균형
                </span>
              )}
            </div>
          </CardHeader>
          <CardContent>
            {/* 분개 테이블 */}
            <div className="border rounded-md">
              <table className="w-full">
                <thead className="bg-muted">
                  <tr>
                    <th className="p-2 text-left text-sm font-medium">구분</th>
                    <th className="p-2 text-left text-sm font-medium">계정과목</th>
                    <th className="p-2 text-left text-sm font-medium">거래처</th>
                    <th className="p-2 text-right text-sm font-medium">차변금액</th>
                    <th className="p-2 text-right text-sm font-medium">대변금액</th>
                    <th className="p-2 w-10"></th>
                  </tr>
                </thead>
                <tbody>
                  {fields.map((field, index) => (
                    <VoucherEntryRow
                      key={field.id}
                      index={index}
                      control={form.control}
                      onRemove={() => remove(index)}
                    />
                  ))}
                </tbody>
                <tfoot className="bg-muted/50">
                  <tr>
                    <td colSpan={3} className="p-2 text-right font-medium">합계</td>
                    <td className="p-2 text-right font-mono font-bold">
                      {debitTotal.toLocaleString()}
                    </td>
                    <td className="p-2 text-right font-mono font-bold">
                      {creditTotal.toLocaleString()}
                    </td>
                    <td></td>
                  </tr>
                </tfoot>
              </table>
            </div>

            {/* 분개 추가 버튼 */}
            <div className="flex gap-2 mt-4">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => append({ entryType: "debit", accountCode: "", amount: 0 })}
              >
                <Plus className="h-4 w-4 mr-1" /> 차변 추가
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => append({ entryType: "credit", accountCode: "", amount: 0 })}
              >
                <Plus className="h-4 w-4 mr-1" /> 대변 추가
              </Button>
            </div>
          </CardContent>
        </Card>

        {/* 버튼 영역 */}
        <div className="flex justify-end gap-2">
          <Button type="button" variant="outline">임시저장</Button>
          <Button type="submit" disabled={!isBalanced}>전표등록</Button>
        </div>
      </form>
    </Form>
  )
}
```

### 4.3 데이터 테이블 (전표 목록)

```
┌─────────────────────────────────────────────────────────────────┐
│  전표조회                                                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─── 검색 필터 ───────────────────────────────────────────────┐ │
│  │  기간: [2024-01-01] ~ [2024-01-31]  전표유형: [전체 ▼]     │ │
│  │  계정: [                  ▼]  거래처: [             ▼]     │ │
│  │  적요: [검색어 입력...                    ]  [검색] [초기화]│ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ┌─── 전표 목록 ───────────────────────────────────────────────┐ │
│  │  [☐ 전체선택]  선택: 0건   [엑셀 다운로드] [선택 삭제]     │ │
│  │                                                             │ │
│  │  ┌─────────────────────────────────────────────────────────┐│ │
│  │  │☐│전표번호 │전표일자  │유형│적요          │차변    │대변 ││ │
│  │  ├─────────────────────────────────────────────────────────┤│ │
│  │  │☐│V2401001│2024-01-15│일반│사무용품 구매 │100,000│    ││ │
│  │  │☐│V2401002│2024-01-14│매출│A사 제품판매  │      │2,500,000││
│  │  │☐│V2401003│2024-01-13│급여│1월 급여지급  │5,000,000│   ││ │
│  │  │☐│V2401004│2024-01-12│매입│원자재 구매   │800,000│    ││ │
│  │  └─────────────────────────────────────────────────────────┘│ │
│  │                                                             │ │
│  │  ◀ 1 2 3 4 5 ... 10 ▶   페이지당 [20 ▼]건  총 195건       │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 데이터 테이블 컴포넌트

```tsx
// components/voucher/voucher-table.tsx
"use client"

import * as React from "react"
import {
  ColumnDef,
  ColumnFiltersState,
  SortingState,
  VisibilityState,
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from "@tanstack/react-table"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Checkbox } from "@/components/ui/checkbox"
import { Badge } from "@/components/ui/badge"
import {
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  ChevronsLeft,
  ChevronsRight,
  Download,
  Trash2,
  Search,
  SlidersHorizontal,
} from "lucide-react"

// 전표 타입
interface Voucher {
  id: string
  voucherNo: string
  voucherDate: string
  voucherType: "general" | "sales" | "purchase" | "payment" | "receipt"
  description: string
  debitAmount: number
  creditAmount: number
  status: "draft" | "approved" | "rejected"
}

// 전표유형 뱃지
const voucherTypeBadge = {
  general: { label: "일반", variant: "secondary" as const },
  sales: { label: "매출", variant: "default" as const },
  purchase: { label: "매입", variant: "outline" as const },
  payment: { label: "지급", variant: "destructive" as const },
  receipt: { label: "입금", variant: "default" as const },
}

// 컬럼 정의
const columns: ColumnDef<Voucher>[] = [
  {
    id: "select",
    header: ({ table }) => (
      <Checkbox
        checked={table.getIsAllPageRowsSelected()}
        onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
        aria-label="전체 선택"
      />
    ),
    cell: ({ row }) => (
      <Checkbox
        checked={row.getIsSelected()}
        onCheckedChange={(value) => row.toggleSelected(!!value)}
        aria-label="행 선택"
      />
    ),
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: "voucherNo",
    header: "전표번호",
    cell: ({ row }) => (
      <span className="font-mono text-sm">{row.getValue("voucherNo")}</span>
    ),
  },
  {
    accessorKey: "voucherDate",
    header: "전표일자",
    cell: ({ row }) => row.getValue("voucherDate"),
  },
  {
    accessorKey: "voucherType",
    header: "유형",
    cell: ({ row }) => {
      const type = row.getValue("voucherType") as keyof typeof voucherTypeBadge
      const badge = voucherTypeBadge[type]
      return <Badge variant={badge.variant}>{badge.label}</Badge>
    },
  },
  {
    accessorKey: "description",
    header: "적요",
    cell: ({ row }) => (
      <span className="max-w-[200px] truncate">{row.getValue("description")}</span>
    ),
  },
  {
    accessorKey: "debitAmount",
    header: () => <div className="text-right">차변금액</div>,
    cell: ({ row }) => {
      const amount = row.getValue("debitAmount") as number
      return (
        <div className="text-right font-mono">
          {amount > 0 ? amount.toLocaleString() : ""}
        </div>
      )
    },
  },
  {
    accessorKey: "creditAmount",
    header: () => <div className="text-right">대변금액</div>,
    cell: ({ row }) => {
      const amount = row.getValue("creditAmount") as number
      return (
        <div className="text-right font-mono">
          {amount > 0 ? amount.toLocaleString() : ""}
        </div>
      )
    },
  },
]

export function VoucherDataTable({ data }: { data: Voucher[] }) {
  const [sorting, setSorting] = React.useState<SortingState>([])
  const [columnFilters, setColumnFilters] = React.useState<ColumnFiltersState>([])
  const [columnVisibility, setColumnVisibility] = React.useState<VisibilityState>({})
  const [rowSelection, setRowSelection] = React.useState({})

  const table = useReactTable({
    data,
    columns,
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    onColumnVisibilityChange: setColumnVisibility,
    onRowSelectionChange: setRowSelection,
    state: {
      sorting,
      columnFilters,
      columnVisibility,
      rowSelection,
    },
  })

  return (
    <div className="space-y-4">
      {/* 필터 및 액션 */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <div className="relative">
            <Search className="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="적요 검색..."
              value={(table.getColumn("description")?.getFilterValue() as string) ?? ""}
              onChange={(event) =>
                table.getColumn("description")?.setFilterValue(event.target.value)
              }
              className="pl-8 w-[250px]"
            />
          </div>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" size="sm">
                <SlidersHorizontal className="mr-2 h-4 w-4" />
                컬럼
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              {table
                .getAllColumns()
                .filter((column) => column.getCanHide())
                .map((column) => (
                  <DropdownMenuCheckboxItem
                    key={column.id}
                    className="capitalize"
                    checked={column.getIsVisible()}
                    onCheckedChange={(value) => column.toggleVisibility(!!value)}
                  >
                    {column.id}
                  </DropdownMenuCheckboxItem>
                ))}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm">
            <Download className="mr-2 h-4 w-4" />
            엑셀 다운로드
          </Button>
          {table.getFilteredSelectedRowModel().rows.length > 0 && (
            <Button variant="destructive" size="sm">
              <Trash2 className="mr-2 h-4 w-4" />
              선택 삭제 ({table.getFilteredSelectedRowModel().rows.length})
            </Button>
          )}
        </div>
      </div>

      {/* 테이블 */}
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <TableHead key={header.id}>
                    {header.isPlaceholder
                      ? null
                      : flexRender(header.column.columnDef.header, header.getContext())}
                  </TableHead>
                ))}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {table.getRowModel().rows?.length ? (
              table.getRowModel().rows.map((row) => (
                <TableRow key={row.id} data-state={row.getIsSelected() && "selected"}>
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id}>
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell colSpan={columns.length} className="h-24 text-center">
                  검색 결과가 없습니다.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>

      {/* 페이지네이션 */}
      <div className="flex items-center justify-between">
        <div className="text-sm text-muted-foreground">
          {table.getFilteredSelectedRowModel().rows.length}개 선택 /
          총 {table.getFilteredRowModel().rows.length}건
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => table.setPageIndex(0)}
            disabled={!table.getCanPreviousPage()}
          >
            <ChevronsLeft className="h-4 w-4" />
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => table.previousPage()}
            disabled={!table.getCanPreviousPage()}
          >
            <ChevronLeft className="h-4 w-4" />
          </Button>
          <span className="text-sm">
            {table.getState().pagination.pageIndex + 1} / {table.getPageCount()}
          </span>
          <Button
            variant="outline"
            size="sm"
            onClick={() => table.nextPage()}
            disabled={!table.getCanNextPage()}
          >
            <ChevronRight className="h-4 w-4" />
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => table.setPageIndex(table.getPageCount() - 1)}
            disabled={!table.getCanNextPage()}
          >
            <ChevronsRight className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  )
}
```

### 4.4 세금계산서 발행 화면

```
┌─────────────────────────────────────────────────────────────────┐
│  세금계산서 발행                                    [발행] [취소]│
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────────────┬───────────────────────────────────┐ │
│  │      공급자 정보        │          공급받는자 정보          │ │
│  ├─────────────────────────┼───────────────────────────────────┤ │
│  │ 등록번호: 123-45-67890 │ 등록번호: [          ] [조회]    │ │
│  │ 상    호: (주)K-ERP    │ 상    호: [자동입력             ] │ │
│  │ 대표자명: 홍길동        │ 대표자명: [자동입력             ] │ │
│  │ 사업장주소: 서울시...   │ 사업장주소: [자동입력           ] │ │
│  │ 업    태: 서비스업      │ 업    태: [자동입력             ] │ │
│  │ 종    목: 소프트웨어    │ 종    목: [자동입력             ] │ │
│  │ 이메일: tax@kerp.co.kr │ 이메일: [                       ] │ │
│  └─────────────────────────┴───────────────────────────────────┘ │
│                                                                 │
│  ┌─── 품목 내역 ───────────────────────────────────────────────┐ │
│  │  작성일자: [2024-01-15]                      [+ 품목 추가] │ │
│  │                                                             │ │
│  │  │월일│품  목│규격│수량│단가    │공급가액  │세액    │비고│ │
│  │  ├──────────────────────────────────────────────────────────┤ │
│  │  │0115│ERP 라이선스│   │ 1 │1,000,000│1,000,000│100,000│   │ │
│  │  │0115│유지보수   │연간│ 1 │  200,000│  200,000│ 20,000│   │ │
│  │  └──────────────────────────────────────────────────────────┘ │
│  │                                                             │ │
│  │  공급가액 합계: ₩1,200,000   세액 합계: ₩120,000            │ │
│  │  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━              │ │
│  │  총 합계금액: ₩1,320,000                                    │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 5. 컴포넌트 라이브러리

### 5.1 사용할 shadcn/ui 컴포넌트 목록

#### 기본 컴포넌트

| 컴포넌트 | 용도 | 우선순위 |
|---------|------|---------|
| **Button** | 모든 버튼 액션 | 필수 |
| **Input** | 텍스트 입력 | 필수 |
| **Label** | 폼 라벨 | 필수 |
| **Select** | 드롭다운 선택 | 필수 |
| **Checkbox** | 다중 선택, 테이블 선택 | 필수 |
| **Radio Group** | 단일 선택 | 필수 |
| **Switch** | 토글 설정 | 필수 |
| **Textarea** | 긴 텍스트 입력 | 필수 |

#### 레이아웃 컴포넌트

| 컴포넌트 | 용도 | 우선순위 |
|---------|------|---------|
| **Card** | 콘텐츠 그룹핑 | 필수 |
| **Sidebar** | 네비게이션 | 필수 |
| **Separator** | 구분선 | 필수 |
| **Tabs** | 탭 네비게이션 | 필수 |
| **Accordion** | 접이식 콘텐츠 | 권장 |
| **Collapsible** | 접기/펼치기 | 권장 |

#### 오버레이 컴포넌트

| 컴포넌트 | 용도 | 우선순위 |
|---------|------|---------|
| **Dialog** | 모달 팝업 | 필수 |
| **Sheet** | 사이드 패널 | 필수 |
| **Popover** | 툴팁/팝업 | 필수 |
| **Dropdown Menu** | 드롭다운 메뉴 | 필수 |
| **Alert Dialog** | 확인/취소 다이얼로그 | 필수 |
| **Tooltip** | 마우스 오버 설명 | 필수 |
| **Toast** | 알림 메시지 | 필수 |

#### 데이터 표시 컴포넌트

| 컴포넌트 | 용도 | 우선순위 |
|---------|------|---------|
| **Table** | 데이터 테이블 | 필수 |
| **Badge** | 상태 표시 | 필수 |
| **Avatar** | 사용자 프로필 | 필수 |
| **Progress** | 진행률 표시 | 권장 |
| **Skeleton** | 로딩 상태 | 권장 |

#### 폼 컴포넌트

| 컴포넌트 | 용도 | 우선순위 |
|---------|------|---------|
| **Form** | 폼 유효성 검사 | 필수 |
| **Calendar** | 날짜 선택 | 필수 |
| **Date Picker** | 날짜 입력 | 필수 |
| **Combobox** | 검색 가능 선택 | 필수 |

#### 차트 컴포넌트

| 컴포넌트 | 용도 | 우선순위 |
|---------|------|---------|
| **Chart (Bar)** | 막대 차트 | 필수 |
| **Chart (Line)** | 선 차트 | 필수 |
| **Chart (Area)** | 영역 차트 | 필수 |
| **Chart (Pie)** | 원형 차트 | 필수 |

### 5.2 커스텀 컴포넌트

#### 금액 입력 컴포넌트

```tsx
// components/ui/currency-input.tsx
import * as React from "react"
import { Input } from "@/components/ui/input"
import { cn } from "@/lib/utils"

interface CurrencyInputProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, "onChange"> {
  value: number
  onChange: (value: number) => void
  currency?: string
}

export function CurrencyInput({
  value,
  onChange,
  currency = "₩",
  className,
  ...props
}: CurrencyInputProps) {
  const [displayValue, setDisplayValue] = React.useState(
    value ? value.toLocaleString() : ""
  )

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const rawValue = e.target.value.replace(/[^\d]/g, "")
    const numValue = parseInt(rawValue, 10) || 0
    setDisplayValue(numValue.toLocaleString())
    onChange(numValue)
  }

  return (
    <div className="relative">
      <span className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground">
        {currency}
      </span>
      <Input
        {...props}
        value={displayValue}
        onChange={handleChange}
        className={cn("pl-8 text-right font-mono", className)}
      />
    </div>
  )
}
```

#### 계정과목 선택 컴포넌트

```tsx
// components/ui/account-select.tsx
import * as React from "react"
import { Check, ChevronsUpDown } from "lucide-react"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"

interface Account {
  code: string
  name: string
  category: "asset" | "liability" | "equity" | "revenue" | "expense"
}

interface AccountSelectProps {
  accounts: Account[]
  value: string
  onValueChange: (value: string) => void
  placeholder?: string
}

const categoryLabels = {
  asset: "자산",
  liability: "부채",
  equity: "자본",
  revenue: "수익",
  expense: "비용",
}

export function AccountSelect({
  accounts,
  value,
  onValueChange,
  placeholder = "계정과목 선택",
}: AccountSelectProps) {
  const [open, setOpen] = React.useState(false)
  const selected = accounts.find((account) => account.code === value)

  // 카테고리별 그룹핑
  const groupedAccounts = accounts.reduce((acc, account) => {
    const category = account.category
    if (!acc[category]) acc[category] = []
    acc[category].push(account)
    return acc
  }, {} as Record<string, Account[]>)

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className="w-full justify-between"
        >
          {selected ? (
            <span>
              <span className="font-mono text-muted-foreground mr-2">
                {selected.code}
              </span>
              {selected.name}
            </span>
          ) : (
            <span className="text-muted-foreground">{placeholder}</span>
          )}
          <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[300px] p-0">
        <Command>
          <CommandInput placeholder="계정과목 검색..." />
          <CommandList>
            <CommandEmpty>검색 결과가 없습니다.</CommandEmpty>
            {Object.entries(groupedAccounts).map(([category, items]) => (
              <CommandGroup
                key={category}
                heading={categoryLabels[category as keyof typeof categoryLabels]}
              >
                {items.map((account) => (
                  <CommandItem
                    key={account.code}
                    value={`${account.code} ${account.name}`}
                    onSelect={() => {
                      onValueChange(account.code)
                      setOpen(false)
                    }}
                  >
                    <Check
                      className={cn(
                        "mr-2 h-4 w-4",
                        value === account.code ? "opacity-100" : "opacity-0"
                      )}
                    />
                    <span className="font-mono text-muted-foreground mr-2">
                      {account.code}
                    </span>
                    {account.name}
                  </CommandItem>
                ))}
              </CommandGroup>
            ))}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  )
}
```

#### 상태 뱃지 컴포넌트

```tsx
// components/ui/status-badge.tsx
import { Badge } from "@/components/ui/badge"
import { cn } from "@/lib/utils"
import {
  CheckCircle,
  Clock,
  XCircle,
  FileEdit,
  AlertTriangle
} from "lucide-react"

type Status = "draft" | "pending" | "approved" | "rejected" | "warning"

interface StatusBadgeProps {
  status: Status
  className?: string
}

const statusConfig = {
  draft: {
    label: "임시저장",
    variant: "secondary" as const,
    icon: FileEdit,
  },
  pending: {
    label: "승인대기",
    variant: "outline" as const,
    icon: Clock,
  },
  approved: {
    label: "승인됨",
    variant: "default" as const,
    icon: CheckCircle,
    className: "bg-green-600 hover:bg-green-700",
  },
  rejected: {
    label: "반려됨",
    variant: "destructive" as const,
    icon: XCircle,
  },
  warning: {
    label: "주의",
    variant: "outline" as const,
    icon: AlertTriangle,
    className: "border-yellow-500 text-yellow-600",
  },
}

export function StatusBadge({ status, className }: StatusBadgeProps) {
  const config = statusConfig[status]
  const Icon = config.icon

  return (
    <Badge
      variant={config.variant}
      className={cn(config.className, className)}
    >
      <Icon className="mr-1 h-3 w-3" />
      {config.label}
    </Badge>
  )
}
```

---

## 6. 반응형 디자인

### 6.1 브레이크포인트

```css
/* Tailwind 기본 브레이크포인트 */
sm: 640px   /* 모바일 (landscape) */
md: 768px   /* 태블릿 */
lg: 1024px  /* 데스크톱 */
xl: 1280px  /* 대형 데스크톱 */
2xl: 1536px /* 초대형 모니터 */
```

### 6.2 레이아웃 변화

```tsx
// 반응형 레이아웃 예시
<div className="grid gap-4 grid-cols-1 md:grid-cols-2 lg:grid-cols-4">
  <KPICard />
  <KPICard />
  <KPICard />
  <KPICard />
</div>

// 사이드바 반응형
<SidebarProvider defaultOpen={true}>
  <Sidebar className="hidden md:flex" /> {/* 태블릿 이상에서만 표시 */}
  <main>
    <SidebarTrigger className="md:hidden" /> {/* 모바일에서 토글 버튼 */}
    {children}
  </main>
</SidebarProvider>
```

### 6.3 모바일 최적화

| 화면 | 데스크톱 | 모바일 |
|------|---------|--------|
| 사이드바 | 항상 표시 | 햄버거 메뉴 |
| 데이터 테이블 | 전체 컬럼 | 필수 컬럼만 |
| 폼 레이아웃 | 2~3열 | 1열 |
| 차트 | 가로 배치 | 세로 스택 |

---

## 7. 접근성 (A11y)

### 7.1 WCAG 2.1 준수

| 항목 | 기준 | 구현 방법 |
|------|------|----------|
| **색상 대비** | 4.5:1 이상 | Tailwind 기본 컬러 사용 |
| **키보드 네비게이션** | 전체 기능 | Radix UI 기본 지원 |
| **스크린 리더** | ARIA 라벨 | shadcn/ui 기본 지원 |
| **포커스 표시** | 명확한 아웃라인 | ring 클래스 사용 |

### 7.2 접근성 체크리스트

```tsx
// 접근성 좋은 버튼 예시
<Button aria-label="새 전표 등록">
  <Plus className="h-4 w-4 mr-2" aria-hidden="true" />
  새 전표
</Button>

// 접근성 좋은 폼 예시
<FormField
  control={form.control}
  name="amount"
  render={({ field }) => (
    <FormItem>
      <FormLabel htmlFor="amount">금액</FormLabel>
      <FormControl>
        <Input
          id="amount"
          aria-describedby="amount-description"
          {...field}
        />
      </FormControl>
      <FormDescription id="amount-description">
        원 단위로 입력하세요
      </FormDescription>
      <FormMessage role="alert" />
    </FormItem>
  )}
/>
```

---

## 8. 다크 모드

### 8.1 테마 전환

```tsx
// components/theme-toggle.tsx
import { Moon, Sun } from "lucide-react"
import { useTheme } from "next-themes"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

export function ThemeToggle() {
  const { setTheme } = useTheme()

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon">
          <Sun className="h-[1.2rem] w-[1.2rem] rotate-0 scale-100 transition-all dark:-rotate-90 dark:scale-0" />
          <Moon className="absolute h-[1.2rem] w-[1.2rem] rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100" />
          <span className="sr-only">테마 변경</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onClick={() => setTheme("light")}>
          라이트
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => setTheme("dark")}>
          다크
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => setTheme("system")}>
          시스템
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
```

---

## 9. 프로젝트 구조

### 9.1 디렉토리 구조

```
web/
├── app/                          # Next.js App Router
│   ├── (auth)/                   # 인증 관련 페이지
│   │   ├── login/
│   │   └── register/
│   ├── (dashboard)/              # 대시보드 레이아웃
│   │   ├── layout.tsx
│   │   ├── page.tsx              # 대시보드 메인
│   │   ├── accounting/           # 회계관리
│   │   │   ├── voucher/
│   │   │   ├── ledger/
│   │   │   └── statements/
│   │   ├── tax/                  # 세금관리
│   │   │   ├── invoice/
│   │   │   └── vat/
│   │   ├── hr/                   # 인사급여
│   │   ├── inventory/            # 재고관리
│   │   ├── analytics/            # 경영분석
│   │   └── settings/             # 설정
│   └── globals.css
├── components/
│   ├── ui/                       # shadcn/ui 컴포넌트
│   │   ├── button.tsx
│   │   ├── card.tsx
│   │   ├── data-table.tsx
│   │   └── ...
│   ├── layout/                   # 레이아웃 컴포넌트
│   │   ├── app-sidebar.tsx
│   │   ├── app-header.tsx
│   │   └── app-layout.tsx
│   ├── forms/                    # 폼 컴포넌트
│   │   ├── voucher-form.tsx
│   │   ├── invoice-form.tsx
│   │   └── ...
│   ├── charts/                   # 차트 컴포넌트
│   │   ├── revenue-chart.tsx
│   │   ├── expense-chart.tsx
│   │   └── ...
│   └── shared/                   # 공통 컴포넌트
│       ├── currency-input.tsx
│       ├── account-select.tsx
│       ├── status-badge.tsx
│       └── ...
├── lib/
│   ├── utils.ts                  # 유틸리티 함수
│   ├── api.ts                    # API 클라이언트
│   └── validations.ts            # Zod 스키마
├── hooks/                        # 커스텀 훅
│   ├── use-accounts.ts
│   ├── use-vouchers.ts
│   └── ...
├── types/                        # TypeScript 타입
│   ├── voucher.ts
│   ├── account.ts
│   └── ...
└── styles/
    └── globals.css
```

### 9.2 shadcn/ui 설치 명령어

```bash
# shadcn/ui 초기화
npx shadcn@latest init

# 필수 컴포넌트 설치
npx shadcn@latest add button card input label select checkbox
npx shadcn@latest add form calendar popover dialog sheet
npx shadcn@latest add table dropdown-menu command
npx shadcn@latest add tabs accordion separator
npx shadcn@latest add badge avatar progress skeleton
npx shadcn@latest add toast alert-dialog tooltip
npx shadcn@latest add sidebar chart

# TanStack Table 설치
npm install @tanstack/react-table

# 폼 관련 패키지
npm install react-hook-form @hookform/resolvers zod

# 날짜 처리
npm install date-fns

# 차트
npm install recharts
```

---

## 10. 개발 가이드라인

### 10.1 컴포넌트 작성 규칙

```tsx
// 1. 명확한 Props 타입 정의
interface VoucherCardProps {
  voucher: Voucher
  onEdit?: (id: string) => void
  onDelete?: (id: string) => void
  className?: string
}

// 2. 기본값 설정
export function VoucherCard({
  voucher,
  onEdit,
  onDelete,
  className,
}: VoucherCardProps) {
  // 구현
}

// 3. forwardRef 사용 (필요시)
const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className, ...props }, ref) => {
    return <input ref={ref} className={cn("...", className)} {...props} />
  }
)
Input.displayName = "Input"
```

### 10.2 스타일링 규칙

```tsx
// 1. Tailwind 클래스 순서 (권장)
// Layout → Spacing → Sizing → Typography → Colors → Effects

// Good
<div className="flex items-center gap-4 p-4 w-full text-sm text-gray-600 bg-white rounded-lg shadow-sm">

// 2. cn() 유틸리티 사용
import { cn } from "@/lib/utils"

<Button className={cn(
  "base-classes",
  isActive && "active-classes",
  className
)}>

// 3. 조건부 스타일링
<div className={cn(
  "rounded-md p-4",
  status === "success" && "bg-green-50 border-green-200",
  status === "error" && "bg-red-50 border-red-200",
)}>
```

---

## 부록

### A. 아이콘 가이드 (Lucide)

| 용도 | 아이콘 | 코드 |
|------|--------|------|
| 대시보드 | LayoutDashboard | `<LayoutDashboard />` |
| 회계 | Calculator | `<Calculator />` |
| 세금 | Receipt | `<Receipt />` |
| 인사 | Users | `<Users />` |
| 재고 | Package | `<Package />` |
| 설정 | Settings | `<Settings />` |
| 추가 | Plus | `<Plus />` |
| 삭제 | Trash2 | `<Trash2 />` |
| 수정 | Pencil | `<Pencil />` |
| 검색 | Search | `<Search />` |
| 다운로드 | Download | `<Download />` |
| 업로드 | Upload | `<Upload />` |

### B. 키보드 단축키

| 단축키 | 동작 |
|--------|------|
| `Ctrl + S` | 저장 |
| `Ctrl + N` | 새 항목 |
| `Ctrl + F` | 검색 |
| `Escape` | 모달 닫기 |
| `Tab` | 다음 필드 |
| `Shift + Tab` | 이전 필드 |
| `Enter` | 제출/확인 |

---

*본 문서는 K-ERP UI/UX 설계 초안이며, 개발 진행에 따라 업데이트됩니다.*
