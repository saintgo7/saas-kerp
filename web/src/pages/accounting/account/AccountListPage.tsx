import { useEffect, useMemo, useRef, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Plus,
  Search,
  Download,
  ChevronRight,
  ChevronDown,
  Edit,
  Trash2,
  Folder,
  FolderOpen,
  FileText,
  CheckCircle,
  XCircle,
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
} from "@/components/ui";
import { ACCOUNT_TYPES } from "@/constants";
import { accountsApi } from "@/api";
import type { AccountTreeNode } from "@/api/accounts";
import { getErrorMessage } from "@/services/api";
import { toast } from "@/stores/ui";
import { notifyUnavailable } from "@/components/common";
import type { AccountType } from "@/types";

const accountTypeStyles: Record<AccountType, { variant: "default" | "secondary" | "success" | "warning" | "destructive"; label: string }> = {
  asset: { variant: "success", label: "자산" },
  liability: { variant: "warning", label: "부채" },
  equity: { variant: "secondary", label: "자본" },
  revenue: { variant: "default", label: "수익" },
  expense: { variant: "destructive", label: "비용" },
};

const unknownType = { variant: "secondary" as const, label: "기타" };

// Tree Node Component
interface TreeNodeProps {
  node: AccountTreeNode;
  expandedNodes: Set<string>;
  onToggle: (id: string) => void;
  onEdit: (id: string) => void;
  onDelete: (id: string) => void;
  searchTerm: string;
  selectedType: string;
}

function TreeNode({ node, expandedNodes, onToggle, onEdit, onDelete, searchTerm, selectedType }: TreeNodeProps) {
  const hasChildren = node.children && node.children.length > 0;
  const isExpanded = expandedNodes.has(node.id);
  const typeStyle = accountTypeStyles[node.type] ?? unknownType;

  // Filter visibility based on search and type
  const matchesSearch = !searchTerm ||
    node.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    node.code.includes(searchTerm);
  const matchesType = !selectedType || node.type === selectedType;

  // Check if any children match
  const hasMatchingChildren = (n: AccountTreeNode): boolean => {
    if (!n.children) return false;
    return n.children.some(child => {
      const childMatches = (!searchTerm || child.name.toLowerCase().includes(searchTerm.toLowerCase()) || child.code.includes(searchTerm)) &&
        (!selectedType || child.type === selectedType);
      return childMatches || hasMatchingChildren(child);
    });
  };

  const shouldShow = matchesSearch && matchesType || hasMatchingChildren(node);

  if (!shouldShow) return null;

  return (
    <div className="select-none">
      <div
        className={`flex items-center py-2 px-2 hover:bg-muted/50 rounded-md group ${
          !node.isActive ? "opacity-50" : ""
        }`}
        style={{ paddingLeft: `${Math.max(node.level - 1, 0) * 24 + 8}px` }}
      >
        {/* Expand/Collapse Button */}
        <button
          onClick={() => onToggle(node.id)}
          className="w-6 h-6 flex items-center justify-center mr-1"
          disabled={!hasChildren}
        >
          {hasChildren ? (
            isExpanded ? (
              <ChevronDown className="h-4 w-4 text-muted-foreground" />
            ) : (
              <ChevronRight className="h-4 w-4 text-muted-foreground" />
            )
          ) : (
            <span className="w-4" />
          )}
        </button>

        {/* Folder/File Icon */}
        <span className="mr-2">
          {hasChildren ? (
            isExpanded ? (
              <FolderOpen className="h-4 w-4 text-amber-500" />
            ) : (
              <Folder className="h-4 w-4 text-amber-500" />
            )
          ) : (
            <FileText className="h-4 w-4 text-muted-foreground" />
          )}
        </span>

        {/* Code */}
        <span className="font-mono text-sm text-muted-foreground w-16 shrink-0">
          {node.code}
        </span>

        {/* Name */}
        <Link
          to={`/accounting/accounts/${node.id}`}
          className="font-medium text-primary hover:underline flex-1 truncate"
        >
          {node.name}
        </Link>

        {/* Type Badge */}
        <Badge variant={typeStyle.variant} className="ml-2 shrink-0">
          {typeStyle.label}
        </Badge>

        {/* Status */}
        <span className="ml-2 shrink-0">
          {node.isActive ? (
            <CheckCircle className="h-4 w-4 text-success" />
          ) : (
            <XCircle className="h-4 w-4 text-muted-foreground" />
          )}
        </span>

        {/* Actions */}
        <div className="ml-2 opacity-0 group-hover:opacity-100 flex items-center space-x-1">
          <Button
            variant="ghost"
            size="icon"
            className="h-7 w-7"
            onClick={() => onEdit(node.id)}
          >
            <Edit className="h-3 w-3" />
          </Button>
          {!hasChildren && (
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7 text-destructive hover:text-destructive"
              onClick={() => onDelete(node.id)}
            >
              <Trash2 className="h-3 w-3" />
            </Button>
          )}
        </div>
      </div>

      {/* Children */}
      {hasChildren && isExpanded && (
        <div>
          {node.children!.map((child) => (
            <TreeNode
              key={child.id}
              node={child}
              expandedNodes={expandedNodes}
              onToggle={onToggle}
              onEdit={onEdit}
              onDelete={onDelete}
              searchTerm={searchTerm}
              selectedType={selectedType}
            />
          ))}
        </div>
      )}
    </div>
  );
}

export function AccountListPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [searchTerm, setSearchTerm] = useState("");
  const [selectedType, setSelectedType] = useState<string>("");
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set());
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [accountToDelete, setAccountToDelete] = useState<string | null>(null);
  const seededExpansion = useRef(false);

  // GET /accounts/tree — the backend fills `children` recursively.
  const {
    data: treeResponse,
    isLoading,
    error,
  } = useQuery({
    queryKey: ["accounts", "tree"],
    queryFn: () => accountsApi.tree(),
  });

  // No mock fallback: a chart of accounts that failed to load must not look real.
  const accounts = useMemo(() => treeResponse?.data ?? [], [treeResponse]);

  // Expand the roots once, when the real tree first arrives.
  useEffect(() => {
    if (seededExpansion.current || accounts.length === 0) return;
    seededExpansion.current = true;
    setExpandedNodes(new Set(accounts.map((account) => account.id)));
  }, [accounts]);

  const deleteMutation = useMutation({
    mutationFn: (id: string) => accountsApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["accounts"] });
      setDeleteModalOpen(false);
      setAccountToDelete(null);
      toast.success("삭제 완료", "계정과목이 삭제되었습니다.");
    },
    onError: (err: unknown) => {
      toast.error(
        "삭제 실패",
        getErrorMessage(err, "계정과목 삭제 중 오류가 발생했습니다.")
      );
    },
  });

  // Count accounts recursively
  const countAccounts = useMemo(() => {
    let total = 0;
    const byType: Record<string, number> = {};

    const traverse = (node: AccountTreeNode) => {
      total++;
      byType[node.type] = (byType[node.type] || 0) + 1;
      node.children?.forEach(traverse);
    };

    accounts.forEach(traverse);
    return { total, byType };
  }, [accounts]);

  const toggleNode = (id: string) => {
    setExpandedNodes((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const expandAll = () => {
    const allIds: string[] = [];
    const traverse = (nodes: AccountTreeNode[]) => {
      nodes.forEach((node) => {
        if (node.children && node.children.length > 0) {
          allIds.push(node.id);
          traverse(node.children);
        }
      });
    };
    traverse(accounts);
    setExpandedNodes(new Set(allIds));
  };

  const collapseAll = () => {
    setExpandedNodes(new Set());
  };

  const handleEdit = (id: string) => {
    navigate(`/accounting/accounts/${id}`);
  };

  const handleDelete = (id: string) => {
    setAccountToDelete(id);
    setDeleteModalOpen(true);
  };

  const confirmDelete = () => {
    if (accountToDelete) {
      deleteMutation.mutate(accountToDelete);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">계정과목 관리</h1>
          <p className="text-muted-foreground">계정과목 체계를 조회하고 관리합니다.</p>
        </div>
        <Link to="/accounting/accounts/new">
          <Button>
            <Plus className="h-4 w-4 mr-2" />
            계정과목 추가
          </Button>
        </Link>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-2 md:grid-cols-6 gap-4">
        <Card>
          <CardContent className="p-4">
            <p className="text-sm text-muted-foreground">전체</p>
            <p className="text-2xl font-bold">{countAccounts.total}개</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <p className="text-sm text-muted-foreground">자산</p>
            <p className="text-2xl font-bold text-success">{countAccounts.byType.asset || 0}개</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <p className="text-sm text-muted-foreground">부채</p>
            <p className="text-2xl font-bold text-warning">{countAccounts.byType.liability || 0}개</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <p className="text-sm text-muted-foreground">자본</p>
            <p className="text-2xl font-bold text-gray-500">{countAccounts.byType.equity || 0}개</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <p className="text-sm text-muted-foreground">수익</p>
            <p className="text-2xl font-bold text-blue-500">{countAccounts.byType.revenue || 0}개</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <p className="text-sm text-muted-foreground">비용</p>
            <p className="text-2xl font-bold text-destructive">{countAccounts.byType.expense || 0}개</p>
          </CardContent>
        </Card>
      </div>

      {/* Filters */}
      <Card>
        <CardContent className="p-4">
          <div className="flex flex-col md:flex-row gap-4">
            <div className="flex-1">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="계정과목명 또는 코드로 검색..."
                  className="pl-10"
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                />
              </div>
            </div>
            <select
              className="h-9 rounded-md border border-input bg-background px-3 text-sm"
              value={selectedType}
              onChange={(e) => setSelectedType(e.target.value)}
            >
              <option value="">전체 유형</option>
              {ACCOUNT_TYPES.map((type) => (
                <option key={type.value} value={type.value}>
                  {type.label}
                </option>
              ))}
            </select>
            <div className="flex items-center space-x-2">
              <Button variant="outline" size="sm" onClick={expandAll}>
                전체 펼치기
              </Button>
              <Button variant="outline" size="sm" onClick={collapseAll}>
                전체 접기
              </Button>
            </div>
            <Button
              variant="outline"
              onClick={() => notifyUnavailable("계정과목 내보내기")}
            >
              <Download className="h-4 w-4 mr-2" />
              내보내기
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Tree View */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">계정과목 체계</CardTitle>
        </CardHeader>
        <CardContent>
          {/* Tree Header */}
          <div className="flex items-center py-2 px-2 border-b mb-2 text-sm text-muted-foreground">
            <span className="w-6 mr-1" />
            <span className="mr-2 w-4" />
            <span className="font-mono w-16 shrink-0">코드</span>
            <span className="flex-1">계정과목명</span>
            <span className="ml-2 w-12 text-center">유형</span>
            <span className="ml-2 w-8 text-center">상태</span>
            <span className="ml-2 w-16" />
          </div>

          {/* Tree Body */}
          <div className="space-y-1">
            {isLoading ? (
              <div className="px-3 py-8 text-center text-sm text-muted-foreground">
                계정과목을 불러오는 중입니다.
              </div>
            ) : error ? (
              <div className="px-3 py-8 text-center text-sm text-destructive">
                {getErrorMessage(error, "계정과목 조회에 실패했습니다.")}
              </div>
            ) : accounts.length === 0 ? (
              <div className="px-3 py-8 text-center text-sm text-muted-foreground">
                등록된 계정과목이 없습니다.
              </div>
            ) : (
              accounts.map((account) => (
                <TreeNode
                  key={account.id}
                  node={account}
                  expandedNodes={expandedNodes}
                  onToggle={toggleNode}
                  onEdit={handleEdit}
                  onDelete={handleDelete}
                  searchTerm={searchTerm}
                  selectedType={selectedType}
                />
              ))
            )}
          </div>

          {/* Info */}
          <div className="mt-4 pt-4 border-t">
            <p className="text-sm text-muted-foreground">
              * K-IFRS 기준 계정과목 체계입니다. 하위 계정은 상위 계정을 삭제하면 함께 삭제됩니다.
            </p>
          </div>
        </CardContent>
      </Card>

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={deleteModalOpen}
        onClose={() => setDeleteModalOpen(false)}
        title="계정과목 삭제"
      >
        <div className="space-y-4">
          <p className="text-muted-foreground">
            이 계정과목을 삭제하시겠습니까? 이미 사용된 계정과목은 삭제할 수 없습니다.
          </p>
          <div className="flex justify-end space-x-2">
            <Button variant="outline" onClick={() => setDeleteModalOpen(false)}>
              취소
            </Button>
            <Button
              variant="destructive"
              onClick={confirmDelete}
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
