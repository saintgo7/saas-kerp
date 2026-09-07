import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  User,
  Save,
  Lock,
  Bell,
  Clock,
  Mail,
  Phone,
  Shield,
  Monitor,
  Smartphone,
  MapPin,
  Eye,
  EyeOff,
  CheckCircle,
  AlertCircle,
} from "lucide-react";
import {
  Button,
  Input,
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  Badge,
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from "@/components/ui";
import { formatDate, formatPhoneNumber } from "@/lib/utils";
import { toast } from "@/stores/ui";
import { useAuthStore } from "@/stores";
import { authService, } from "@/services/auth";
import { getErrorMessage } from "@/services/api";
import { FeatureUnavailable, notifyUnavailable } from "@/components/common";
import { USER_ROLES } from "@/constants";

// Validation schemas
const profileSchema = z.object({
  name: z.string().min(1, "이름을 입력하세요"),
  phone: z.string().optional(),
  email: z.string().email("올바른 이메일 형식이 아닙니다"),
});

const passwordSchema = z
  .object({
    currentPassword: z.string().min(1, "현재 비밀번호를 입력하세요"),
    newPassword: z
      .string()
      .min(8, "비밀번호는 8자 이상이어야 합니다")
      .regex(
        /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)/,
        "비밀번호는 대문자, 소문자, 숫자를 포함해야 합니다"
      ),
    confirmPassword: z.string().min(1, "비밀번호 확인을 입력하세요"),
  })
  .refine((data) => data.newPassword === data.confirmPassword, {
    message: "비밀번호가 일치하지 않습니다",
    path: ["confirmPassword"],
  });

type ProfileFormData = z.infer<typeof profileSchema>;
type PasswordFormData = z.infer<typeof passwordSchema>;

// Mock notification settings
interface NotificationSetting {
  id: string;
  label: string;
  description: string;
  email: boolean;
  push: boolean;
}

const mockNotificationSettings: NotificationSetting[] = [
  {
    id: "voucher_approval",
    label: "전표 승인 요청",
    description: "전표 승인 요청이 도착하면 알림을 받습니다.",
    email: true,
    push: true,
  },
  {
    id: "invoice_issued",
    label: "세금계산서 발행",
    description: "세금계산서가 발행되면 알림을 받습니다.",
    email: true,
    push: false,
  },
  {
    id: "payroll_processed",
    label: "급여 처리 완료",
    description: "급여 처리가 완료되면 알림을 받습니다.",
    email: true,
    push: true,
  },
  {
    id: "system_update",
    label: "시스템 공지",
    description: "시스템 업데이트 및 공지사항을 받습니다.",
    email: false,
    push: true,
  },
  {
    id: "security_alert",
    label: "보안 알림",
    description: "비정상 로그인 시도 등 보안 관련 알림을 받습니다.",
    email: true,
    push: true,
  },
];

// Mock login history
interface LoginHistory {
  id: string;
  timestamp: string;
  device: string;
  browser: string;
  ip: string;
  location: string;
  success: boolean;
}

const mockLoginHistory: LoginHistory[] = [
  {
    id: "1",
    timestamp: "2024-01-15T09:30:00",
    device: "Desktop",
    browser: "Chrome 120",
    ip: "192.168.1.100",
    location: "서울, 대한민국",
    success: true,
  },
  {
    id: "2",
    timestamp: "2024-01-14T18:20:00",
    device: "Mobile",
    browser: "Safari Mobile",
    ip: "192.168.1.101",
    location: "서울, 대한민국",
    success: true,
  },
  {
    id: "3",
    timestamp: "2024-01-14T10:15:00",
    device: "Desktop",
    browser: "Chrome 120",
    ip: "192.168.1.100",
    location: "서울, 대한민국",
    success: true,
  },
  {
    id: "4",
    timestamp: "2024-01-13T22:45:00",
    device: "Desktop",
    browser: "Firefox 121",
    ip: "103.215.24.55",
    location: "부산, 대한민국",
    success: false,
  },
  {
    id: "5",
    timestamp: "2024-01-13T14:30:00",
    device: "Desktop",
    browser: "Chrome 120",
    ip: "192.168.1.100",
    location: "서울, 대한민국",
    success: true,
  },
];

export function ProfilePage() {
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [showCurrentPassword, setShowCurrentPassword] = useState(false);
  const [showNewPassword, setShowNewPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const [notificationSettings, setNotificationSettings] = useState(
    mockNotificationSettings
  );

  // The signed-in user, not a hardcoded demo account. Every visitor used to
  // see "김관리 / admin@techsolution.co.kr" regardless of who was logged in.
  const user = useAuthStore((state) => state.user);

  // Profile form
  const {
    register: registerProfile,
    handleSubmit: handleSubmitProfile,
    formState: { errors: profileErrors },
  } = useForm<ProfileFormData>({
    resolver: zodResolver(profileSchema),
    values: {
      name: user?.name ?? "",
      phone: user?.phone ?? "",
      email: user?.email ?? "",
    },
  });

  // Password form
  const {
    register: registerPassword,
    handleSubmit: handleSubmitPassword,
    reset: resetPassword,
    formState: { errors: passwordErrors },
  } = useForm<PasswordFormData>({
    resolver: zodResolver(passwordSchema),
  });

  // The backend has no "update my own profile" route: /users/:id is the admin
  // CRUD endpoint (and would let a user rewrite their own role), and it carries
  // no phone field at all. Say so rather than pretending the save worked.
  const onSubmitProfile = () => {
    notifyUnavailable("프로필 수정");
  };

  // PUT /api/v1/auth/password (internal/handler/auth.go ChangePasswordRequest)
  const onSubmitPassword = async (data: PasswordFormData) => {
    setIsSubmitting(true);
    try {
      await authService.changePassword(data.currentPassword, data.newPassword);
      toast.success("변경 완료", "비밀번호가 변경되었습니다.");
      resetPassword();
    } catch (err) {
      toast.error(
        "변경 실패",
        getErrorMessage(err, "비밀번호 변경 중 오류가 발생했습니다.")
      );
    } finally {
      setIsSubmitting(false);
    }
  };

  // Handle notification toggle
  const toggleNotification = (
    settingId: string,
    type: "email" | "push"
  ) => {
    setNotificationSettings(
      notificationSettings.map((setting) =>
        setting.id === settingId
          ? { ...setting, [type]: !setting[type] }
          : setting
      )
    );
    // Toggling is local only: there is no notification-settings endpoint.
    notifyUnavailable("알림 설정 저장");
  };

  if (!user) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold">내 프로필</h1>
        </div>
        <p className="text-muted-foreground">
          로그인 정보를 불러오지 못했습니다. 다시 로그인해 주십시오.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">내 프로필</h1>
          <p className="text-muted-foreground">계정 정보 및 설정을 관리합니다.</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Profile Summary */}
        <Card className="lg:col-span-1">
          <CardContent className="pt-6">
            <div className="flex flex-col items-center text-center">
              <div className="w-24 h-24 rounded-full bg-primary/10 flex items-center justify-center mb-4">
                <span className="text-3xl font-bold text-primary">
                  {user.name.charAt(0)}
                </span>
              </div>
              <h2 className="text-xl font-bold">{user.name}</h2>
              <Badge variant="outline" className="mt-2">
                <Shield className="h-3 w-3 mr-1" />
                {USER_ROLES[user.role]}
              </Badge>
              <div className="w-full mt-6 space-y-3">
                <div className="flex items-center text-sm">
                  <Mail className="h-4 w-4 mr-3 text-muted-foreground" />
                  <span>{user.email}</span>
                </div>
                {user.phone && (
                  <div className="flex items-center text-sm">
                    <Phone className="h-4 w-4 mr-3 text-muted-foreground" />
                    <span>{formatPhoneNumber(user.phone)}</span>
                  </div>
                )}
                {user.createdAt && (
                  <div className="flex items-center text-sm">
                    <Clock className="h-4 w-4 mr-3 text-muted-foreground" />
                    <span>
                      가입일: {formatDate(user.createdAt, { format: "short" })}
                    </span>
                  </div>
                )}
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Profile & Settings */}
        <div className="lg:col-span-2 space-y-6">
          {/* Profile Edit */}
          <form onSubmit={handleSubmitProfile(onSubmitProfile)}>
            <Card>
              <CardHeader className="flex flex-row items-center justify-between">
                <CardTitle className="text-lg flex items-center">
                  <User className="h-5 w-5 mr-2" />
                  기본 정보
                </CardTitle>
                <Button type="submit" isLoading={isSubmitting}>
                  <Save className="h-4 w-4 mr-2" />
                  저장
                </Button>
              </CardHeader>
              <CardContent className="space-y-4">
                <FeatureUnavailable
                  feature="프로필 수정"
                  detail="서버에 본인 프로필 수정 API가 아직 없어 변경 내용은 저장되지 않습니다. 비밀번호 변경은 아래에서 정상 동작합니다."
                />
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <Input
                    label="이름"
                    required
                    error={profileErrors.name?.message}
                    {...registerProfile("name")}
                  />
                  <Input
                    label="연락처"
                    placeholder="'-' 없이 입력"
                    {...registerProfile("phone")}
                  />
                  <div className="md:col-span-2">
                    <Input
                      label="이메일"
                      type="email"
                      required
                      disabled
                      error={profileErrors.email?.message}
                      {...registerProfile("email")}
                    />
                    <p className="text-sm text-muted-foreground mt-1">
                      이메일 변경은 관리자에게 문의하세요.
                    </p>
                  </div>
                </div>
              </CardContent>
            </Card>
          </form>

          {/* Password Change */}
          <form onSubmit={handleSubmitPassword(onSubmitPassword)}>
            <Card>
              <CardHeader>
                <CardTitle className="text-lg flex items-center">
                  <Lock className="h-5 w-5 mr-2" />
                  비밀번호 변경
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  <div className="relative">
                    <Input
                      type={showCurrentPassword ? "text" : "password"}
                      label="현재 비밀번호"
                      required
                      error={passwordErrors.currentPassword?.message}
                      {...registerPassword("currentPassword")}
                    />
                    <button
                      type="button"
                      className="absolute right-3 top-9 text-muted-foreground hover:text-foreground"
                      onClick={() => setShowCurrentPassword(!showCurrentPassword)}
                    >
                      {showCurrentPassword ? (
                        <EyeOff className="h-4 w-4" />
                      ) : (
                        <Eye className="h-4 w-4" />
                      )}
                    </button>
                  </div>
                  <div className="relative">
                    <Input
                      type={showNewPassword ? "text" : "password"}
                      label="새 비밀번호"
                      required
                      error={passwordErrors.newPassword?.message}
                      {...registerPassword("newPassword")}
                    />
                    <button
                      type="button"
                      className="absolute right-3 top-9 text-muted-foreground hover:text-foreground"
                      onClick={() => setShowNewPassword(!showNewPassword)}
                    >
                      {showNewPassword ? (
                        <EyeOff className="h-4 w-4" />
                      ) : (
                        <Eye className="h-4 w-4" />
                      )}
                    </button>
                  </div>
                  <div className="relative">
                    <Input
                      type={showConfirmPassword ? "text" : "password"}
                      label="새 비밀번호 확인"
                      required
                      error={passwordErrors.confirmPassword?.message}
                      {...registerPassword("confirmPassword")}
                    />
                    <button
                      type="button"
                      className="absolute right-3 top-9 text-muted-foreground hover:text-foreground"
                      onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                    >
                      {showConfirmPassword ? (
                        <EyeOff className="h-4 w-4" />
                      ) : (
                        <Eye className="h-4 w-4" />
                      )}
                    </button>
                  </div>
                  <div className="text-sm text-muted-foreground">
                    <p>비밀번호 요구사항:</p>
                    <ul className="list-disc list-inside mt-1 space-y-1">
                      <li>최소 8자 이상</li>
                      <li>대문자, 소문자, 숫자 포함</li>
                    </ul>
                  </div>
                  <div className="flex justify-end">
                    <Button type="submit" isLoading={isSubmitting}>
                      비밀번호 변경
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          </form>

          {/* Notification Settings */}
          <Card>
            <CardHeader>
              <CardTitle className="text-lg flex items-center">
                <Bell className="h-5 w-5 mr-2" />
                알림 설정
              </CardTitle>
            </CardHeader>
            <CardContent>
              <FeatureUnavailable
                feature="알림 설정"
                detail="서버에 알림 설정 API가 없어 변경 내용은 저장되지 않습니다. 아래 값은 예시입니다."
                className="mb-4"
              />
              <div className="space-y-4">
                {notificationSettings.map((setting) => (
                  <div
                    key={setting.id}
                    className="flex items-center justify-between py-3 border-b last:border-0"
                  >
                    <div>
                      <p className="font-medium">{setting.label}</p>
                      <p className="text-sm text-muted-foreground">
                        {setting.description}
                      </p>
                    </div>
                    <div className="flex items-center space-x-4">
                      <label className="flex items-center space-x-2 cursor-pointer">
                        <input
                          type="checkbox"
                          className="rounded border-input"
                          checked={setting.email}
                          onChange={() =>
                            toggleNotification(setting.id, "email")
                          }
                        />
                        <Mail className="h-4 w-4 text-muted-foreground" />
                      </label>
                      <label className="flex items-center space-x-2 cursor-pointer">
                        <input
                          type="checkbox"
                          className="rounded border-input"
                          checked={setting.push}
                          onChange={() => toggleNotification(setting.id, "push")}
                        />
                        <Bell className="h-4 w-4 text-muted-foreground" />
                      </label>
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>

          {/* Login History */}
          <Card>
            <CardHeader>
              <CardTitle className="text-lg flex items-center">
                <Clock className="h-5 w-5 mr-2" />
                로그인 이력
              </CardTitle>
            </CardHeader>
            <CardContent>
              <FeatureUnavailable
                feature="로그인 이력"
                detail="서버에 로그인 이력 API가 없어 아래 표는 예시 데이터입니다."
                className="mb-4"
              />
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>일시</TableHead>
                    <TableHead>기기</TableHead>
                    <TableHead>위치</TableHead>
                    <TableHead>IP</TableHead>
                    <TableHead>상태</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {mockLoginHistory.map((login) => (
                    <TableRow key={login.id}>
                      <TableCell>
                        {formatDate(login.timestamp, { format: "time" })}
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center">
                          {login.device === "Desktop" ? (
                            <Monitor className="h-4 w-4 mr-2 text-muted-foreground" />
                          ) : (
                            <Smartphone className="h-4 w-4 mr-2 text-muted-foreground" />
                          )}
                          <span className="text-sm">
                            {login.device} / {login.browser}
                          </span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center text-sm">
                          <MapPin className="h-3 w-3 mr-1 text-muted-foreground" />
                          {login.location}
                        </div>
                      </TableCell>
                      <TableCell className="font-mono text-sm">
                        {login.ip}
                      </TableCell>
                      <TableCell>
                        {login.success ? (
                          <Badge variant="success">
                            <CheckCircle className="h-3 w-3 mr-1" />
                            성공
                          </Badge>
                        ) : (
                          <Badge variant="destructive">
                            <AlertCircle className="h-3 w-3 mr-1" />
                            실패
                          </Badge>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
              <div className="mt-4 text-center">
                <Button variant="outline" size="sm">
                  더 보기
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
