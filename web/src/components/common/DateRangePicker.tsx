import { useState, useRef, useEffect, useCallback } from "react";
import { Calendar, ChevronLeft, ChevronRight, X } from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui";

interface DateRangePickerProps {
  startDate: string;
  endDate: string;
  onDateChange: (startDate: string, endDate: string) => void;
  disabled?: boolean;
  minDate?: string;
  maxDate?: string;
  label?: string;
  error?: string;
  presets?: boolean;
}

interface PresetOption {
  label: string;
  getRange: () => { start: string; end: string };
}

export function DateRangePicker({
  startDate,
  endDate,
  onDateChange,
  disabled = false,
  minDate,
  maxDate,
  label,
  error,
  presets = true,
}: DateRangePickerProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [currentMonth, setCurrentMonth] = useState(() => {
    const date = startDate ? new Date(startDate) : new Date();
    return new Date(date.getFullYear(), date.getMonth(), 1);
  });
  const [selectingStart, setSelectingStart] = useState(true);
  const containerRef = useRef<HTMLDivElement>(null);

  // Close dropdown on outside click
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    }

    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const formatDate = (dateStr: string) => {
    if (!dateStr) return "";
    const date = new Date(dateStr);
    return date.toLocaleDateString("ko-KR", {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
    });
  };

  const toISODate = (date: Date) => {
    return date.toISOString().split("T")[0];
  };

  const presetOptions: PresetOption[] = [
    {
      label: "오늘",
      getRange: () => {
        const today = toISODate(new Date());
        return { start: today, end: today };
      },
    },
    {
      label: "이번 주",
      getRange: () => {
        const today = new Date();
        const dayOfWeek = today.getDay();
        const start = new Date(today);
        start.setDate(today.getDate() - dayOfWeek);
        const end = new Date(start);
        end.setDate(start.getDate() + 6);
        return { start: toISODate(start), end: toISODate(end) };
      },
    },
    {
      label: "이번 달",
      getRange: () => {
        const today = new Date();
        const start = new Date(today.getFullYear(), today.getMonth(), 1);
        const end = new Date(today.getFullYear(), today.getMonth() + 1, 0);
        return { start: toISODate(start), end: toISODate(end) };
      },
    },
    {
      label: "지난 달",
      getRange: () => {
        const today = new Date();
        const start = new Date(today.getFullYear(), today.getMonth() - 1, 1);
        const end = new Date(today.getFullYear(), today.getMonth(), 0);
        return { start: toISODate(start), end: toISODate(end) };
      },
    },
    {
      label: "이번 분기",
      getRange: () => {
        const today = new Date();
        const quarter = Math.floor(today.getMonth() / 3);
        const start = new Date(today.getFullYear(), quarter * 3, 1);
        const end = new Date(today.getFullYear(), quarter * 3 + 3, 0);
        return { start: toISODate(start), end: toISODate(end) };
      },
    },
    {
      label: "올해",
      getRange: () => {
        const today = new Date();
        const start = new Date(today.getFullYear(), 0, 1);
        const end = new Date(today.getFullYear(), 11, 31);
        return { start: toISODate(start), end: toISODate(end) };
      },
    },
    {
      label: "작년",
      getRange: () => {
        const today = new Date();
        const start = new Date(today.getFullYear() - 1, 0, 1);
        const end = new Date(today.getFullYear() - 1, 11, 31);
        return { start: toISODate(start), end: toISODate(end) };
      },
    },
  ];

  const getDaysInMonth = useCallback((date: Date) => {
    return new Date(date.getFullYear(), date.getMonth() + 1, 0).getDate();
  }, []);

  const getFirstDayOfMonth = useCallback((date: Date) => {
    return new Date(date.getFullYear(), date.getMonth(), 1).getDay();
  }, []);

  const navigateMonth = (direction: "prev" | "next") => {
    setCurrentMonth((prev) => {
      const newMonth = new Date(prev);
      newMonth.setMonth(prev.getMonth() + (direction === "next" ? 1 : -1));
      return newMonth;
    });
  };

  const handleDateClick = (date: Date) => {
    const dateStr = toISODate(date);

    if (selectingStart) {
      onDateChange(dateStr, endDate && dateStr <= endDate ? endDate : dateStr);
      setSelectingStart(false);
    } else {
      if (dateStr < startDate) {
        onDateChange(dateStr, startDate);
      } else {
        onDateChange(startDate, dateStr);
      }
      setSelectingStart(true);
      setIsOpen(false);
    }
  };

  const handlePresetClick = (preset: PresetOption) => {
    const range = preset.getRange();
    onDateChange(range.start, range.end);
    setIsOpen(false);
  };

  const isDateInRange = (date: Date) => {
    if (!startDate || !endDate) return false;
    const dateStr = toISODate(date);
    return dateStr >= startDate && dateStr <= endDate;
  };

  const isDateDisabled = (date: Date) => {
    const dateStr = toISODate(date);
    if (minDate && dateStr < minDate) return true;
    if (maxDate && dateStr > maxDate) return true;
    return false;
  };

  const isStartDate = (date: Date) => {
    return startDate && toISODate(date) === startDate;
  };

  const isEndDate = (date: Date) => {
    return endDate && toISODate(date) === endDate;
  };

  const renderCalendar = () => {
    const daysInMonth = getDaysInMonth(currentMonth);
    const firstDayOfMonth = getFirstDayOfMonth(currentMonth);
    const days: JSX.Element[] = [];

    // Empty cells for days before the first day of the month
    for (let i = 0; i < firstDayOfMonth; i++) {
      days.push(<div key={`empty-${i}`} className="h-8 w-8" />);
    }

    // Days of the month
    for (let day = 1; day <= daysInMonth; day++) {
      const date = new Date(currentMonth.getFullYear(), currentMonth.getMonth(), day);
      const isDisabled = isDateDisabled(date);
      const isInRange = isDateInRange(date);
      const isStart = isStartDate(date);
      const isEnd = isEndDate(date);
      const isToday = toISODate(date) === toISODate(new Date());

      days.push(
        <button
          key={day}
          type="button"
          disabled={isDisabled}
          className={cn(
            "h-8 w-8 rounded-md text-sm font-medium transition-colors",
            "hover:bg-primary hover:text-primary-foreground",
            "focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-1",
            isDisabled && "opacity-30 cursor-not-allowed hover:bg-transparent",
            isInRange && !isStart && !isEnd && "bg-primary/20",
            (isStart || isEnd) && "bg-primary text-primary-foreground",
            isToday && !isInRange && "border border-primary"
          )}
          onClick={() => handleDateClick(date)}
        >
          {day}
        </button>
      );
    }

    return days;
  };

  const handleClear = (e: React.MouseEvent) => {
    e.stopPropagation();
    onDateChange("", "");
  };

  return (
    <div ref={containerRef} className="relative">
      {label && (
        <label className="block text-sm font-medium mb-1.5">{label}</label>
      )}

      {/* Trigger */}
      <div
        className={cn(
          "flex items-center h-9 w-full rounded-md border border-input bg-background px-3 text-sm",
          "cursor-pointer hover:bg-muted/50 transition-colors",
          disabled && "opacity-50 cursor-not-allowed",
          error && "border-destructive",
          isOpen && "ring-2 ring-ring ring-offset-2"
        )}
        onClick={() => !disabled && setIsOpen(!isOpen)}
      >
        <Calendar className="h-4 w-4 text-muted-foreground mr-2" />
        <div className="flex-1">
          {startDate && endDate ? (
            <span>
              {formatDate(startDate)} ~ {formatDate(endDate)}
            </span>
          ) : (
            <span className="text-muted-foreground">기간 선택</span>
          )}
        </div>
        {startDate && !disabled && (
          <button
            type="button"
            className="p-0.5 hover:bg-muted rounded"
            onClick={handleClear}
          >
            <X className="h-3 w-3 text-muted-foreground" />
          </button>
        )}
      </div>

      {error && <p className="mt-1 text-xs text-destructive">{error}</p>}

      {/* Dropdown */}
      {isOpen && !disabled && (
        <div className="absolute z-50 mt-1 bg-popover border rounded-lg shadow-lg">
          <div className="flex">
            {/* Presets */}
            {presets && (
              <div className="w-32 border-r p-2">
                <div className="text-xs font-medium text-muted-foreground mb-2 px-2">
                  빠른 선택
                </div>
                <div className="space-y-0.5">
                  {presetOptions.map((preset) => (
                    <button
                      key={preset.label}
                      type="button"
                      className="w-full text-left px-2 py-1.5 text-sm rounded hover:bg-muted"
                      onClick={() => handlePresetClick(preset)}
                    >
                      {preset.label}
                    </button>
                  ))}
                </div>
              </div>
            )}

            {/* Calendar */}
            <div className="p-4">
              {/* Header */}
              <div className="flex items-center justify-between mb-4">
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-7 w-7"
                  onClick={() => navigateMonth("prev")}
                >
                  <ChevronLeft className="h-4 w-4" />
                </Button>
                <span className="font-semibold">
                  {currentMonth.toLocaleDateString("ko-KR", {
                    year: "numeric",
                    month: "long",
                  })}
                </span>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-7 w-7"
                  onClick={() => navigateMonth("next")}
                >
                  <ChevronRight className="h-4 w-4" />
                </Button>
              </div>

              {/* Days of week */}
              <div className="grid grid-cols-7 gap-1 mb-2">
                {["일", "월", "화", "수", "목", "금", "토"].map((day) => (
                  <div
                    key={day}
                    className="h-8 w-8 flex items-center justify-center text-xs text-muted-foreground font-medium"
                  >
                    {day}
                  </div>
                ))}
              </div>

              {/* Days */}
              <div className="grid grid-cols-7 gap-1">{renderCalendar()}</div>

              {/* Selection hint */}
              <div className="mt-3 pt-3 border-t text-xs text-muted-foreground text-center">
                {selectingStart ? "시작일을 선택하세요" : "종료일을 선택하세요"}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
