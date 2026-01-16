import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useForm, useFieldArray, Controller } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Plus, Trash2, Save, ArrowLeft, AlertCircle } from "lucide-react";
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
} from "@/components/ui";
import { formatCurrency } from "@/lib/utils";
import { toast } from "@/stores/ui";

// Mock account options
const accountOptions = [
  { value: "101", label: "101 현금" },
  { value: "102", label: "102 보통예금" },
  { value: "103", label: "103 당좌예금" },
  { value: "108", label: "108 받을어음" },
  { value: "109", label: "109 외상매출금" },
  { value: "141", label: "141 상품" },
  { value: "142", label: "142 제품" },
  { value: "201", label: "201 지급어음" },
  { value: "202", label: "202 외상매입금" },
  { value: "253", label: "253 미지급금" },
  { value: "254", label: "254 예수금" },
  { value: "255", label: "255 부가세예수금" },
  { value: "401", label: "401 상품매출" },
  { value: "402", label: "402 제품매출" },
  { value: "501", label: "501 상품매입" },
  { value: "502", label: "502 원재료매입" },
  { value: "801", label: "801 급여" },
  { value: "802", label: "802 복리후생비" },
  { value: "803", label: "803 여비교통비" },
  { value: "804", label: "804 접대비" },
  { value: "805", label: "805 통신비" },
  { value: "806", label: "806 소모품비" },
  { value: "810", label: "810 지급임차료" },
];

// Validation schema
const voucherEntrySchema = z.object({
  accountCode: z.string().min(1, "계정과목을 선택하세요"),
  debitAmount: z.number().min(0, "0 이상이어야 합니다"),
  creditAmount: z.number().min(0, "0 이상이어야 합니다"),
  description: z.string().optional(),
});

const voucherSchema = z
  .object({
    voucherDate: z.string().min(1, "전표일자를 입력하세요"),
    description: z.string().min(1, "적요를 입력하세요"),
    entries: z
      .array(voucherEntrySchema)
      .min(2, "최소 2개 이상의 분개가 필요합니다"),
  })
  .refine(
    (data) => {
      const totalDebit = data.entries.reduce((sum, e) => sum + e.debitAmount, 0);
      const totalCredit = data.entries.reduce(
        (sum, e) => sum + e.creditAmount,
        0
      );
      return totalDebit === totalCredit && totalDebit > 0;
    },
    {
      message: "차변과 대변의 합계가 일치해야 합니다",
      path: ["entries"],
    }
  );

type VoucherFormData = z.infer<typeof voucherSchema>;

export function VoucherFormPage() {
  const navigate = useNavigate();
  const [isSubmitting, setIsSubmitting] = useState(false);

  const {
    register,
    control,
    handleSubmit,
    watch,
    formState: { errors },
  } = useForm<VoucherFormData>({
    resolver: zodResolver(voucherSchema),
    defaultValues: {
      voucherDate: new Date().toISOString().split("T")[0],
      description: "",
      entries: [
        { accountCode: "", debitAmount: 0, creditAmount: 0, description: "" },
        { accountCode: "", debitAmount: 0, creditAmount: 0, description: "" },
      ],
    },
  });

  const { fields, append, remove } = useFieldArray({
    control,
    name: "entries",
  });

  const entries = watch("entries");
  const totalDebit = entries?.reduce((sum, e) => sum + (Number(e.debitAmount) || 0), 0) || 0;
  const totalCredit = entries?.reduce((sum, e) => sum + (Number(e.creditAmount) || 0), 0) || 0;
  const isBalanced = totalDebit === totalCredit && totalDebit > 0;

  const onSubmit = async (data: VoucherFormData) => {
    setIsSubmitting(true);
    try {
      // TODO: API call
      console.log("Voucher data:", data);
      toast.success("전표 저장 완료", "전표가 성공적으로 저장되었습니다.");
      navigate("/accounting/voucher");
    } catch {
      toast.error("저장 실패", "전표 저장 중 오류가 발생했습니다.");
    } finally {
      setIsSubmitting(false);
    }
  };

  const addEntry = () => {
    append({ accountCode: "", debitAmount: 0, creditAmount: 0, description: "" });
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <Button variant="ghost" size="icon" onClick={() => navigate(-1)}>
            <ArrowLeft className="h-5 w-5" />
          </Button>
          <div>
            <h1 className="text-2xl font-bold">전표 작성</h1>
            <p className="text-muted-foreground">새로운 회계 전표를 작성합니다.</p>
          </div>
        </div>
        <div className="flex items-center space-x-2">
          <Button variant="outline" onClick={() => navigate(-1)}>
            취소
          </Button>
          <Button onClick={handleSubmit(onSubmit)} isLoading={isSubmitting}>
            <Save className="h-4 w-4 mr-2" />
            저장
          </Button>
        </div>
      </div>

      <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
        {/* Basic Info */}
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">기본 정보</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <Input
                type="date"
                label="전표일자"
                required
                error={errors.voucherDate?.message}
                {...register("voucherDate")}
              />
              <div className="md:col-span-2">
                <Textarea
                  label="적요"
                  placeholder="전표에 대한 설명을 입력하세요"
                  required
                  error={errors.description?.message}
                  {...register("description")}
                />
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Entries */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-lg">분개 입력</CardTitle>
            <Button type="button" variant="outline" size="sm" onClick={addEntry}>
              <Plus className="h-4 w-4 mr-2" />
              분개 추가
            </Button>
          </CardHeader>
          <CardContent>
            {/* Error message */}
            {errors.entries?.root && (
              <div className="flex items-center space-x-2 p-3 mb-4 bg-destructive/10 text-destructive rounded-lg">
                <AlertCircle className="h-4 w-4" />
                <span className="text-sm">{errors.entries.root.message}</span>
              </div>
            )}

            {/* Table Header */}
            <div className="grid grid-cols-12 gap-2 px-2 py-2 bg-muted rounded-t-lg font-medium text-sm">
              <div className="col-span-3">계정과목</div>
              <div className="col-span-2 text-right">차변</div>
              <div className="col-span-2 text-right">대변</div>
              <div className="col-span-4">적요</div>
              <div className="col-span-1"></div>
            </div>

            {/* Entry Rows */}
            <div className="divide-y">
              {fields.map((field, index) => (
                <div
                  key={field.id}
                  className="grid grid-cols-12 gap-2 px-2 py-3 items-start"
                >
                  <div className="col-span-3">
                    <Controller
                      name={`entries.${index}.accountCode`}
                      control={control}
                      render={({ field }) => (
                        <Select
                          options={accountOptions}
                          placeholder="계정선택"
                          error={errors.entries?.[index]?.accountCode?.message}
                          {...field}
                        />
                      )}
                    />
                  </div>
                  <div className="col-span-2">
                    <Input
                      type="number"
                      min={0}
                      placeholder="0"
                      className="text-right font-mono"
                      error={errors.entries?.[index]?.debitAmount?.message}
                      {...register(`entries.${index}.debitAmount`, { valueAsNumber: true })}
                    />
                  </div>
                  <div className="col-span-2">
                    <Input
                      type="number"
                      min={0}
                      placeholder="0"
                      className="text-right font-mono"
                      error={errors.entries?.[index]?.creditAmount?.message}
                      {...register(`entries.${index}.creditAmount`, { valueAsNumber: true })}
                    />
                  </div>
                  <div className="col-span-4">
                    <Input
                      placeholder="분개 적요"
                      {...register(`entries.${index}.description`)}
                    />
                  </div>
                  <div className="col-span-1 flex justify-center">
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="text-destructive hover:text-destructive"
                      onClick={() => remove(index)}
                      disabled={fields.length <= 2}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              ))}
            </div>

            {/* Totals */}
            <div className="grid grid-cols-12 gap-2 px-2 py-3 bg-muted/50 rounded-b-lg font-semibold">
              <div className="col-span-3 text-right">합계</div>
              <div className="col-span-2 text-right font-mono">
                {formatCurrency(totalDebit, { showSymbol: false })}
              </div>
              <div className="col-span-2 text-right font-mono">
                {formatCurrency(totalCredit, { showSymbol: false })}
              </div>
              <div className="col-span-5 flex items-center space-x-2">
                <Badge
                  variant={isBalanced ? "success" : "destructive"}
                  className="ml-2"
                >
                  {isBalanced ? "균형" : "불균형"}
                </Badge>
                {!isBalanced && totalDebit > 0 && (
                  <span className="text-sm text-destructive">
                    차이: {formatCurrency(Math.abs(totalDebit - totalCredit), { showSymbol: false })}
                  </span>
                )}
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Submit Button (Mobile) */}
        <div className="flex justify-end space-x-2 lg:hidden">
          <Button variant="outline" onClick={() => navigate(-1)}>
            취소
          </Button>
          <Button type="submit" isLoading={isSubmitting}>
            <Save className="h-4 w-4 mr-2" />
            저장
          </Button>
        </div>
      </form>
    </div>
  );
}
