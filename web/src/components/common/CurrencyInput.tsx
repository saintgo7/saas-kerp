import { forwardRef, useState, useCallback, useMemo } from "react";
import { cn } from "@/lib/utils";

interface CurrencyInputProps
  extends Omit<React.InputHTMLAttributes<HTMLInputElement>, "type" | "onChange"> {
  error?: string;
  label?: string;
  showSymbol?: boolean;
  allowNegative?: boolean;
  maxValue?: number;
  onChange?: (e: React.ChangeEvent<HTMLInputElement>) => void;
}

export const CurrencyInput = forwardRef<HTMLInputElement, CurrencyInputProps>(
  (
    {
      className,
      error,
      label,
      showSymbol = false,
      allowNegative = false,
      maxValue,
      disabled,
      value,
      onChange,
      ...props
    },
    ref
  ) => {
    // Format number with thousand separators
    const formatNumber = useCallback((num: number | string): string => {
      if (num === "" || num === undefined || num === null) return "";
      const numValue = typeof num === "string" ? parseFloat(num) : num;
      if (isNaN(numValue)) return "";
      return new Intl.NumberFormat("ko-KR").format(numValue);
    }, []);

    // Parse formatted string to number
    const parseNumber = useCallback((str: string): number => {
      if (!str) return 0;
      // Remove thousand separators and parse
      const cleaned = str.replace(/,/g, "").replace(/[^0-9.-]/g, "");
      const num = parseFloat(cleaned);
      return isNaN(num) ? 0 : num;
    }, []);

    // Track if user is actively editing and their local input
    const [editState, setEditState] = useState<{ isEditing: boolean; localValue: string }>({
      isEditing: false,
      localValue: "",
    });

    // Compute display value from controlled value prop
    const controlledDisplayValue = useMemo(() => {
      const numValue = typeof value === "string" ? parseFloat(value as string) : (value as number);
      if (!isNaN(numValue) && numValue !== 0) {
        return formatNumber(numValue);
      }
      return "";
    }, [value, formatNumber]);

    // Use local value when editing, otherwise use controlled value
    const displayValue = editState.isEditing ? editState.localValue : controlledDisplayValue;

    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      let inputValue = e.target.value;

      // Allow only numbers, commas, and optionally minus sign
      if (allowNegative) {
        inputValue = inputValue.replace(/[^0-9,.-]/g, "");
      } else {
        inputValue = inputValue.replace(/[^0-9,.]/g, "");
      }

      // Parse to number
      let numValue = parseNumber(inputValue);

      // Apply max value constraint
      if (maxValue !== undefined && numValue > maxValue) {
        numValue = maxValue;
      }

      // Prevent negative if not allowed
      if (!allowNegative && numValue < 0) {
        numValue = 0;
      }

      // Update local display value
      const formattedValue = formatNumber(numValue);
      setEditState({ isEditing: true, localValue: formattedValue });

      // Create synthetic event with numeric value for form handling
      if (onChange) {
        const syntheticEvent = {
          ...e,
          target: {
            ...e.target,
            value: String(numValue),
            valueAsNumber: numValue,
          },
        } as React.ChangeEvent<HTMLInputElement>;
        onChange(syntheticEvent);
      }
    };

    const handleBlur = (e: React.FocusEvent<HTMLInputElement>) => {
      // Exit editing mode and let controlled value take over
      setEditState({ isEditing: false, localValue: "" });

      if (props.onBlur) {
        props.onBlur(e);
      }
    };

    const handleFocus = (e: React.FocusEvent<HTMLInputElement>) => {
      // Enter editing mode with current controlled value
      setEditState({ isEditing: true, localValue: controlledDisplayValue });
      // Select all on focus for easy editing
      e.target.select();

      if (props.onFocus) {
        props.onFocus(e);
      }
    };

    const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
      // Allow: backspace, delete, tab, escape, enter
      if (
        [8, 46, 9, 27, 13].includes(e.keyCode) ||
        // Allow: Ctrl+A, Ctrl+C, Ctrl+V, Ctrl+X
        (e.ctrlKey && [65, 67, 86, 88].includes(e.keyCode)) ||
        // Allow: Cmd+A, Cmd+C, Cmd+V, Cmd+X (Mac)
        (e.metaKey && [65, 67, 86, 88].includes(e.keyCode)) ||
        // Allow: home, end, left, right
        (e.keyCode >= 35 && e.keyCode <= 39)
      ) {
        return;
      }

      // Allow minus if negative values are allowed
      if (allowNegative && e.key === "-" && e.currentTarget.selectionStart === 0) {
        return;
      }

      // Allow only numbers
      if (!/[0-9]/.test(e.key)) {
        e.preventDefault();
      }
    };

    return (
      <div className="w-full">
        {label && (
          <label className="block text-sm font-medium mb-1.5">{label}</label>
        )}
        <div className="relative">
          {showSymbol && (
            <span className="absolute left-3 top-1/2 transform -translate-y-1/2 text-muted-foreground text-sm">
              ₩
            </span>
          )}
          <input
            ref={ref}
            type="text"
            inputMode="numeric"
            className={cn(
              "flex h-9 w-full rounded-md border border-input bg-background px-3 py-1",
              "text-sm font-mono text-right",
              "placeholder:text-muted-foreground",
              "focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2",
              "disabled:cursor-not-allowed disabled:opacity-50",
              showSymbol && "pl-8",
              error && "border-destructive focus:ring-destructive",
              className
            )}
            disabled={disabled}
            value={displayValue}
            onChange={handleChange}
            onBlur={handleBlur}
            onFocus={handleFocus}
            onKeyDown={handleKeyDown}
            {...props}
          />
        </div>
        {error && <p className="mt-1 text-xs text-destructive">{error}</p>}
      </div>
    );
  }
);

CurrencyInput.displayName = "CurrencyInput";
