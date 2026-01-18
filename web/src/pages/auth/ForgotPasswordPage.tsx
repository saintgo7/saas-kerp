import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Eye, EyeOff, ArrowLeft, Mail, KeyRound, Lock, CheckCircle } from "lucide-react";
import { Button, Input, Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from "@/components/ui";
import { Logo } from "@/components/common";
import { toast } from "@/stores/ui";

// Step 1: Email input schema
const emailSchema = z.object({
  email: z.string().email("올바른 이메일 형식이 아닙니다."),
});

// Step 2: Verification code schema
const verificationSchema = z.object({
  code: z.string().length(6, "인증번호는 6자리입니다."),
});

// Step 3: New password schema
const passwordSchema = z
  .object({
    password: z
      .string()
      .min(8, "비밀번호는 최소 8자 이상이어야 합니다.")
      .regex(
        /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)/,
        "비밀번호는 대문자, 소문자, 숫자를 포함해야 합니다."
      ),
    confirmPassword: z.string(),
  })
  .refine((data) => data.password === data.confirmPassword, {
    message: "비밀번호가 일치하지 않습니다.",
    path: ["confirmPassword"],
  });

type EmailFormData = z.infer<typeof emailSchema>;
type VerificationFormData = z.infer<typeof verificationSchema>;
type PasswordFormData = z.infer<typeof passwordSchema>;

// Mock verification code for demo purposes
const MOCK_VERIFICATION_CODE = "123456";

export function ForgotPasswordPage() {
  const navigate = useNavigate();
  const [step, setStep] = useState(1);
  const [isLoading, setIsLoading] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [email, setEmail] = useState("");

  // Step 1: Email form
  const emailForm = useForm<EmailFormData>({
    resolver: zodResolver(emailSchema),
  });

  // Step 2: Verification form
  const verificationForm = useForm<VerificationFormData>({
    resolver: zodResolver(verificationSchema),
  });

  // Step 3: Password form
  const passwordForm = useForm<PasswordFormData>({
    resolver: zodResolver(passwordSchema),
  });

  // Handle Step 1: Send verification email
  const handleEmailSubmit = async (data: EmailFormData) => {
    setIsLoading(true);
    try {
      // Mock API call - simulate sending verification email
      await new Promise((resolve) => setTimeout(resolve, 1000));
      setEmail(data.email);
      toast.success("인증번호 발송", "이메일로 인증번호가 발송되었습니다.");
      setStep(2);
    } catch {
      toast.error("발송 실패", "인증번호 발송에 실패했습니다. 다시 시도해주세요.");
    } finally {
      setIsLoading(false);
    }
  };

  // Handle Step 2: Verify code
  const handleVerificationSubmit = async (data: VerificationFormData) => {
    setIsLoading(true);
    try {
      // Mock verification - check if code matches
      await new Promise((resolve) => setTimeout(resolve, 500));
      if (data.code === MOCK_VERIFICATION_CODE) {
        toast.success("인증 완료", "인증번호가 확인되었습니다.");
        setStep(3);
      } else {
        toast.error("인증 실패", "인증번호가 올바르지 않습니다.");
      }
    } catch {
      toast.error("인증 실패", "인증 처리 중 오류가 발생했습니다.");
    } finally {
      setIsLoading(false);
    }
  };

  // Handle Step 3: Set new password
  const handlePasswordSubmit = async (_data: PasswordFormData) => {
    setIsLoading(true);
    try {
      // Mock API call - simulate password reset
      await new Promise((resolve) => setTimeout(resolve, 1000));
      toast.success("비밀번호 변경 완료", "새 비밀번호로 로그인해주세요.");
      setStep(4);
    } catch {
      toast.error("변경 실패", "비밀번호 변경에 실패했습니다. 다시 시도해주세요.");
    } finally {
      setIsLoading(false);
    }
  };

  // Resend verification code
  const handleResendCode = async () => {
    setIsLoading(true);
    try {
      await new Promise((resolve) => setTimeout(resolve, 1000));
      toast.success("재발송 완료", "인증번호가 다시 발송되었습니다.");
    } catch {
      toast.error("발송 실패", "인증번호 발송에 실패했습니다.");
    } finally {
      setIsLoading(false);
    }
  };

  // Get step title and description
  const getStepInfo = () => {
    switch (step) {
      case 1:
        return {
          title: "비밀번호 찾기",
          description: "가입한 이메일 주소를 입력해주세요",
          icon: <Mail className="h-6 w-6" />,
        };
      case 2:
        return {
          title: "인증번호 입력",
          description: `${email}으로 발송된 인증번호를 입력해주세요`,
          icon: <KeyRound className="h-6 w-6" />,
        };
      case 3:
        return {
          title: "새 비밀번호 설정",
          description: "새로운 비밀번호를 입력해주세요",
          icon: <Lock className="h-6 w-6" />,
        };
      case 4:
        return {
          title: "비밀번호 변경 완료",
          description: "비밀번호가 성공적으로 변경되었습니다",
          icon: <CheckCircle className="h-6 w-6 text-green-500" />,
        };
      default:
        return {
          title: "",
          description: "",
          icon: null,
        };
    }
  };

  const stepInfo = getStepInfo();

  return (
    <Card className="w-full max-w-md">
      <CardHeader className="text-center">
        <div className="flex justify-center mb-4">
          <Logo size="xl" />
        </div>
        <div className="flex justify-center mb-2 text-primary">
          {stepInfo.icon}
        </div>
        <CardTitle className="text-2xl">{stepInfo.title}</CardTitle>
        <CardDescription>{stepInfo.description}</CardDescription>

        {/* Step progress indicator */}
        {step < 4 && (
          <div className="flex justify-center items-center space-x-2 pt-4">
            {[1, 2, 3].map((s) => (
              <div key={s} className="flex items-center">
                <div
                  className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium transition-colors ${
                    step >= s
                      ? "bg-primary text-primary-foreground"
                      : "bg-muted text-muted-foreground"
                  }`}
                >
                  {s}
                </div>
                {s < 3 && (
                  <div
                    className={`w-8 h-0.5 mx-1 ${
                      step > s ? "bg-primary" : "bg-muted"
                    }`}
                  />
                )}
              </div>
            ))}
          </div>
        )}
      </CardHeader>

      <CardContent>
        {/* Step 1: Email input */}
        {step === 1 && (
          <form onSubmit={emailForm.handleSubmit(handleEmailSubmit)} className="space-y-4">
            <Input
              type="email"
              label="이메일"
              placeholder="example@company.com"
              error={emailForm.formState.errors.email?.message}
              {...emailForm.register("email")}
            />

            <Button
              type="submit"
              className="w-full"
              size="lg"
              isLoading={isLoading}
            >
              인증번호 발송
            </Button>
          </form>
        )}

        {/* Step 2: Verification code input */}
        {step === 2 && (
          <form onSubmit={verificationForm.handleSubmit(handleVerificationSubmit)} className="space-y-4">
            <Input
              type="text"
              label="인증번호"
              placeholder="6자리 인증번호 입력"
              maxLength={6}
              error={verificationForm.formState.errors.code?.message}
              {...verificationForm.register("code")}
            />

            <p className="text-sm text-muted-foreground text-center">
              인증번호를 받지 못하셨나요?{" "}
              <button
                type="button"
                onClick={handleResendCode}
                disabled={isLoading}
                className="text-primary hover:underline font-medium"
              >
                재발송
              </button>
            </p>

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
                확인
              </Button>
            </div>
          </form>
        )}

        {/* Step 3: New password input */}
        {step === 3 && (
          <form onSubmit={passwordForm.handleSubmit(handlePasswordSubmit)} className="space-y-4">
            <div className="relative">
              <Input
                type={showPassword ? "text" : "password"}
                label="새 비밀번호"
                placeholder="8자 이상, 대소문자 및 숫자 포함"
                error={passwordForm.formState.errors.password?.message}
                {...passwordForm.register("password")}
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
              placeholder="새 비밀번호를 다시 입력하세요"
              error={passwordForm.formState.errors.confirmPassword?.message}
              {...passwordForm.register("confirmPassword")}
            />

            <div className="flex space-x-2">
              <Button
                type="button"
                variant="outline"
                className="flex-1"
                size="lg"
                onClick={() => setStep(2)}
              >
                이전
              </Button>
              <Button
                type="submit"
                className="flex-1"
                size="lg"
                isLoading={isLoading}
              >
                변경하기
              </Button>
            </div>
          </form>
        )}

        {/* Step 4: Success message */}
        {step === 4 && (
          <div className="space-y-4 text-center">
            <div className="flex justify-center">
              <div className="w-16 h-16 rounded-full bg-green-100 flex items-center justify-center">
                <CheckCircle className="h-8 w-8 text-green-500" />
              </div>
            </div>
            <p className="text-muted-foreground">
              비밀번호가 성공적으로 변경되었습니다.
              <br />
              새 비밀번호로 로그인해주세요.
            </p>
            <Button
              className="w-full"
              size="lg"
              onClick={() => navigate("/login")}
            >
              로그인 페이지로 이동
            </Button>
          </div>
        )}
      </CardContent>

      {step < 4 && (
        <CardFooter className="justify-center">
          <Link
            to="/login"
            className="text-sm text-muted-foreground hover:text-foreground flex items-center gap-1"
          >
            <ArrowLeft className="h-4 w-4" />
            로그인으로 돌아가기
          </Link>
        </CardFooter>
      )}
    </Card>
  );
}
