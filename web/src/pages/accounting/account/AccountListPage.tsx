import { useState, useMemo } from "react";
import { Link } from "react-router-dom";
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
import type { AccountType } from "@/types";

// Account tree node interface
interface AccountNode {
  id: string;
  code: string;
  name: string;
  type: AccountType;
  parentId?: string;
  level: number;
  isActive: boolean;
  description?: string;
  children?: AccountNode[];
}

// Mock data - hierarchical account structure
const mockAccounts: AccountNode[] = [
  {
    id: "1",
    code: "1",
    name: "자산",
    type: "asset",
    level: 1,
    isActive: true,
    children: [
      {
        id: "11",
        code: "11",
        name: "유동자산",
        type: "asset",
        parentId: "1",
        level: 2,
        isActive: true,
        children: [
          { id: "111", code: "111", name: "현금및현금성자산", type: "asset", parentId: "11", level: 3, isActive: true },
          { id: "112", code: "112", name: "단기금융상품", type: "asset", parentId: "11", level: 3, isActive: true },
          { id: "113", code: "113", name: "매출채권", type: "asset", parentId: "11", level: 3, isActive: true },
          { id: "114", code: "114", name: "재고자산", type: "asset", parentId: "11", level: 3, isActive: true },
          { id: "115", code: "115", name: "선급금", type: "asset", parentId: "11", level: 3, isActive: true },
          { id: "116", code: "116", name: "선급비용", type: "asset", parentId: "11", level: 3, isActive: false },
        ],
      },
      {
        id: "12",
        code: "12",
        name: "비유동자산",
        type: "asset",
        parentId: "1",
        level: 2,
        isActive: true,
        children: [
          { id: "121", code: "121", name: "장기금융상품", type: "asset", parentId: "12", level: 3, isActive: true },
          { id: "122", code: "122", name: "유형자산", type: "asset", parentId: "12", level: 3, isActive: true },
          { id: "123", code: "123", name: "무형자산", type: "asset", parentId: "12", level: 3, isActive: true },
        ],
      },
    ],
  },
  {
    id: "2",
    code: "2",
    name: "부채",
    type: "liability",
    level: 1,
    isActive: true,
    children: [
      {
        id: "21",
        code: "21",
        name: "유동부채",
        type: "liability",
        parentId: "2",
        level: 2,
        isActive: true,
        children: [
          { id: "211", code: "211", name: "매입채무", type: "liability", parentId: "21", level: 3, isActive: true },
          { id: "212", code: "212", name: "단기차입금", type: "liability", parentId: "21", level: 3, isActive: true },
          { id: "213", code: "213", name: "미지급금", type: "liability", parentId: "21", level: 3, isActive: true },
          { id: "214", code: "214", name: "예수금", type: "liability", parentId: "21", level: 3, isActive: true },
          { id: "215", code: "215", name: "미지급비용", type: "liability", parentId: "21", level: 3, isActive: true },
        ],
      },
      {
        id: "22",
        code: "22",
        name: "비유동부채",
        type: "liability",
        parentId: "2",
        level: 2,
        isActive: true,
        children: [
          { id: "221", code: "221", name: "장기차입금", type: "liability", parentId: "22", level: 3, isActive: true },
          { id: "222", code: "222", name: "퇴직급여충당부채", type: "liability", parentId: "22", level: 3, isActive: true },
        ],
      },
    ],
  },
  {
    id: "3",
    code: "3",
    name: "자본",
    type: "equity",
    level: 1,
    isActive: true,
    children: [
      { id: "31", code: "31", name: "자본금", type: "equity", parentId: "3", level: 2, isActive: true },
      { id: "32", code: "32", name: "자본잉여금", type: "equity", parentId: "3", level: 2, isActive: true },
      { id: "33", code: "33", name: "이익잉여금", type: "equity", parentId: "3", level: 2, isActive: true },
    ],
  },
  {
    id: "4",
    code: "4",
    name: "수익",
    type: "revenue",
    level: 1,
    isActive: true,
    children: [
      {
        id: "41",
        code: "41",
        name: "매출",
        type: "revenue",
        parentId: "4",
        level: 2,
        isActive: true,
        children: [
          { id: "411", code: "411", name: "상품매출", type: "revenue", parentId: "41", level: 3, isActive: true },
          { id: "412", code: "412", name: "제품매출", type: "revenue", parentId: "41", level: 3, isActive: true },
          { id: "413", code: "413", name: "서비스매출", type: "revenue", parentId: "41", level: 3, isActive: true },
        ],
      },
      {
        id: "42",
        code: "42",
        name: "영업외수익",
        type: "revenue",
        parentId: "4",
        level: 2,
        isActive: true,
        children: [
          { id: "421", code: "421", name: "이자수익", type: "revenue", parentId: "42", level: 3, isActive: true },
          { id: "422", code: "422", name: "배당금수익", type: "revenue", parentId: "42", level: 3, isActive: true },
          { id: "423", code: "423", name: "잡이익", type: "revenue", parentId: "42", level: 3, isActive: true },
        ],
      },
    ],
  },
  {
    id: "5",
    code: "5",
    name: "비용",
    type: "expense",
    level: 1,
    isActive: true,
    children: [
      {
        id: "51",
        code: "51",
        name: "매출원가",
        type: "expense",
        parentId: "5",
        level: 2,
        isActive: true,
        children: [
          { id: "511", code: "511", name: "상품매출원가", type: "expense", parentId: "51", level: 3, isActive: true },
          { id: "512", code: "512", name: "제품매출원가", type: "expense", parentId: "51", level: 3, isActive: true },
        ],
      },
      {
        id: "52",
        code: "52",
        name: "판매비와관리비",
        type: "expense",
        parentId: "5",
        level: 2,
        isActive: true,
        children: [
          { id: "521", code: "521", name: "급여", type: "expense", parentId: "52", level: 3, isActive: true },
          { id: "522", code: "522", name: "퇴직급여", type: "expense", parentId: "52", level: 3, isActive: true },
          { id: "523", code: "523", name: "복리후생비", type: "expense", parentId: "52", level: 3, isActive: true },
          { id: "524", code: "524", name: "임차료", type: "expense", parentId: "52", level: 3, isActive: true },
          { id: "525", code: "525", name: "접대비", type: "expense", parentId: "52", level: 3, isActive: true },
          { id: "526", code: "526", name: "감가상각비", type: "expense", parentId: "52", level: 3, isActive: true },
          { id: "527", code: "527", name: "통신비", type: "expense", parentId: "52", level: 3, isActive: true },
          { id: "528", code: "528", name: "소모품비", type: "expense", parentId: "52", level: 3, isActive: true },
        ],
      },
      {
        id: "53",
        code: "53",
        name: "영업외비용",
        type: "expense",
        parentId: "5",
        level: 2,
        isActive: true,
        children: [
          { id: "531", code: "531", name: "이자비용", type: "expense", parentId: "53", level: 3, isActive: true },
          { id: "532", code: "532", name: "잡손실", type: "expense", parentId: "53", level: 3, isActive: true },
        ],
      },
    ],
  },
];

const accountTypeStyles: Record<AccountType, { variant: "default" | "secondary" | "success" | "warning" | "destructive"; label: string }> = {
  asset: { variant: "success", label: "자산" },
  liability: { variant: "warning", label: "부채" },
  equity: { variant: "secondary", label: "자본" },
  revenue: { variant: "default", label: "수익" },
  expense: { variant: "destructive", label: "비용" },
};

// Tree Node Component
interface TreeNodeProps {
  node: AccountNode;
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

  // Filter visibility based on search and type
  const matchesSearch = !searchTerm ||
    node.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    node.code.includes(searchTerm);
  const matchesType = !selectedType || node.type === selectedType;

  // Check if any children match
  const hasMatchingChildren = (n: AccountNode): boolean => {
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
        style={{ paddingLeft: `${(node.level - 1) * 24 + 8}px` }}
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
        <Badge variant={accountTypeStyles[node.type].variant} className="ml-2 shrink-0">
          {accountTypeStyles[node.type].label}
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
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedType, setSelectedType] = useState<string>("");
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set(["1", "2", "3", "4", "5"]));
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [accountToDelete, setAccountToDelete] = useState<string | null>(null);

  // Count accounts recursively
  const countAccounts = useMemo(() => {
    const count = (nodes: AccountNode[]): { total: number; byType: Record<string, number> } => {
      let total = 0;
      const byType: Record<string, number> = {};

      const traverse = (node: AccountNode) => {
        total++;
        byType[node.type] = (byType[node.type] || 0) + 1;
        node.children?.forEach(traverse);
      };

      nodes.forEach(traverse);
      return { total, byType };
    };
    return count(mockAccounts);
  }, []);

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
    const traverse = (nodes: AccountNode[]) => {
      nodes.forEach((node) => {
        if (node.children && node.children.length > 0) {
          allIds.push(node.id);
          traverse(node.children);
        }
      });
    };
    traverse(mockAccounts);
    setExpandedNodes(new Set(allIds));
  };

  const collapseAll = () => {
    setExpandedNodes(new Set());
  };

  const handleEdit = (id: string) => {
    window.location.href = `/accounting/accounts/${id}`;
  };

  const handleDelete = (id: string) => {
    setAccountToDelete(id);
    setDeleteModalOpen(true);
  };

  const confirmDelete = () => {
    // TODO: API call to delete account
    console.log("Deleting account:", accountToDelete);
    setDeleteModalOpen(false);
    setAccountToDelete(null);
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
            <Button variant="outline">
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
            {mockAccounts.map((account) => (
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
            ))}
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
            <Button variant="destructive" onClick={confirmDelete}>
              삭제
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
