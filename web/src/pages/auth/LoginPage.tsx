import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Eye, EyeOff } from "lucide-react";
import { Button, Input, Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from "@/components/ui";
import { Logo } from "@/components/common";
import { useAuthStore } from "@/stores";
import { toast } from "@/stores/ui";

const loginSchema = z.object({
  email: z.string().email("올바른 이메일 형식이 아닙니다."),
  password: z.string().min(8, "비밀번호는 최소 8자 이상이어야 합니다."),
});

type LoginFormData = z.infer<typeof loginSchema>;

export function LoginPage() {
  const navigate = useNavigate();
  const { login, isLoading } = useAuthStore();
  const [showPassword, setShowPassword] = useState(false);

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginFormData>({
    resolver: zodResolver(loginSchema),
  });

  const onSubmit = async (data: LoginFormData) => {
    try {
      await login(data.email, data.password);
      toast.success("로그인 성공", "환영합니다!");
      navigate("/dashboard");
    } catch {
      toast.error("로그인 실패", "이메일 또는 비밀번호를 확인해주세요.");
    }
  };

  return (
    <Card>
      <CardHeader className="text-center">
        <div className="flex justify-center mb-4">
          <Logo size="xl" />
        </div>
        <CardTitle className="text-2xl">로그인</CardTitle>
        <CardDescription>
          K-ERP 계정으로 로그인하세요
        </CardDescription>
      </CardHeader>

      <CardContent>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <Input
            type="email"
            label="이메일"
            placeholder="example@company.com"
            error={errors.email?.message}
            {...register("email")}
          />

          <div className="relative">
            <Input
              type={showPassword ? "text" : "password"}
              label="비밀번호"
              placeholder="비밀번호를 입력하세요"
              error={errors.password?.message}
              {...register("password")}
            />
            <button
              type="button"
              onClick={() => setShowPassword(!showPassword)}
              className="absolute right-3 top-9 text-muted-foreground hover:text-foreground"
            >
              {showPassword ? (
                <EyeOff className="h-4 w-4" />
              ) : (
                <Eye className="h-4 w-4" />
              )}
            </button>
          </div>

          <div className="flex items-center justify-between">
            <label className="flex items-center space-x-2 text-sm">
              <input type="checkbox" className="rounded border-input" />
              <span>로그인 상태 유지</span>
            </label>
            <Link
              to="/forgot-password"
              className="text-sm text-primary hover:underline"
            >
              비밀번호 찾기
            </Link>
          </div>

          <Button
            type="submit"
            className="w-full"
            size="lg"
            isLoading={isLoading}
          >
            로그인
          </Button>
        </form>
      </CardContent>

      <CardFooter className="justify-center">
        <p className="text-sm text-muted-foreground">
          계정이 없으신가요?{" "}
          <Link to="/register" className="text-primary hover:underline font-medium">
            회원가입
          </Link>
        </p>
      </CardFooter>
    </Card>
  );
}
