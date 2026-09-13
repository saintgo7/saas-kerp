import { useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { useForm, useFieldArray, Controller } from "react-hook-form";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
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
import { accountsApi, vouchersApi } from "@/api";
import { getErrorMessage } from "@/services/api";
import type { VoucherType } from "@/hooks/useVoucher";

/**
 * Voucher types accepted by the backend
 * (internal/dto/voucher_dto.go: oneof=general sales purchase payment receipt
 * adjustment closing). `voucher_type` is required on create; omitting it made
 * every POST /vouchers fail with 400.
 */
const VOUCHER_TYPES: { value: VoucherType; label: string }[] = [
  { value: "general", label: "일반전표" },
  { value: "sales", label: "매출전표" },
  { value: "purchase", label: "매입전표" },
  { value: "payment", label: "지급전표" },
  { value: "receipt", label: "입금전표" },
  { value: "adjustment", label: "결산조정" },
  { value: "closing", label: "마감전표" },
];

/**
 * Won has no sub-unit, so amounts are whole numbers, but a debit/credit sum can
 * still pick up float error. Compare with a tolerance instead of `===`; the
 * backend does the same.
 */
const BALANCE_EPSILON = 0.005;

function isBalanced(totalDebit: number, totalCredit: number): boolean {
  return (
    Math.abs(totalDebit - totalCredit) < BALANCE_EPSILON && totalDebit > 0
  );
}

/**
 * An empty number input yields NaN under `valueAsNumber`, which zod rejects at
 * the type layer with an English message and skips the balance refine entirely.
 * Coerce blank input to 0 instead.
 */
function toAmount(value: unknown): number {
  if (value === "" || value === null || value === undefined) return 0;
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : 0;
}

const amountSchema = z
  .number({ message: "숫자를 입력하세요" })
  .min(0, "0 이상이어야 합니다");

const voucherEntrySchema = z.object({
  accountId: z.string().min(1, "계정과목을 선택하세요"),
  debitAmount: amountSchema,
  creditAmount: amountSchema,
  description: z.string().optional(),
});

const voucherSchema = z
  .object({
    voucherDate: z.string().min(1, "전표일자를 입력하세요"),
    voucherType: z.string().min(1, "전표유형을 선택하세요"),
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
      return isBalanced(totalDebit, totalCredit);
    },
    {
      message: "차변과 대변의 합계가 일치해야 합니다",
      path: ["entries"],
    }
  );

type VoucherFormData = z.infer<typeof voucherSchema>;

export function VoucherFormPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  // GET /api/v1/accounts — the picker used to be a hardcoded list of 23 codes
  // that had nothing to do with the company's actual chart of accounts.
  const {
    data: accountsResponse,
    isLoading: isLoadingAccounts,
    isError: isAccountsError,
    error: accountsError,
  } = useQuery({
    queryKey: ["accounts", "list", { pageSize: 100, isActive: true }],
    queryFn: () => accountsApi.list({ pageSize: 100, isActive: true }),
  });

  const accountOptions = useMemo(
    () =>
      (accountsResponse?.data.items ?? [])
        // Only leaf accounts may be posted to.
        .filter((account) => account.allowDirectPosting !== false)
        .map((account) => ({
          value: account.id,
          label: `${account.code} ${account.name}`,
        })),
    [accountsResponse]
  );

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
      voucherType: "general",
      description: "",
      entries: [
        { accountId: "", debitAmount: 0, creditAmount: 0, description: "" },
        { accountId: "", debitAmount: 0, creditAmount: 0, description: "" },
      ],
    },
  });

  const { fields, append, remove } = useFieldArray({
    control,
    name: "entries",
  });

  const entries = watch("entries");
  const totalDebit =
    entries?.reduce((sum, e) => sum + toAmount(e.debitAmount), 0) || 0;
  const totalCredit =
    entries?.reduce((sum, e) => sum + toAmount(e.creditAmount), 0) || 0;
  const balanced = isBalanced(totalDebit, totalCredit);

  // POST /api/v1/vouchers
  const createMutation = useMutation({
    mutationFn: (data: VoucherFormData) =>
      vouchersApi.create({
        voucherDate: data.voucherDate,
        voucherType: data.voucherType as VoucherType,
        description: data.description,
        entries: data.entries.map((entry) => ({
          accountId: entry.accountId,
          debitAmount: toAmount(entry.debitAmount),
          creditAmount: toAmount(entry.creditAmount),
          description: entry.description,
        })),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["vouchers"] });
      toast.success("전표 저장 완료", "전표가 성공적으로 저장되었습니다.");
      navigate("/accounting/voucher");
    },
    onError: (err: unknown) => {
      // Only the server can tell us the save succeeded. Previously this screen
      // logged the form and claimed success without any request at all.
      toast.error(
        "저장 실패",
        getErrorMessage(err, "전표 저장 중 오류가 발생했습니다.")
      );
    },
  });

  const onSubmit = (data: VoucherFormData) => createMutation.mutate(data);
  const isSubmitting = createMutation.isPending;

  const addEntry = () => {
    append({ accountId: "", debitAmount: 0, creditAmount: 0, description: "" });
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
              <Controller
                name="voucherType"
                control={control}
                render={({ field }) => (
                  <Select
                    label="전표유형"
                    required
                    options={VOUCHER_TYPES}
                    error={errors.voucherType?.message}
                    {...field}
                  />
                )}
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

            {/* The chart of accounts must load before entries can be posted. */}
            {isAccountsError && (
              <div className="flex items-center space-x-2 p-3 mb-4 bg-destructive/10 text-destructive rounded-lg">
                <AlertCircle className="h-4 w-4" />
                <span className="text-sm">
                  {getErrorMessage(
                    accountsError,
                    "계정과목을 불러오지 못했습니다."
                  )}
                </span>
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
                      name={`entries.${index}.accountId`}
                      control={control}
                      render={({ field }) => (
                        <Select
                          options={accountOptions}
                          placeholder={
                            isLoadingAccounts ? "불러오는 중..." : "계정선택"
                          }
                          error={errors.entries?.[index]?.accountId?.message}
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
                      {...register(`entries.${index}.debitAmount`, {
                        setValueAs: toAmount,
                      })}
                    />
                  </div>
                  <div className="col-span-2">
                    <Input
                      type="number"
                      min={0}
                      placeholder="0"
                      className="text-right font-mono"
                      error={errors.entries?.[index]?.creditAmount?.message}
                      {...register(`entries.${index}.creditAmount`, {
                        setValueAs: toAmount,
                      })}
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
                  variant={balanced ? "success" : "destructive"}
                  className="ml-2"
                >
                  {balanced ? "균형" : "불균형"}
                </Badge>
                {!balanced && totalDebit > 0 && (
                  <span className="text-sm text-destructive">
                    차이:{" "}
                    {formatCurrency(Math.abs(totalDebit - totalCredit), {
                      showSymbol: false,
                    })}
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
