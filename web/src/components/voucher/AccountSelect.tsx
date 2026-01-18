import { useState, useRef, useEffect, useMemo } from "react";
import { Controller } from "react-hook-form";
import type { Control } from "react-hook-form";
import { useQuery } from "@tanstack/react-query";
import { Search, ChevronDown, X, Loader2 } from "lucide-react";
import { cn, debounce } from "@/lib/utils";
import { accountsApi } from "@/api";
import type { AccountType } from "@/types";

interface AccountSelectProps {
  name: string;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  control: Control<any>;
  error?: string;
  disabled?: boolean;
  accountType?: AccountType;
  placeholder?: string;
  onSelect?: (account: { id: string; code: string; name: string } | null) => void;
}

interface AccountOption {
  id: string;
  code: string;
  name: string;
  type: AccountType;
  level: number;
}

export function AccountSelect({
  name,
  control,
  error,
  disabled = false,
  accountType,
  placeholder = "계정과목 선택",
  onSelect,
}: AccountSelectProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [searchTerm, setSearchTerm] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // Fetch accounts
  const { data: accountsResponse, isLoading } = useQuery({
    queryKey: ["accounts", "search", debouncedSearch, accountType],
    queryFn: async () => {
      if (debouncedSearch) {
        return accountsApi.search(debouncedSearch, accountType);
      }
      return accountsApi.list({ type: accountType, pageSize: 50, isActive: true });
    },
    enabled: isOpen,
  });

  const accounts: AccountOption[] = Array.isArray(accountsResponse?.data)
    ? accountsResponse.data
    : (accountsResponse?.data as { items?: AccountOption[] })?.items || [];

  // Debounced search using useMemo
  const debouncedSetSearch = useMemo(
    () =>
      debounce((value: string) => {
        setDebouncedSearch(value);
      }, 300),
    []
  );

  useEffect(() => {
    debouncedSetSearch(searchTerm);
  }, [searchTerm, debouncedSetSearch]);

  // Close dropdown on outside click
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setIsOpen(false);
        setSearchTerm("");
      }
    }

    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  // Focus input when dropdown opens
  useEffect(() => {
    if (isOpen && inputRef.current) {
      inputRef.current.focus();
    }
  }, [isOpen]);

  const getAccountTypeLabel = (type: AccountType) => {
    const labels: Record<AccountType, string> = {
      asset: "자산",
      liability: "부채",
      equity: "자본",
      revenue: "수익",
      expense: "비용",
    };
    return labels[type];
  };

  const getAccountTypeColor = (type: AccountType) => {
    const colors: Record<AccountType, string> = {
      asset: "text-blue-600",
      liability: "text-red-600",
      equity: "text-purple-600",
      revenue: "text-green-600",
      expense: "text-orange-600",
    };
    return colors[type];
  };

  return (
    <Controller
      name={name}
      control={control}
      render={({ field }) => {
        // Find selected account
        const selectedAccount = accounts.find((a) => a.id === field.value);

        const handleSelect = (account: AccountOption) => {
          field.onChange(account.id);
          setIsOpen(false);
          setSearchTerm("");
          onSelect?.({ id: account.id, code: account.code, name: account.name });
        };

        const handleClear = (e: React.MouseEvent) => {
          e.stopPropagation();
          field.onChange("");
          onSelect?.(null);
        };

        return (
          <div ref={containerRef} className="relative">
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
              {selectedAccount ? (
                <div className="flex items-center justify-between w-full">
                  <div className="flex items-center space-x-2 truncate">
                    <span className="font-mono text-xs bg-muted px-1.5 py-0.5 rounded">
                      {selectedAccount.code}
                    </span>
                    <span className="truncate">{selectedAccount.name}</span>
                  </div>
                  {!disabled && (
                    <button
                      type="button"
                      className="p-0.5 hover:bg-muted rounded"
                      onClick={handleClear}
                    >
                      <X className="h-3 w-3 text-muted-foreground" />
                    </button>
                  )}
                </div>
              ) : (
                <div className="flex items-center justify-between w-full text-muted-foreground">
                  <span>{placeholder}</span>
                  <ChevronDown className="h-4 w-4" />
                </div>
              )}
            </div>

            {/* Error message */}
            {error && <p className="mt-1 text-xs text-destructive">{error}</p>}

            {/* Dropdown */}
            {isOpen && !disabled && (
              <div className="absolute z-50 w-full mt-1 bg-popover border rounded-md shadow-lg">
                {/* Search Input */}
                <div className="p-2 border-b">
                  <div className="relative">
                    <Search className="absolute left-2 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                    <input
                      ref={inputRef}
                      type="text"
                      className="w-full h-8 pl-8 pr-3 text-sm border rounded-md focus:outline-none focus:ring-2 focus:ring-ring"
                      placeholder="코드 또는 이름으로 검색..."
                      value={searchTerm}
                      onChange={(e) => setSearchTerm(e.target.value)}
                    />
                  </div>
                </div>

                {/* Options */}
                <div className="max-h-60 overflow-y-auto">
                  {isLoading ? (
                    <div className="flex items-center justify-center py-4">
                      <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
                    </div>
                  ) : accounts.length === 0 ? (
                    <div className="py-4 text-center text-sm text-muted-foreground">
                      검색 결과가 없습니다.
                    </div>
                  ) : (
                    accounts.map((account) => (
                      <div
                        key={account.id}
                        className={cn(
                          "flex items-center justify-between px-3 py-2 cursor-pointer hover:bg-muted",
                          field.value === account.id && "bg-primary/10"
                        )}
                        style={{ paddingLeft: `${(account.level || 0) * 12 + 12}px` }}
                        onClick={() => handleSelect(account)}
                      >
                        <div className="flex items-center space-x-2">
                          <span className="font-mono text-xs bg-muted px-1.5 py-0.5 rounded">
                            {account.code}
                          </span>
                          <span className="text-sm">{account.name}</span>
                        </div>
                        <span
                          className={cn(
                            "text-xs",
                            getAccountTypeColor(account.type)
                          )}
                        >
                          {getAccountTypeLabel(account.type)}
                        </span>
                      </div>
                    ))
                  )}
                </div>
              </div>
            )}
          </div>
        );
      }}
    />
  );
}
