import { useState } from "react";
import { Link } from "react-router-dom";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { ArrowLeft, Mail, MailCheck } from "lucide-react";
import {
  Button,
  Input,
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  CardFooter,
} from "@/components/ui";
import { FeatureUnavailable, Logo } from "@/components/common";
import { authService } from "@/services/auth";
import { getErrorMessage } from "@/services/api";
import { toast } from "@/stores/ui";

/**
 * Password reset request.
 *
 * The backend exposes exactly one route for this flow:
 *   POST /api/v1/auth/forgot-password  (internal/router/v1.go)
 * which mails a reset link. There is no endpoint that accepts a reset token
 * plus a new password, so the reset cannot be completed in the app yet.
 *
 * This page previously faked all three steps: it compared the entered code
 * against a hardcoded "123456", slept for a second, and then announced
 * "비밀번호 변경 완료" while discarding the new password. Anyone could reach
 * that success screen for any email address, and users who trusted it were
 * locked out of their own account. Only the step the server actually supports
 * is kept.
 */

const emailSchema = z.object({
  email: z.string().email("올바른 이메일 형식이 아닙니다."),
});

type EmailFormData = z.infer<typeof emailSchema>;

export function ForgotPasswordPage() {
  const [isLoading, setIsLoading] = useState(false);
  const [sentTo, setSentTo] = useState<string | null>(null);
  const [serverMessage, setServerMessage] = useState<string>("");

  const emailForm = useForm<EmailFormData>({
    resolver: zodResolver(emailSchema),
  });

  const handleEmailSubmit = async (data: EmailFormData) => {
    setIsLoading(true);
    try {
      const result = await authService.requestPasswordReset(data.email);
      setSentTo(data.email);
      setServerMessage(
        result?.message ??
          "해당 이메일로 가입된 계정이 있다면 재설정 안내를 보냈습니다."
      );
      toast.success("요청 접수", "비밀번호 재설정 요청을 서버로 보냈습니다.");
    } catch (err) {
      toast.error(
        "요청 실패",
        getErrorMessage(err, "비밀번호 재설정 요청에 실패했습니다.")
      );
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <Card className="w-full max-w-md">
      <CardHeader className="text-center">
        <div className="flex justify-center mb-4">
          <Logo size="xl" />
        </div>
        <div className="flex justify-center mb-2 text-primary">
          {sentTo ? (
            <MailCheck className="h-6 w-6" />
          ) : (
            <Mail className="h-6 w-6" />
          )}
        </div>
        <CardTitle className="text-2xl">
          {sentTo ? "재설정 안내 발송" : "비밀번호 찾기"}
        </CardTitle>
        <CardDescription>
          {sentTo
            ? `${sentTo} 주소로 재설정 안내를 요청했습니다`
            : "가입한 이메일 주소를 입력해주세요"}
        </CardDescription>
      </CardHeader>

      <CardContent className="space-y-4">
        {sentTo ? (
          <>
            <p className="text-sm text-muted-foreground text-center">
              {serverMessage}
            </p>
            <FeatureUnavailable
              feature="앱 내 비밀번호 재설정"
              detail="서버에 재설정 확인 API가 아직 없어 이 화면에서 새 비밀번호를 직접 설정할 수 없습니다. 메일로 받은 안내를 따르거나 관리자에게 문의하십시오."
            />
            <Button
              variant="outline"
              className="w-full"
              size="lg"
              onClick={() => {
                setSentTo(null);
                setServerMessage("");
                emailForm.reset();
              }}
            >
              다른 이메일로 다시 요청
            </Button>
          </>
        ) : (
          <form
            onSubmit={emailForm.handleSubmit(handleEmailSubmit)}
            className="space-y-4"
          >
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
              재설정 안내 받기
            </Button>
          </form>
        )}
      </CardContent>

      <CardFooter className="justify-center">
        <Link
          to="/login"
          className="text-sm text-muted-foreground hover:text-foreground flex items-center gap-1"
        >
          <ArrowLeft className="h-4 w-4" />
          로그인으로 돌아가기
        </Link>
      </CardFooter>
    </Card>
  );
}
