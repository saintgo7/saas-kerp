import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Eye, EyeOff } from "lucide-react";
import { Button, Input, Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from "@/components/ui";
import { useAuthStore } from "@/stores";
import { toast } from "@/stores/ui";

const registerSchema = z
  .object({
    companyName: z.string().min(2, "회사명은 최소 2자 이상이어야 합니다."),
    businessNumber: z
      .string()
      .regex(/^\d{3}-\d{2}-\d{5}$|^\d{10}$/, "올바른 사업자등록번호 형식이 아닙니다."),
    name: z.string().min(2, "이름은 최소 2자 이상이어야 합니다."),
    email: z.string().email("올바른 이메일 형식이 아닙니다."),
    phone: z
      .string()
      .regex(/^01\d{1}-?\d{3,4}-?\d{4}$/, "올바른 휴대폰 번호 형식이 아닙니다.")
      .optional()
      .or(z.literal("")),
    password: z
      .string()
      .min(8, "비밀번호는 최소 8자 이상이어야 합니다.")
      .regex(
        /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)/,
        "비밀번호는 대문자, 소문자, 숫자를 포함해야 합니다."
      ),
    confirmPassword: z.string(),
    agreeTerms: z.boolean().refine((val) => val === true, "이용약관에 동의해주세요."),
  })
  .refine((data) => data.password === data.confirmPassword, {
    message: "비밀번호가 일치하지 않습니다.",
    path: ["confirmPassword"],
  });

type RegisterFormData = z.infer<typeof registerSchema>;

export function RegisterPage() {
  const navigate = useNavigate();
  const { register: registerUser, isLoading } = useAuthStore();
  const [showPassword, setShowPassword] = useState(false);
  const [step, setStep] = useState(1);

  const {
    register,
    handleSubmit,
    formState: { errors },
    trigger,
  } = useForm<RegisterFormData>({
    resolver: zodResolver(registerSchema),
  });

  const handleNextStep = async () => {
    const isValid = await trigger(["companyName", "businessNumber"]);
    if (isValid) {
      setStep(2);
    }
  };

  const onSubmit = async (data: RegisterFormData) => {
    try {
      await registerUser({
        email: data.email,
        password: data.password,
        name: data.name,
        phone: data.phone || undefined,
        companyName: data.companyName,
        businessNumber: data.businessNumber.replace(/-/g, ""),
      });
      toast.success("회원가입 완료", "K-ERP에 오신 것을 환영합니다!");
      navigate("/dashboard");
    } catch {
      toast.error("회원가입 실패", "입력 정보를 확인해주세요.");
    }
  };

  return (
    <Card>
      <CardHeader className="text-center">
        <div className="flex justify-center mb-4 lg:hidden">
          <div className="w-12 h-12 bg-primary rounded-xl flex items-center justify-center">
            <span className="text-primary-foreground font-bold text-2xl">K</span>
          </div>
        </div>
        <CardTitle className="text-2xl">회원가입</CardTitle>
        <CardDescription>
          {step === 1 ? "회사 정보를 입력해주세요" : "계정 정보를 입력해주세요"}
        </CardDescription>

        {/* Step indicator */}
        <div className="flex justify-center space-x-2 pt-4">
          <div
            className={`w-3 h-3 rounded-full ${
              step >= 1 ? "bg-primary" : "bg-muted"
            }`}
          />
          <div
            className={`w-3 h-3 rounded-full ${
              step >= 2 ? "bg-primary" : "bg-muted"
            }`}
          />
        </div>
      </CardHeader>

      <CardContent>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          {step === 1 && (
            <>
              <Input
                label="회사명"
                placeholder="주식회사 OO"
                error={errors.companyName?.message}
                required
                {...register("companyName")}
              />

              <Input
                label="사업자등록번호"
                placeholder="000-00-00000"
                error={errors.businessNumber?.message}
                required
                {...register("businessNumber")}
              />

              <Button
                type="button"
                className="w-full"
                size="lg"
                onClick={handleNextStep}
              >
                다음
              </Button>
            </>
          )}

          {step === 2 && (
            <>
              <Input
                label="이름"
                placeholder="홍길동"
                error={errors.name?.message}
                required
                {...register("name")}
              />

              <Input
                type="email"
                label="이메일"
                placeholder="example@company.com"
                error={errors.email?.message}
                required
                {...register("email")}
              />

              <Input
                type="tel"
                label="휴대폰 번호"
                placeholder="010-0000-0000"
                error={errors.phone?.message}
                {...register("phone")}
              />

              <div className="relative">
                <Input
                  type={showPassword ? "text" : "password"}
                  label="비밀번호"
                  placeholder="8자 이상, 대소문자 및 숫자 포함"
                  error={errors.password?.message}
                  required
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

              <Input
                type="password"
                label="비밀번호 확인"
                placeholder="비밀번호를 다시 입력하세요"
                error={errors.confirmPassword?.message}
                required
                {...register("confirmPassword")}
              />

              <div className="space-y-2">
                <label className="flex items-start space-x-2 text-sm">
                  <input
                    type="checkbox"
                    className="rounded border-input mt-1"
                    {...register("agreeTerms")}
                  />
                  <span>
                    <Link to="/terms" className="text-primary hover:underline">
                      이용약관
                    </Link>
                    과{" "}
                    <Link to="/privacy" className="text-primary hover:underline">
                      개인정보처리방침
                    </Link>
                    에 동의합니다.
                  </span>
                </label>
                {errors.agreeTerms && (
                  <p className="text-sm text-destructive">
                    {errors.agreeTerms.message}
                  </p>
                )}
              </div>

              <div className="flex space-x-2">
                <Button
                  type="button"
                  variant="outline"
                  className="flex-1"
                  size="lg"
                  onClick={() => setStep(1)}
                >
                  이전
                </Button>
                <Button
                  type="submit"
                  className="flex-1"
                  size="lg"
                  isLoading={isLoading}
                >
                  가입하기
                </Button>
              </div>
            </>
          )}
        </form>
      </CardContent>

      <CardFooter className="justify-center">
        <p className="text-sm text-muted-foreground">
          이미 계정이 있으신가요?{" "}
          <Link to="/login" className="text-primary hover:underline font-medium">
            로그인
          </Link>
        </p>
      </CardFooter>
    </Card>
  );
}
