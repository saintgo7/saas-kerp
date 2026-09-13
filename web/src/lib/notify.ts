import { toast } from "@/stores/ui";

/**
 * Toast for an action whose backend endpoint does not exist yet.
 *
 * Screens used to log to the console and then show a success toast, so users
 * believed data had been saved when nothing left the browser.
 */
export function notifyUnavailable(feature: string): void {
  toast.warning(
    `${feature} 기능 미구현`,
    "서버에 해당 API가 없어 처리하지 못했습니다. 입력한 내용은 저장되지 않았습니다."
  );
}
