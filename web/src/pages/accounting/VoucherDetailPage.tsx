import { useState } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  ArrowLeft,
  Edit,
  Trash2,
  Copy,
  RotateCcw,
  Check,
  X,
  Download,
  Printer,
  MoreHorizontal,
  Clock,
  User,
  Calendar,
  FileText,
} from "lucide-react";
import {
  Button,
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
  Modal,
  Textarea,
} from "@/components/ui";
import { formatCurrency, formatDate } from "@/lib/utils";
import { vouchersApi } from "@/api";
import { toast } from "@/stores/ui";
import type { VoucherStatus } from "@/types";

const statusStyles: Record<
  VoucherStatus,
  { variant: "default" | "secondary" | "destructive" | "success" | "warning"; label: string }
> = {
  draft: { variant: "secondary", label: "작성중" },
  pending: { variant: "warning", label: "승인대기" },
  approved: { variant: "success", label: "승인완료" },
  rejected: { variant: "destructive", label: "반려" },
};

export function VoucherDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [showApprovalModal, setShowApprovalModal] = useState(false);
  const [approvalAction, setApprovalAction] = useState<"approve" | "reject">("approve");
  const [approvalComment, setApprovalComment] = useState("");
  const [showDeleteModal, setShowDeleteModal] = useState(false);

  // Fetch voucher detail
  const {
    data: voucherResponse,
    isLoading,
    error,
  } = useQuery({
    queryKey: ["voucher", id],
    queryFn: () => vouchersApi.get(id!),
    enabled: !!id,
  });

  const voucher = voucherResponse?.data;

  // Approval mutation
  const approvalMutation = useMutation({
    mutationFn: (data: { action: "approve" | "reject"; comment?: string }) =>
      vouchersApi.approval(id!, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["voucher", id] });
      queryClient.invalidateQueries({ queryKey: ["vouchers"] });
      setShowApprovalModal(false);
      setApprovalComment("");
      toast.success(
        approvalAction === "approve" ? "승인 완료" : "반려 완료",
        approvalAction === "approve"
          ? "전표가 승인되었습니다."
          : "전표가 반려되었습니다."
      );
    },
    onError: (error: Error) => {
      toast.error("처리 실패", error.message);
    },
  });

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: () => vouchersApi.delete(id!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["vouchers"] });
      toast.success("삭제 완료", "전표가 삭제되었습니다.");
      navigate("/accounting/voucher");
    },
    onError: (error: Error) => {
      toast.error("삭제 실패", error.message);
    },
  });

  // Copy mutation
  const copyMutation = useMutation({
    mutationFn: () => vouchersApi.copy(id!),
    onSuccess: (response) => {
      toast.success("복사 완료", "전표가 복사되었습니다.");
      navigate(`/accounting/voucher/${response.data.id}`);
    },
    onError: (error: Error) => {
      toast.error("복사 실패", error.message);
    },
  });

  // Reverse mutation
  const reverseMutation = useMutation({
    mutationFn: () => vouchersApi.reverse(id!),
    onSuccess: (response) => {
      toast.success("역분개 완료", "역분개 전표가 생성되었습니다.");
      navigate(`/accounting/voucher/${response.data.id}`);
    },
    onError: (error: Error) => {
      toast.error("역분개 실패", error.message);
    },
  });

  const handleApproval = () => {
    approvalMutation.mutate({
      action: approvalAction,
      comment: approvalComment || undefined,
    });
  };

  const openApprovalModal = (action: "approve" | "reject") => {
    setApprovalAction(action);
    setShowApprovalModal(true);
  };

  // Calculate totals
  const totalDebit = voucher?.entries.reduce((sum, e) => sum + e.debitAmount, 0) || 0;
  const totalCredit = voucher?.entries.reduce((sum, e) => sum + e.creditAmount, 0) || 0;

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
      </div>
    );
  }

  if (error || !voucher) {
    return (
      <div className="flex flex-col items-center justify-center h-64 text-center">
        <p className="text-destructive mb-4">전표를 불러올 수 없습니다.</p>
        <Button variant="outline" onClick={() => navigate("/accounting/voucher")}>
          목록으로 돌아가기
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div className="flex items-center space-x-4">
          <Button variant="ghost" size="icon" onClick={() => navigate(-1)}>
            <ArrowLeft className="h-5 w-5" />
          </Button>
          <div>
            <div className="flex items-center space-x-3">
              <h1 className="text-2xl font-bold">{voucher.voucherNumber}</h1>
              <Badge variant={statusStyles[voucher.status].variant}>
                {statusStyles[voucher.status].label}
              </Badge>
            </div>
            <p className="text-muted-foreground">{voucher.description}</p>
          </div>
        </div>

        <div className="flex items-center space-x-2">
          {/* Status-dependent actions */}
          {voucher.status === "draft" && (
            <>
              <Link to={`/accounting/voucher/${id}/edit`}>
                <Button variant="outline">
                  <Edit className="h-4 w-4 mr-2" />
                  수정
                </Button>
              </Link>
              <Button
                variant="outline"
                className="text-destructive"
                onClick={() => setShowDeleteModal(true)}
              >
                <Trash2 className="h-4 w-4 mr-2" />
                삭제
              </Button>
            </>
          )}

          {voucher.status === "pending" && (
            <>
              <Button
                variant="outline"
                className="text-success"
                onClick={() => openApprovalModal("approve")}
              >
                <Check className="h-4 w-4 mr-2" />
                승인
              </Button>
              <Button
                variant="outline"
                className="text-destructive"
                onClick={() => openApprovalModal("reject")}
              >
                <X className="h-4 w-4 mr-2" />
                반려
              </Button>
            </>
          )}

          {/* Common actions */}
          <div className="relative group">
            <Button variant="outline" size="icon">
              <MoreHorizontal className="h-4 w-4" />
            </Button>
            <div className="absolute right-0 top-full mt-1 hidden group-hover:block z-10">
              <div className="bg-popover border rounded-lg shadow-lg py-1 min-w-[160px]">
                <button
                  className="flex items-center w-full px-3 py-2 text-sm hover:bg-muted"
                  onClick={() => copyMutation.mutate()}
                >
                  <Copy className="h-4 w-4 mr-2" />
                  전표 복사
                </button>
                {voucher.status === "approved" && (
                  <button
                    className="flex items-center w-full px-3 py-2 text-sm hover:bg-muted"
                    onClick={() => reverseMutation.mutate()}
                  >
                    <RotateCcw className="h-4 w-4 mr-2" />
                    역분개
                  </button>
                )}
                <button className="flex items-center w-full px-3 py-2 text-sm hover:bg-muted">
                  <Download className="h-4 w-4 mr-2" />
                  내보내기
                </button>
                <button className="flex items-center w-full px-3 py-2 text-sm hover:bg-muted">
                  <Printer className="h-4 w-4 mr-2" />
                  인쇄
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Info Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center space-x-3">
              <div className="p-2 bg-primary/10 rounded-lg">
                <Calendar className="h-5 w-5 text-primary" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">전표일자</p>
                <p className="font-semibold">{formatDate(voucher.voucherDate)}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-center space-x-3">
              <div className="p-2 bg-primary/10 rounded-lg">
                <User className="h-5 w-5 text-primary" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">작성자</p>
                <p className="font-semibold">{voucher.createdBy}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-center space-x-3">
              <div className="p-2 bg-primary/10 rounded-lg">
                <Clock className="h-5 w-5 text-primary" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">작성일시</p>
                <p className="font-semibold">{formatDate(voucher.createdAt, { format: "time" })}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-center space-x-3">
              <div className="p-2 bg-primary/10 rounded-lg">
                <FileText className="h-5 w-5 text-primary" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">분개 수</p>
                <p className="font-semibold">{voucher.entries.length}건</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Entries Table */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">분개 내역</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-12">No.</TableHead>
                <TableHead>계정과목</TableHead>
                <TableHead className="text-right">차변</TableHead>
                <TableHead className="text-right">대변</TableHead>
                <TableHead>적요</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {voucher.entries.map((entry, index) => (
                <TableRow key={entry.id}>
                  <TableCell className="text-muted-foreground">{index + 1}</TableCell>
                  <TableCell>
                    <span className="font-mono text-sm mr-2">
                      {entry.account?.code || entry.accountId}
                    </span>
                    <span>{entry.account?.name}</span>
                  </TableCell>
                  <TableCell className="text-right font-mono">
                    {entry.debitAmount > 0
                      ? formatCurrency(entry.debitAmount, { showSymbol: false })
                      : "-"}
                  </TableCell>
                  <TableCell className="text-right font-mono">
                    {entry.creditAmount > 0
                      ? formatCurrency(entry.creditAmount, { showSymbol: false })
                      : "-"}
                  </TableCell>
                  <TableCell className="text-muted-foreground">{entry.description || "-"}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>

          {/* Totals */}
          <div className="mt-4 pt-4 border-t">
            <div className="flex justify-end">
              <div className="w-full max-w-md">
                <div className="flex justify-between items-center py-2">
                  <span className="font-semibold">차변 합계</span>
                  <span className="font-mono font-semibold">
                    {formatCurrency(totalDebit, { showSymbol: false })}
                  </span>
                </div>
                <div className="flex justify-between items-center py-2">
                  <span className="font-semibold">대변 합계</span>
                  <span className="font-mono font-semibold">
                    {formatCurrency(totalCredit, { showSymbol: false })}
                  </span>
                </div>
                <div className="flex justify-between items-center py-2 border-t">
                  <span className="font-semibold">차이</span>
                  <span
                    className={`font-mono font-semibold ${
                      totalDebit !== totalCredit ? "text-destructive" : "text-success"
                    }`}
                  >
                    {formatCurrency(Math.abs(totalDebit - totalCredit), { showSymbol: false })}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Approval Info */}
      {(voucher.status === "approved" || voucher.status === "rejected") && voucher.approvedBy && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">승인 정보</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <p className="text-sm text-muted-foreground">처리자</p>
                <p className="font-semibold">{voucher.approvedBy}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">처리일시</p>
                <p className="font-semibold">{formatDate(voucher.updatedAt, { format: "time" })}</p>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Approval Modal */}
      <Modal
        isOpen={showApprovalModal}
        onClose={() => setShowApprovalModal(false)}
        title={approvalAction === "approve" ? "전표 승인" : "전표 반려"}
      >
        <div className="space-y-4">
          <p className="text-muted-foreground">
            {approvalAction === "approve"
              ? "이 전표를 승인하시겠습니까?"
              : "이 전표를 반려하시겠습니까?"}
          </p>
          <Textarea
            label="코멘트 (선택)"
            placeholder="승인/반려 사유를 입력하세요"
            value={approvalComment}
            onChange={(e) => setApprovalComment(e.target.value)}
            rows={3}
          />
          <div className="flex justify-end space-x-2">
            <Button variant="outline" onClick={() => setShowApprovalModal(false)}>
              취소
            </Button>
            <Button
              variant={approvalAction === "approve" ? "default" : "destructive"}
              onClick={handleApproval}
              isLoading={approvalMutation.isPending}
            >
              {approvalAction === "approve" ? "승인" : "반려"}
            </Button>
          </div>
        </div>
      </Modal>

      {/* Delete Modal */}
      <Modal
        isOpen={showDeleteModal}
        onClose={() => setShowDeleteModal(false)}
        title="전표 삭제"
      >
        <div className="space-y-4">
          <p className="text-muted-foreground">
            이 전표를 삭제하시겠습니까? 이 작업은 되돌릴 수 없습니다.
          </p>
          <div className="flex justify-end space-x-2">
            <Button variant="outline" onClick={() => setShowDeleteModal(false)}>
              취소
            </Button>
            <Button
              variant="destructive"
              onClick={() => deleteMutation.mutate()}
              isLoading={deleteMutation.isPending}
            >
              삭제
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
