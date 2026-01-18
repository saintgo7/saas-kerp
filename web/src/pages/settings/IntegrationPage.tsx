import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  Link2,
  Plus,
  Settings,
  CheckCircle2,
  XCircle,
  AlertCircle,
  RefreshCw,
  Key,
  Eye,
  EyeOff,
  Copy,
  Trash2,
  FileText,
  Shield,
  Building2,
  Clock,
  Zap,
} from "lucide-react";
import {
  Button,
  Input,
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  Badge,
  Modal,
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from "@/components/ui";
import { formatDate } from "@/lib/utils";
import { toast } from "@/stores/ui";

// Validation schemas
const hometaxSchema = z.object({
  certType: z.enum(["personal", "business"]),
  certId: z.string().min(1, "인증서 ID를 입력하세요"),
  certPassword: z.string().min(1, "인증서 비밀번호를 입력하세요"),
});

const ediSchema = z.object({
  institutionType: z.enum(["nps", "nhis", "ei", "comwel"]),
  userId: z.string().min(1, "사용자 ID를 입력하세요"),
  password: z.string().min(1, "비밀번호를 입력하세요"),
});

const apiKeySchema = z.object({
  name: z.string().min(1, "API 키 이름을 입력하세요"),
  description: z.string().optional(),
});

type HometaxFormData = z.infer<typeof hometaxSchema>;
type EdiFormData = z.infer<typeof ediSchema>;
type ApiKeyFormData = z.infer<typeof apiKeySchema>;

// Integration status type
type IntegrationStatus = "connected" | "disconnected" | "error";

// Mock integration data
interface Integration {
  id: string;
  name: string;
  description: string;
  icon: typeof FileText;
  status: IntegrationStatus;
  lastSync?: string;
  errorMessage?: string;
}

interface ApiKey {
  id: string;
  name: string;
  key: string;
  description?: string;
  createdAt: string;
  lastUsed?: string;
  isActive: boolean;
}

// EDI institution types
const ediInstitutions = [
  { value: "nps", label: "국민연금공단", description: "국민연금 EDI 연동" },
  { value: "nhis", label: "국민건강보험공단", description: "건강보험/장기요양보험 EDI 연동" },
  { value: "ei", label: "고용보험", description: "고용보험 EDI 연동" },
  { value: "comwel", label: "근로복지공단", description: "산재보험 EDI 연동" },
];

const mockIntegrations: Integration[] = [
  {
    id: "hometax",
    name: "홈택스 연동",
    description: "국세청 홈택스 세금계산서 발행/조회 연동",
    icon: FileText,
    status: "connected",
    lastSync: "2024-01-15T09:30:00",
  },
  {
    id: "nps",
    name: "국민연금공단",
    description: "국민연금 EDI 신고 연동",
    icon: Building2,
    status: "connected",
    lastSync: "2024-01-14T18:00:00",
  },
  {
    id: "nhis",
    name: "국민건강보험공단",
    description: "건강보험/장기요양보험 EDI 신고 연동",
    icon: Shield,
    status: "disconnected",
  },
  {
    id: "ei",
    name: "고용보험",
    description: "고용보험 EDI 신고 연동",
    icon: Building2,
    status: "error",
    errorMessage: "인증서 만료",
  },
  {
    id: "comwel",
    name: "근로복지공단",
    description: "산재보험 EDI 신고 연동",
    icon: Shield,
    status: "disconnected",
  },
];

const mockApiKeys: ApiKey[] = [
  {
    id: "1",
    name: "모바일 앱 연동",
    key: "DEMO_API_KEY_MOBILE_APP_1234567890",
    description: "모바일 앱에서 사용하는 API 키",
    createdAt: "2024-01-01",
    lastUsed: "2024-01-15T08:45:00",
    isActive: true,
  },
  {
    id: "2",
    name: "외부 시스템 연동",
    key: "DEMO_API_KEY_EXTERNAL_SYSTEM_1234567890",
    description: "ERP 연동용 API 키",
    createdAt: "2024-01-10",
    lastUsed: "2024-01-14T14:20:00",
    isActive: true,
  },
  {
    id: "3",
    name: "테스트용",
    key: "DEMO_API_KEY_TEST_STAGING_1234567890",
    createdAt: "2024-01-05",
    isActive: false,
  },
];

// Status badge styles
const statusStyles: Record<IntegrationStatus, { variant: "success" | "secondary" | "destructive"; label: string; icon: typeof CheckCircle2 }> = {
  connected: { variant: "success", label: "연결됨", icon: CheckCircle2 },
  disconnected: { variant: "secondary", label: "미연결", icon: XCircle },
  error: { variant: "destructive", label: "오류", icon: AlertCircle },
};

export function IntegrationPage() {
  const [integrations, setIntegrations] = useState(mockIntegrations);
  const [apiKeys, setApiKeys] = useState(mockApiKeys);
  const [hometaxModalOpen, setHometaxModalOpen] = useState(false);
  const [ediModalOpen, setEdiModalOpen] = useState(false);
  const [apiKeyModalOpen, setApiKeyModalOpen] = useState(false);
  const [selectedEdiType, setSelectedEdiType] = useState<string | null>(null);
  const [showApiKey, setShowApiKey] = useState<string | null>(null);
  const [deleteApiKeyModalOpen, setDeleteApiKeyModalOpen] = useState(false);
  const [apiKeyToDelete, setApiKeyToDelete] = useState<string | null>(null);
  const [isSyncing, setIsSyncing] = useState<string | null>(null);

  // Hometax form
  const {
    register: registerHometax,
    handleSubmit: handleSubmitHometax,
    reset: resetHometax,
    formState: { errors: hometaxErrors },
  } = useForm<HometaxFormData>({
    resolver: zodResolver(hometaxSchema),
    defaultValues: {
      certType: "business",
    },
  });

  // EDI form
  const {
    register: registerEdi,
    handleSubmit: handleSubmitEdi,
    reset: resetEdi,
    formState: { errors: ediErrors },
  } = useForm<EdiFormData>({
    resolver: zodResolver(ediSchema),
  });

  // API Key form
  const {
    register: registerApiKey,
    handleSubmit: handleSubmitApiKey,
    reset: resetApiKey,
    formState: { errors: apiKeyErrors },
  } = useForm<ApiKeyFormData>({
    resolver: zodResolver(apiKeySchema),
  });

  // Handle Hometax connection
  const onSubmitHometax = async (data: HometaxFormData) => {
    try {
      // TODO: API call to connect Hometax
      console.log("Hometax data:", data);
      setIntegrations(
        integrations.map((i) =>
          i.id === "hometax"
            ? { ...i, status: "connected", lastSync: new Date().toISOString() }
            : i
        )
      );
      toast.success("연동 완료", "홈택스 연동이 완료되었습니다.");
      setHometaxModalOpen(false);
      resetHometax();
    } catch {
      toast.error("연동 실패", "홈택스 연동 중 오류가 발생했습니다.");
    }
  };

  // Handle EDI connection
  const onSubmitEdi = async (data: EdiFormData) => {
    try {
      // TODO: API call to connect EDI
      console.log("EDI data:", data);
      setIntegrations(
        integrations.map((i) =>
          i.id === selectedEdiType
            ? { ...i, status: "connected", lastSync: new Date().toISOString(), errorMessage: undefined }
            : i
        )
      );
      toast.success("연동 완료", "EDI 연동이 완료되었습니다.");
      setEdiModalOpen(false);
      resetEdi();
    } catch {
      toast.error("연동 실패", "EDI 연동 중 오류가 발생했습니다.");
    }
  };

  // Handle API Key creation
  const onSubmitApiKey = async (data: ApiKeyFormData) => {
    try {
      // Generate random API key (in real implementation, this would come from server)
      const newKey = `kerp_api_${Math.random().toString(36).substring(2)}${Date.now().toString(36)}`;
      const newApiKey: ApiKey = {
        id: Date.now().toString(),
        name: data.name,
        key: newKey,
        description: data.description,
        createdAt: new Date().toISOString().split("T")[0],
        isActive: true,
      };
      setApiKeys([...apiKeys, newApiKey]);
      toast.success("생성 완료", "새 API 키가 생성되었습니다. 키를 안전한 곳에 저장하세요.");
      setApiKeyModalOpen(false);
      resetApiKey();
      setShowApiKey(newApiKey.id);
    } catch {
      toast.error("생성 실패", "API 키 생성 중 오류가 발생했습니다.");
    }
  };

  // Handle sync
  const handleSync = async (integrationId: string) => {
    setIsSyncing(integrationId);
    try {
      // TODO: API call to sync
      await new Promise((resolve) => setTimeout(resolve, 2000));
      setIntegrations(
        integrations.map((i) =>
          i.id === integrationId
            ? { ...i, lastSync: new Date().toISOString() }
            : i
        )
      );
      toast.success("동기화 완료", "데이터 동기화가 완료되었습니다.");
    } catch {
      toast.error("동기화 실패", "데이터 동기화 중 오류가 발생했습니다.");
    } finally {
      setIsSyncing(null);
    }
  };

  // Handle disconnect
  const handleDisconnect = (integrationId: string) => {
    setIntegrations(
      integrations.map((i) =>
        i.id === integrationId
          ? { ...i, status: "disconnected", lastSync: undefined, errorMessage: undefined }
          : i
      )
    );
    toast.success("연동 해제", "연동이 해제되었습니다.");
  };

  // Handle API key copy
  const handleCopyApiKey = (key: string) => {
    navigator.clipboard.writeText(key);
    toast.success("복사 완료", "API 키가 클립보드에 복사되었습니다.");
  };

  // Handle API key toggle
  const handleToggleApiKey = (id: string) => {
    setApiKeys(
      apiKeys.map((k) => (k.id === id ? { ...k, isActive: !k.isActive } : k))
    );
    const key = apiKeys.find((k) => k.id === id);
    toast.success(
      "상태 변경",
      `API 키가 ${key?.isActive ? "비활성화" : "활성화"}되었습니다.`
    );
  };

  // Handle API key delete
  const handleDeleteApiKey = (id: string) => {
    setApiKeyToDelete(id);
    setDeleteApiKeyModalOpen(true);
  };

  const confirmDeleteApiKey = () => {
    if (apiKeyToDelete) {
      setApiKeys(apiKeys.filter((k) => k.id !== apiKeyToDelete));
      toast.success("삭제 완료", "API 키가 삭제되었습니다.");
    }
    setDeleteApiKeyModalOpen(false);
    setApiKeyToDelete(null);
  };

  // Open EDI modal
  const openEdiModal = (institutionType: string) => {
    setSelectedEdiType(institutionType);
    resetEdi({
      institutionType: institutionType as "nps" | "nhis" | "ei" | "comwel",
      userId: "",
      password: "",
    });
    setEdiModalOpen(true);
  };

  // Mask API key
  const maskApiKey = (key: string) => {
    return `${key.substring(0, 12)}${"*".repeat(20)}${key.substring(key.length - 4)}`;
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">연동 설정</h1>
          <p className="text-muted-foreground">외부 시스템 연동 및 API 키를 관리합니다.</p>
        </div>
      </div>

      {/* Hometax Integration */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <div className="flex items-center">
            <div className="w-10 h-10 rounded-lg bg-blue-100 flex items-center justify-center mr-4">
              <FileText className="h-5 w-5 text-blue-600" />
            </div>
            <div>
              <CardTitle className="text-lg">홈택스 연동</CardTitle>
              <p className="text-sm text-muted-foreground">
                국세청 홈택스 세금계산서 발행/조회
              </p>
            </div>
          </div>
          {integrations.find((i) => i.id === "hometax")?.status === "connected" ? (
            <div className="flex items-center space-x-2">
              <Badge variant="success">
                <CheckCircle2 className="h-3 w-3 mr-1" />
                연결됨
              </Badge>
              <Button
                variant="outline"
                size="sm"
                onClick={() => handleSync("hometax")}
                disabled={isSyncing === "hometax"}
              >
                {isSyncing === "hometax" ? (
                  <RefreshCw className="h-4 w-4 mr-2 animate-spin" />
                ) : (
                  <RefreshCw className="h-4 w-4 mr-2" />
                )}
                동기화
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => handleDisconnect("hometax")}
              >
                연동 해제
              </Button>
            </div>
          ) : (
            <Button onClick={() => setHometaxModalOpen(true)}>
              <Link2 className="h-4 w-4 mr-2" />
              연동하기
            </Button>
          )}
        </CardHeader>
        {integrations.find((i) => i.id === "hometax")?.lastSync && (
          <CardContent className="pt-0">
            <div className="flex items-center text-sm text-muted-foreground">
              <Clock className="h-4 w-4 mr-2" />
              마지막 동기화:{" "}
              {formatDate(
                integrations.find((i) => i.id === "hometax")!.lastSync!,
                { format: "time" }
              )}
            </div>
          </CardContent>
        )}
      </Card>

      {/* 4대보험 EDI */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg flex items-center">
            <Shield className="h-5 w-5 mr-2" />
            4대보험 EDI 연동
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {ediInstitutions.map((institution) => {
              const integration = integrations.find(
                (i) => i.id === institution.value
              );
              const StatusIcon = statusStyles[integration?.status || "disconnected"].icon;

              return (
                <div
                  key={institution.value}
                  className="p-4 border rounded-lg hover:bg-muted/50 transition-colors"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center">
                      <div className="w-10 h-10 rounded-lg bg-green-100 flex items-center justify-center mr-3">
                        <Building2 className="h-5 w-5 text-green-600" />
                      </div>
                      <div>
                        <h3 className="font-medium">{institution.label}</h3>
                        <p className="text-sm text-muted-foreground">
                          {institution.description}
                        </p>
                      </div>
                    </div>
                    <Badge
                      variant={
                        statusStyles[integration?.status || "disconnected"]
                          .variant
                      }
                    >
                      <StatusIcon className="h-3 w-3 mr-1" />
                      {
                        statusStyles[integration?.status || "disconnected"]
                          .label
                      }
                    </Badge>
                  </div>

                  {integration?.errorMessage && (
                    <div className="mt-2 p-2 bg-destructive/10 text-destructive text-sm rounded">
                      <AlertCircle className="h-4 w-4 inline mr-1" />
                      {integration.errorMessage}
                    </div>
                  )}

                  <div className="mt-4 flex items-center justify-between">
                    {integration?.lastSync && (
                      <span className="text-sm text-muted-foreground">
                        마지막 동기화:{" "}
                        {formatDate(integration.lastSync, { format: "time" })}
                      </span>
                    )}
                    <div className="flex items-center space-x-2 ml-auto">
                      {integration?.status === "connected" ? (
                        <>
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => handleSync(institution.value)}
                            disabled={isSyncing === institution.value}
                          >
                            {isSyncing === institution.value ? (
                              <RefreshCw className="h-4 w-4 animate-spin" />
                            ) : (
                              <RefreshCw className="h-4 w-4" />
                            )}
                          </Button>
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => openEdiModal(institution.value)}
                          >
                            <Settings className="h-4 w-4" />
                          </Button>
                        </>
                      ) : (
                        <Button
                          size="sm"
                          onClick={() => openEdiModal(institution.value)}
                        >
                          <Link2 className="h-4 w-4 mr-2" />
                          연동하기
                        </Button>
                      )}
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        </CardContent>
      </Card>

      {/* API Keys */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <div>
            <CardTitle className="text-lg flex items-center">
              <Key className="h-5 w-5 mr-2" />
              API 키 관리
            </CardTitle>
            <p className="text-sm text-muted-foreground mt-1">
              외부 시스템 연동을 위한 API 키를 관리합니다.
            </p>
          </div>
          <Button onClick={() => setApiKeyModalOpen(true)}>
            <Plus className="h-4 w-4 mr-2" />
            API 키 생성
          </Button>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>이름</TableHead>
                <TableHead>API 키</TableHead>
                <TableHead>생성일</TableHead>
                <TableHead>마지막 사용</TableHead>
                <TableHead>상태</TableHead>
                <TableHead className="w-24"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {apiKeys.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={6}
                    className="text-center py-8 text-muted-foreground"
                  >
                    등록된 API 키가 없습니다.
                  </TableCell>
                </TableRow>
              ) : (
                apiKeys.map((apiKey) => (
                  <TableRow key={apiKey.id}>
                    <TableCell>
                      <div>
                        <p className="font-medium">{apiKey.name}</p>
                        {apiKey.description && (
                          <p className="text-sm text-muted-foreground">
                            {apiKey.description}
                          </p>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center space-x-2">
                        <code className="text-sm bg-muted px-2 py-1 rounded font-mono">
                          {showApiKey === apiKey.id
                            ? apiKey.key
                            : maskApiKey(apiKey.key)}
                        </code>
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() =>
                            setShowApiKey(
                              showApiKey === apiKey.id ? null : apiKey.id
                            )
                          }
                        >
                          {showApiKey === apiKey.id ? (
                            <EyeOff className="h-4 w-4" />
                          ) : (
                            <Eye className="h-4 w-4" />
                          )}
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => handleCopyApiKey(apiKey.key)}
                        >
                          <Copy className="h-4 w-4" />
                        </Button>
                      </div>
                    </TableCell>
                    <TableCell>
                      {formatDate(apiKey.createdAt, { format: "short" })}
                    </TableCell>
                    <TableCell>
                      {apiKey.lastUsed
                        ? formatDate(apiKey.lastUsed, { format: "time" })
                        : "-"}
                    </TableCell>
                    <TableCell>
                      <Badge
                        variant={apiKey.isActive ? "success" : "secondary"}
                        className="cursor-pointer"
                        onClick={() => handleToggleApiKey(apiKey.id)}
                      >
                        {apiKey.isActive ? (
                          <>
                            <Zap className="h-3 w-3 mr-1" />
                            활성
                          </>
                        ) : (
                          <>
                            <XCircle className="h-3 w-3 mr-1" />
                            비활성
                          </>
                        )}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="text-destructive hover:text-destructive"
                        onClick={() => handleDeleteApiKey(apiKey.id)}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Hometax Modal */}
      <Modal
        isOpen={hometaxModalOpen}
        onClose={() => setHometaxModalOpen(false)}
        title="홈택스 연동 설정"
        size="md"
      >
        <form
          onSubmit={handleSubmitHometax(onSubmitHometax)}
          className="space-y-4"
        >
          <div>
            <label className="block text-sm font-medium mb-2">인증서 유형</label>
            <div className="flex space-x-4">
              <label className="flex items-center">
                <input
                  type="radio"
                  value="business"
                  {...registerHometax("certType")}
                  className="mr-2"
                />
                사업자 공동인증서
              </label>
              <label className="flex items-center">
                <input
                  type="radio"
                  value="personal"
                  {...registerHometax("certType")}
                  className="mr-2"
                />
                개인 공동인증서
              </label>
            </div>
          </div>
          <Input
            label="인증서 ID"
            required
            placeholder="인증서 식별 ID"
            error={hometaxErrors.certId?.message}
            {...registerHometax("certId")}
          />
          <Input
            label="인증서 비밀번호"
            type="password"
            required
            placeholder="인증서 비밀번호"
            error={hometaxErrors.certPassword?.message}
            {...registerHometax("certPassword")}
          />
          <p className="text-sm text-muted-foreground">
            <AlertCircle className="h-4 w-4 inline mr-1" />
            인증서 정보는 암호화되어 안전하게 저장됩니다.
          </p>
          <div className="flex justify-end space-x-2 pt-4">
            <Button
              type="button"
              variant="outline"
              onClick={() => setHometaxModalOpen(false)}
            >
              취소
            </Button>
            <Button type="submit">연동하기</Button>
          </div>
        </form>
      </Modal>

      {/* EDI Modal */}
      <Modal
        isOpen={ediModalOpen}
        onClose={() => setEdiModalOpen(false)}
        title={`${
          ediInstitutions.find((i) => i.value === selectedEdiType)?.label || ""
        } EDI 연동`}
        size="md"
      >
        <form onSubmit={handleSubmitEdi(onSubmitEdi)} className="space-y-4">
          <Input
            label="사용자 ID"
            required
            placeholder="EDI 사용자 ID"
            error={ediErrors.userId?.message}
            {...registerEdi("userId")}
          />
          <Input
            label="비밀번호"
            type="password"
            required
            placeholder="EDI 비밀번호"
            error={ediErrors.password?.message}
            {...registerEdi("password")}
          />
          <p className="text-sm text-muted-foreground">
            <AlertCircle className="h-4 w-4 inline mr-1" />
            EDI 계정 정보는 암호화되어 안전하게 저장됩니다.
          </p>
          <div className="flex justify-end space-x-2 pt-4">
            <Button
              type="button"
              variant="outline"
              onClick={() => setEdiModalOpen(false)}
            >
              취소
            </Button>
            <Button type="submit">연동하기</Button>
          </div>
        </form>
      </Modal>

      {/* API Key Modal */}
      <Modal
        isOpen={apiKeyModalOpen}
        onClose={() => setApiKeyModalOpen(false)}
        title="API 키 생성"
        size="md"
      >
        <form
          onSubmit={handleSubmitApiKey(onSubmitApiKey)}
          className="space-y-4"
        >
          <Input
            label="키 이름"
            required
            placeholder="예: 모바일 앱 연동"
            error={apiKeyErrors.name?.message}
            {...registerApiKey("name")}
          />
          <Input
            label="설명"
            placeholder="이 키의 용도를 입력하세요"
            {...registerApiKey("description")}
          />
          <p className="text-sm text-muted-foreground">
            <AlertCircle className="h-4 w-4 inline mr-1" />
            생성된 API 키는 한 번만 표시됩니다. 안전한 곳에 저장하세요.
          </p>
          <div className="flex justify-end space-x-2 pt-4">
            <Button
              type="button"
              variant="outline"
              onClick={() => setApiKeyModalOpen(false)}
            >
              취소
            </Button>
            <Button type="submit">생성</Button>
          </div>
        </form>
      </Modal>

      {/* Delete API Key Confirmation Modal */}
      <Modal
        isOpen={deleteApiKeyModalOpen}
        onClose={() => setDeleteApiKeyModalOpen(false)}
        title="API 키 삭제"
      >
        <div className="space-y-4">
          <p className="text-muted-foreground">
            이 API 키를 삭제하시겠습니까? 삭제된 키는 더 이상 사용할 수 없으며,
            이 키를 사용하는 서비스가 중단될 수 있습니다.
          </p>
          <div className="flex justify-end space-x-2">
            <Button
              variant="outline"
              onClick={() => setDeleteApiKeyModalOpen(false)}
            >
              취소
            </Button>
            <Button variant="destructive" onClick={confirmDeleteApiKey}>
              삭제
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
