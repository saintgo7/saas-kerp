import { AlertTriangle } from "lucide-react";
import { cn } from "@/lib/utils";

/**
 * Marks a screen or action whose backend endpoint does not exist yet.
 *
 * These screens used to log to the console and then show a "저장 완료" toast,
 * so users believed data had been saved when nothing left the browser. Anything
 * without a server endpoint must say so instead of faking success.
 */

interface FeatureUnavailableProps {
  /** What is missing, e.g. "직원 등록". */
  feature: string;
  /** Optional extra context shown under the headline. */
  detail?: string;
  className?: string;
}

export function FeatureUnavailable({
  feature,
  detail,
  className,
}: FeatureUnavailableProps) {
  return (
    <div
      role="status"
      className={cn(
        "flex items-start gap-3 rounded-lg border border-amber-500/40 bg-amber-50 p-4 text-amber-900",
        className
      )}
    >
      <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0" aria-hidden="true" />
      <div className="min-w-0 text-sm">
        <p className="font-semibold">{feature} 기능은 아직 제공되지 않습니다</p>
        <p className="mt-1 opacity-90">
          {detail ??
            "서버에 해당 API가 아직 없어 입력한 내용은 저장되지 않습니다. 화면의 값은 예시이며 실제 데이터가 아닙니다."}
        </p>
      </div>
    </div>
  );
}

export default FeatureUnavailable;
