import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useForm, useFieldArray, Controller } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  Plus,
  Trash2,
  Save,
  ArrowLeft,
  Send,
  Calculator,
  Building2,
  Search,
} from "lucide-react";
import {
  Button,
  Input,
  Textarea,
  Select,
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  Badge,
  Modal,
} from "@/components/ui";
import { formatCurrency, cn } from "@/lib/utils";
import { toast } from "@/stores/ui";

// Invoice type options
const invoiceTypeOptions = [
  { value: "01", label: "01 일반" },
  { value: "02", label: "02 영세율" },
  { value: "03", label: "03 위수탁" },
  { value: "04", label: "04 수입" },
];

// Tax type options
const taxTypeOptions = [
  { value: "taxable", label: "과세" },
  { value: "zero", label: "영세율" },
  { value: "exempt", label: "면세" },
];

// Item schema
const invoiceItemSchema = z.object({
  itemDate: z.string().min(1, "일자를 입력하세요"),
  itemName: z.string().min(1, "품목명을 입력하세요"),
  specification: z.string().optional(),
  quantity: z.number().min(1, "수량을 입력하세요"),
  unitPrice: z.number().min(0, "단가를 입력하세요"),
  supplyAmount: z.number().min(0),
  taxAmount: z.number().min(0),
  note: z.string().optional(),
});

// Main invoice schema
const invoiceSchema = z.object({
  // Supplier info
  supplierBizNo: z.string().min(10, "사업자번호 10자리를 입력하세요"),
  supplierName: z.string().min(1, "상호를 입력하세요"),
  supplierCeoName: z.string().min(1, "대표자명을 입력하세요"),
  supplierAddress: z.string().min(1, "사업장 주소를 입력하세요"),
  supplierBizType: z.string().optional(),
  supplierBizItem: z.string().optional(),
  supplierEmail: z.string().email("올바른 이메일을 입력하세요"),

  // Buyer info
  buyerBizNo: z.string().min(10, "사업자번호 10자리를 입력하세요"),
  buyerName: z.string().min(1, "상호를 입력하세요"),
  buyerCeoName: z.string().min(1, "대표자명을 입력하세요"),
  buyerAddress: z.string().optional(),
  buyerBizType: z.string().optional(),
  buyerBizItem: z.string().optional(),
  buyerEmail: z.string().email("올바른 이메일을 입력하세요"),
  buyerEmail2: z.string().email().optional().or(z.literal("")),

  // Invoice details
  invoiceType: z.string().min(1, "종류를 선택하세요"),
  taxType: z.string().min(1, "과세유형을 선택하세요"),
  issueDate: z.string().min(1, "작성일자를 입력하세요"),
  note: z.string().optional(),

  // Items
  items: z.array(invoiceItemSchema).min(1, "최소 1개 이상의 품목이 필요합니다"),
});

type InvoiceFormData = z.infer<typeof invoiceSchema>;

// Mock company data for lookup
const mockCompanies = [
  {
    bizNo: "1234567890",
    name: "(주)테스트회사",
    ceoName: "홍길동",
    address: "서울시 강남구 테헤란로 123",
    bizType: "제조업",
    bizItem: "전자제품",
  },
  {
    bizNo: "9876543210",
    name: "(주)샘플기업",
    ceoName: "김철수",
    address: "서울시 서초구 서초대로 456",
    bizType: "도소매업",
    bizItem: "의류",
  },
];

export function InvoiceIssuePage() {
  const navigate = useNavigate();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [showBuyerSearch, setShowBuyerSearch] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");

  const {
    register,
    control,
    handleSubmit,
    watch,
    setValue,
    formState: { errors },
  } = useForm<InvoiceFormData>({
    resolver: zodResolver(invoiceSchema),
    defaultValues: {
      // Default supplier (current company)
      supplierBizNo: "1111111111",
      supplierName: "우리회사(주)",
      supplierCeoName: "대표자",
      supplierAddress: "서울시 강남구",
      supplierBizType: "서비스업",
      supplierBizItem: "소프트웨어",
      supplierEmail: "invoice@company.com",

      // Empty buyer
      buyerBizNo: "",
      buyerName: "",
      buyerCeoName: "",
      buyerAddress: "",
      buyerBizType: "",
      buyerBizItem: "",
      buyerEmail: "",
      buyerEmail2: "",

      // Invoice defaults
      invoiceType: "01",
      taxType: "taxable",
      issueDate: new Date().toISOString().split("T")[0],
      note: "",

      // Default item
      items: [
        {
          itemDate: new Date().toISOString().split("T")[0],
          itemName: "",
          specification: "",
          quantity: 1,
          unitPrice: 0,
          supplyAmount: 0,
          taxAmount: 0,
          note: "",
        },
      ],
    },
  });

  const { fields, append, remove } = useFieldArray({
    control,
    name: "items",
  });

  const items = watch("items");
  const taxType = watch("taxType");

  // Calculate totals
  const totalSupply = items?.reduce(
    (sum, item) => sum + (Number(item.supplyAmount) || 0),
    0
  ) || 0;
  const totalTax = items?.reduce(
    (sum, item) => sum + (Number(item.taxAmount) || 0),
    0
  ) || 0;
  const totalAmount = totalSupply + totalTax;

  // Calculate item amounts when quantity or unit price changes
  const calculateItemAmounts = (index: number) => {
    const quantity = Number(items[index]?.quantity) || 0;
    const unitPrice = Number(items[index]?.unitPrice) || 0;
    const supplyAmount = quantity * unitPrice;
    const taxAmount = taxType === "taxable" ? Math.round(supplyAmount * 0.1) : 0;

    setValue(`items.${index}.supplyAmount`, supplyAmount);
    setValue(`items.${index}.taxAmount`, taxAmount);
  };

  const onSubmit = async (data: InvoiceFormData) => {
    setIsSubmitting(true);
    try {
      console.log("Invoice data:", data);
      toast.success("세금계산서 저장 완료", "세금계산서가 저장되었습니다.");
      navigate("/invoice/list");
    } catch {
      toast.error("저장 실패", "세금계산서 저장 중 오류가 발생했습니다.");
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleIssue = async () => {
    setIsSubmitting(true);
    try {
      toast.success("세금계산서 발행 완료", "세금계산서가 국세청에 전송되었습니다.");
      navigate("/invoice/list");
    } catch {
      toast.error("발행 실패", "세금계산서 발행 중 오류가 발생했습니다.");
    } finally {
      setIsSubmitting(false);
    }
  };

  const addItem = () => {
    append({
      itemDate: new Date().toISOString().split("T")[0],
      itemName: "",
      specification: "",
      quantity: 1,
      unitPrice: 0,
      supplyAmount: 0,
      taxAmount: 0,
      note: "",
    });
  };

  const selectBuyer = (company: (typeof mockCompanies)[0]) => {
    setValue("buyerBizNo", company.bizNo);
    setValue("buyerName", company.name);
    setValue("buyerCeoName", company.ceoName);
    setValue("buyerAddress", company.address);
    setValue("buyerBizType", company.bizType);
    setValue("buyerBizItem", company.bizItem);
    setShowBuyerSearch(false);
    setSearchQuery("");
  };

  const filteredCompanies = mockCompanies.filter(
    (c) =>
      c.name.includes(searchQuery) ||
      c.bizNo.includes(searchQuery) ||
      c.ceoName.includes(searchQuery)
  );

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <Button variant="ghost" size="icon" onClick={() => navigate(-1)}>
            <ArrowLeft className="h-5 w-5" />
          </Button>
          <div>
            <h1 className="text-2xl font-bold">세금계산서 발행</h1>
            <p className="text-muted-foreground">
              전자세금계산서를 작성하고 발행합니다.
            </p>
          </div>
        </div>
        <div className="flex items-center space-x-2">
          <Button variant="outline" onClick={() => navigate(-1)}>
            취소
          </Button>
          <Button variant="outline" onClick={handleSubmit(onSubmit)} isLoading={isSubmitting}>
            <Save className="h-4 w-4 mr-2" />
            임시저장
          </Button>
          <Button onClick={handleSubmit(handleIssue)} isLoading={isSubmitting}>
            <Send className="h-4 w-4 mr-2" />
            발행
          </Button>
        </div>
      </div>

      <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
        {/* Invoice Type & Date */}
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">발행 정보</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
              <Controller
                name="invoiceType"
                control={control}
                render={({ field }) => (
                  <Select
                    label="세금계산서 종류"
                    options={invoiceTypeOptions}
                    required
                    error={errors.invoiceType?.message}
                    {...field}
                  />
                )}
              />
              <Controller
                name="taxType"
                control={control}
                render={({ field }) => (
                  <Select
                    label="과세유형"
                    options={taxTypeOptions}
                    required
                    error={errors.taxType?.message}
                    {...field}
                  />
                )}
              />
              <Input
                type="date"
                label="작성일자"
                required
                error={errors.issueDate?.message}
                {...register("issueDate")}
              />
              <div className="flex flex-col justify-end">
                <Badge
                  variant={taxType === "taxable" ? "default" : "outline"}
                  className="self-start"
                >
                  {taxType === "taxable"
                    ? "부가세 10%"
                    : taxType === "zero"
                    ? "영세율 0%"
                    : "면세"}
                </Badge>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Supplier & Buyer */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Supplier */}
          <Card>
            <CardHeader className="flex flex-row items-center justify-between">
              <CardTitle className="text-lg flex items-center">
                <Building2 className="h-5 w-5 mr-2" />
                공급자 (우리회사)
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <Input
                  label="사업자번호"
                  required
                  error={errors.supplierBizNo?.message}
                  {...register("supplierBizNo")}
                />
                <Input
                  label="상호"
                  required
                  error={errors.supplierName?.message}
                  {...register("supplierName")}
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <Input
                  label="대표자"
                  required
                  error={errors.supplierCeoName?.message}
                  {...register("supplierCeoName")}
                />
                <Input
                  label="이메일"
                  type="email"
                  required
                  error={errors.supplierEmail?.message}
                  {...register("supplierEmail")}
                />
              </div>
              <Input
                label="사업장 주소"
                required
                error={errors.supplierAddress?.message}
                {...register("supplierAddress")}
              />
              <div className="grid grid-cols-2 gap-4">
                <Input label="업태" {...register("supplierBizType")} />
                <Input label="종목" {...register("supplierBizItem")} />
              </div>
            </CardContent>
          </Card>

          {/* Buyer */}
          <Card>
            <CardHeader className="flex flex-row items-center justify-between">
              <CardTitle className="text-lg flex items-center">
                <Building2 className="h-5 w-5 mr-2" />
                공급받는자 (거래처)
              </CardTitle>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => setShowBuyerSearch(true)}
              >
                <Search className="h-4 w-4 mr-2" />
                거래처 검색
              </Button>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <Input
                  label="사업자번호"
                  required
                  error={errors.buyerBizNo?.message}
                  {...register("buyerBizNo")}
                />
                <Input
                  label="상호"
                  required
                  error={errors.buyerName?.message}
                  {...register("buyerName")}
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <Input
                  label="대표자"
                  required
                  error={errors.buyerCeoName?.message}
                  {...register("buyerCeoName")}
                />
                <Input
                  label="이메일 (1)"
                  type="email"
                  required
                  error={errors.buyerEmail?.message}
                  {...register("buyerEmail")}
                />
              </div>
              <Input label="사업장 주소" {...register("buyerAddress")} />
              <div className="grid grid-cols-2 gap-4">
                <Input label="업태" {...register("buyerBizType")} />
                <Input label="종목" {...register("buyerBizItem")} />
              </div>
              <Input
                label="이메일 (2) - 참조"
                type="email"
                {...register("buyerEmail2")}
              />
            </CardContent>
          </Card>
        </div>

        {/* Items */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-lg flex items-center">
              <Calculator className="h-5 w-5 mr-2" />
              품목 내역
            </CardTitle>
            <Button type="button" variant="outline" size="sm" onClick={addItem}>
              <Plus className="h-4 w-4 mr-2" />
              품목 추가
            </Button>
          </CardHeader>
          <CardContent>
            {/* Table Header */}
            <div className="grid grid-cols-12 gap-2 px-2 py-2 bg-muted rounded-t-lg font-medium text-sm">
              <div className="col-span-1">일자</div>
              <div className="col-span-2">품목명</div>
              <div className="col-span-1">규격</div>
              <div className="col-span-1 text-right">수량</div>
              <div className="col-span-2 text-right">단가</div>
              <div className="col-span-2 text-right">공급가액</div>
              <div className="col-span-2 text-right">세액</div>
              <div className="col-span-1"></div>
            </div>

            {/* Item Rows */}
            <div className="divide-y">
              {fields.map((field, index) => (
                <div
                  key={field.id}
                  className="grid grid-cols-12 gap-2 px-2 py-3 items-start"
                >
                  <div className="col-span-1">
                    <Input
                      type="date"
                      className="text-xs"
                      {...register(`items.${index}.itemDate`)}
                    />
                  </div>
                  <div className="col-span-2">
                    <Input
                      placeholder="품목명"
                      error={errors.items?.[index]?.itemName?.message}
                      {...register(`items.${index}.itemName`)}
                    />
                  </div>
                  <div className="col-span-1">
                    <Input
                      placeholder="규격"
                      {...register(`items.${index}.specification`)}
                    />
                  </div>
                  <div className="col-span-1">
                    <Input
                      type="number"
                      min={1}
                      className="text-right font-mono"
                      {...register(`items.${index}.quantity`, {
                        valueAsNumber: true,
                        onChange: () => calculateItemAmounts(index),
                      })}
                    />
                  </div>
                  <div className="col-span-2">
                    <Input
                      type="number"
                      min={0}
                      className="text-right font-mono"
                      {...register(`items.${index}.unitPrice`, {
                        valueAsNumber: true,
                        onChange: () => calculateItemAmounts(index),
                      })}
                    />
                  </div>
                  <div className="col-span-2">
                    <Input
                      type="number"
                      readOnly
                      className="text-right font-mono bg-muted"
                      {...register(`items.${index}.supplyAmount`, {
                        valueAsNumber: true,
                      })}
                    />
                  </div>
                  <div className="col-span-2">
                    <Input
                      type="number"
                      readOnly
                      className="text-right font-mono bg-muted"
                      {...register(`items.${index}.taxAmount`, {
                        valueAsNumber: true,
                      })}
                    />
                  </div>
                  <div className="col-span-1 flex justify-center">
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="text-destructive hover:text-destructive"
                      onClick={() => remove(index)}
                      disabled={fields.length <= 1}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              ))}
            </div>

            {/* Totals */}
            <div className="grid grid-cols-12 gap-2 px-2 py-4 bg-muted/50 rounded-b-lg font-semibold border-t-2">
              <div className="col-span-6 text-right">합계</div>
              <div className="col-span-2 text-right font-mono">
                {formatCurrency(totalSupply, { showSymbol: false })}
              </div>
              <div className="col-span-2 text-right font-mono">
                {formatCurrency(totalTax, { showSymbol: false })}
              </div>
              <div className="col-span-2"></div>
            </div>

            {/* Grand Total */}
            <div className="mt-4 p-4 bg-primary/5 rounded-lg">
              <div className="flex justify-between items-center">
                <span className="text-lg font-medium">총 합계금액</span>
                <span className="text-2xl font-bold text-primary">
                  {formatCurrency(totalAmount)}
                </span>
              </div>
              <div className="flex justify-end space-x-6 mt-2 text-sm text-muted-foreground">
                <span>공급가액: {formatCurrency(totalSupply)}</span>
                <span>세액: {formatCurrency(totalTax)}</span>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Note */}
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">비고</CardTitle>
          </CardHeader>
          <CardContent>
            <Textarea
              placeholder="세금계산서에 표시할 비고 내용을 입력하세요"
              {...register("note")}
            />
          </CardContent>
        </Card>

        {/* Submit Buttons (Mobile) */}
        <div className="flex justify-end space-x-2 lg:hidden">
          <Button variant="outline" onClick={() => navigate(-1)}>
            취소
          </Button>
          <Button variant="outline" type="submit" isLoading={isSubmitting}>
            <Save className="h-4 w-4 mr-2" />
            임시저장
          </Button>
          <Button onClick={handleSubmit(handleIssue)} isLoading={isSubmitting}>
            <Send className="h-4 w-4 mr-2" />
            발행
          </Button>
        </div>
      </form>

      {/* Buyer Search Modal */}
      <Modal
        isOpen={showBuyerSearch}
        onClose={() => setShowBuyerSearch(false)}
        title="거래처 검색"
      >
        <div className="space-y-4">
          <Input
            placeholder="상호명, 사업자번호, 대표자명으로 검색"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
          <div className="max-h-80 overflow-y-auto space-y-2">
            {filteredCompanies.length === 0 ? (
              <p className="text-center text-muted-foreground py-8">
                검색 결과가 없습니다.
              </p>
            ) : (
              filteredCompanies.map((company) => (
                <div
                  key={company.bizNo}
                  className={cn(
                    "p-3 border rounded-lg cursor-pointer hover:bg-accent transition-colors"
                  )}
                  onClick={() => selectBuyer(company)}
                >
                  <div className="flex justify-between items-start">
                    <div>
                      <p className="font-medium">{company.name}</p>
                      <p className="text-sm text-muted-foreground">
                        {company.bizNo} | {company.ceoName}
                      </p>
                    </div>
                    <Badge variant="outline">{company.bizType}</Badge>
                  </div>
                  <p className="text-sm text-muted-foreground mt-1">
                    {company.address}
                  </p>
                </div>
              ))
            )}
          </div>
        </div>
      </Modal>
    </div>
  );
}
