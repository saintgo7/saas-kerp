import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { authService } from "@/services/auth";
import { useAuthStore } from "@/stores";
import { toast } from "@/stores/ui";

// Query keys
export const authKeys = {
  all: ["auth"] as const,
  me: () => [...authKeys.all, "me"] as const,
};

// Get current user
export function useCurrentUser() {
  const { isAuthenticated } = useAuthStore();

  return useQuery({
    queryKey: authKeys.me(),
    queryFn: authService.getCurrentUser,
    enabled: isAuthenticated,
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
}

// Login mutation
export function useLogin() {
  const queryClient = useQueryClient();
  const { login } = useAuthStore();

  return useMutation({
    mutationFn: ({ email, password }: { email: string; password: string }) =>
      login(email, password),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: authKeys.all });
      toast.success("로그인 성공", "환영합니다!");
    },
    onError: (error: Error) => {
      toast.error("로그인 실패", error.message || "이메일 또는 비밀번호를 확인하세요.");
    },
  });
}

// Logout mutation
export function useLogout() {
  const queryClient = useQueryClient();
  const { logout } = useAuthStore();

  return useMutation({
    mutationFn: logout,
    onSuccess: () => {
      queryClient.clear();
      toast.success("로그아웃", "안전하게 로그아웃되었습니다.");
    },
  });
}

// Register mutation
export function useRegister() {
  return useMutation({
    mutationFn: authService.register,
    onSuccess: () => {
      toast.success("회원가입 완료", "로그인 페이지로 이동합니다.");
    },
    onError: (error: Error) => {
      toast.error("회원가입 실패", error.message || "회원가입 중 오류가 발생했습니다.");
    },
  });
}
