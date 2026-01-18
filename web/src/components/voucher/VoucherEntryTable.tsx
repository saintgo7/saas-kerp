import { useCallback } from "react";
import { useFieldArray, Control, UseFormRegister, FieldErrors } from "react-hook-form";
import { Plus, Trash2, AlertCircle } from "lucide-react";
import { Button, Input, Badge } from "@/components/ui";
import { AccountSelect } from "./AccountSelect";
import { CurrencyInput } from "../common/CurrencyInput";
import { formatCurrency } from "@/lib/utils";

interface VoucherEntry {
  accountId: string;
  accountCode?: string;
  accountName?: string;
  debitAmount: number;
  creditAmount: number;
  description?: string;
}

interface VoucherFormData {
  voucherDate: string;
  description: string;
  entries: VoucherEntry[];
}

interface VoucherEntryTableProps {
  control: Control<VoucherFormData>;
  register: UseFormRegister<VoucherFormData>;
  errors: FieldErrors<VoucherFormData>;
  entries: VoucherEntry[];
  disabled?: boolean;
}

export function VoucherEntryTable({
  control,
  register,
  errors,
  entries,
  disabled = false,
}: VoucherEntryTableProps) {
  const { fields, append, remove } = useFieldArray({
    control,
    name: "entries",
  });

  // Calculate totals
  const totalDebit = entries?.reduce((sum, e) => sum + (Number(e.debitAmount) || 0), 0) || 0;
  const totalCredit = entries?.reduce((sum, e) => sum + (Number(e.creditAmount) || 0), 0) || 0;
  const isBalanced = totalDebit === totalCredit && totalDebit > 0;

  const addEntry = useCallback(() => {
    append({ accountId: "", debitAmount: 0, creditAmount: 0, description: "" });
  }, [append]);

  const handleAccountSelect = useCallback(
    (index: number, account: { id: string; code: string; name: string } | null) => {
      if (account) {
        // Account selection is handled by the AccountSelect component internally
        // This callback is for any additional logic needed
      }
    },
    []
  );

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold">분개 입력</h3>
        {!disabled && (
          <Button type="button" variant="outline" size="sm" onClick={addEntry}>
            <Plus className="h-4 w-4 mr-2" />
            분개 추가
          </Button>
        )}
      </div>

      {/* Error message */}
      {errors.entries?.root && (
        <div className="flex items-center space-x-2 p-3 bg-destructive/10 text-destructive rounded-lg">
          <AlertCircle className="h-4 w-4 flex-shrink-0" />
          <span className="text-sm">{errors.entries.root.message}</span>
        </div>
      )}

      {/* Table */}
      <div className="border rounded-lg overflow-hidden">
        {/* Table Header */}
        <div className="grid grid-cols-12 gap-2 px-4 py-3 bg-muted font-medium text-sm">
          <div className="col-span-1 text-center">No.</div>
          <div className="col-span-3">계정과목</div>
          <div className="col-span-2 text-right">차변</div>
          <div className="col-span-2 text-right">대변</div>
          <div className="col-span-3">적요</div>
          <div className="col-span-1 text-center">삭제</div>
        </div>

        {/* Entry Rows */}
        <div className="divide-y">
          {fields.map((field, index) => (
            <div
              key={field.id}
              className="grid grid-cols-12 gap-2 px-4 py-3 items-start hover:bg-muted/30 transition-colors"
            >
              {/* Row Number */}
              <div className="col-span-1 flex items-center justify-center pt-2">
                <span className="text-muted-foreground text-sm">{index + 1}</span>
              </div>

              {/* Account Select */}
              <div className="col-span-3">
                <AccountSelect
                  name={`entries.${index}.accountId`}
                  control={control}
                  error={errors.entries?.[index]?.accountId?.message}
                  disabled={disabled}
                  onSelect={(account) => handleAccountSelect(index, account)}
                />
              </div>

              {/* Debit Amount */}
              <div className="col-span-2">
                <CurrencyInput
                  {...register(`entries.${index}.debitAmount`, { valueAsNumber: true })}
                  error={errors.entries?.[index]?.debitAmount?.message}
                  disabled={disabled}
                  placeholder="0"
                />
              </div>

              {/* Credit Amount */}
              <div className="col-span-2">
                <CurrencyInput
                  {...register(`entries.${index}.creditAmount`, { valueAsNumber: true })}
                  error={errors.entries?.[index]?.creditAmount?.message}
                  disabled={disabled}
                  placeholder="0"
                />
              </div>

              {/* Description */}
              <div className="col-span-3">
                <Input
                  placeholder="분개 적요"
                  {...register(`entries.${index}.description`)}
                  disabled={disabled}
                />
              </div>

              {/* Delete Button */}
              <div className="col-span-1 flex items-center justify-center pt-1">
                {!disabled && (
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="text-destructive hover:text-destructive hover:bg-destructive/10"
                    onClick={() => remove(index)}
                    disabled={fields.length <= 2}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                )}
              </div>
            </div>
          ))}
        </div>

        {/* Totals */}
        <div className="grid grid-cols-12 gap-2 px-4 py-3 bg-muted/50 font-semibold border-t">
          <div className="col-span-1"></div>
          <div className="col-span-3 text-right pr-4">합계</div>
          <div className="col-span-2 text-right font-mono">
            {formatCurrency(totalDebit, { showSymbol: false })}
          </div>
          <div className="col-span-2 text-right font-mono">
            {formatCurrency(totalCredit, { showSymbol: false })}
          </div>
          <div className="col-span-4 flex items-center space-x-2">
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
      </div>

      {/* Quick Entry Tips */}
      {!disabled && (
        <div className="text-sm text-muted-foreground">
          <p>* 최소 2개 이상의 분개가 필요합니다.</p>
          <p>* 차변과 대변의 합계가 일치해야 저장할 수 있습니다.</p>
        </div>
      )}
    </div>
  );
}
