import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import api from "@/services/api";
import { toast } from "@/stores/ui";

// Types
export interface Employee {
  id: string;
  employeeNo: string;
  name: string;
  nameEn?: string;
  email: string;
  phone: string;
  birthDate: string;
  gender: "M" | "F";
  address?: string;

  // Employment
  employmentType: "regular" | "contract" | "parttime" | "intern";
  department: string;
  departmentName?: string;
  position: string;
  positionName?: string;
  hireDate: string;
  resignDate?: string;
  status: "active" | "leave" | "resigned";

  // Salary
  baseSalary: number;
  bankName: string;
  accountNumber: string;
  accountHolder: string;

  // Insurance (4대보험)
  nationalPensionNo?: string;
  healthInsuranceNo?: string;
  employmentInsuranceNo?: string;
  industrialAccidentNo?: string;

  note?: string;
  createdAt: string;
  updatedAt: string;
}

export interface EmployeeListParams {
  page?: number;
  limit?: number;
  department?: string;
  status?: string;
  search?: string;
}

export interface CreateEmployeeInput {
  name: string;
  nameEn?: string;
  email: string;
  phone: string;
  birthDate: string;
  gender: string;
  address?: string;
  employeeNo: string;
  employmentType: string;
  department: string;
  position: string;
  hireDate: string;
  resignDate?: string;
  status: string;
  baseSalary: number;
  bankName: string;
  accountNumber: string;
  accountHolder: string;
  nationalPensionNo?: string;
  healthInsuranceNo?: string;
  employmentInsuranceNo?: string;
  industrialAccidentNo?: string;
  note?: string;
}

// Query keys
export const employeeKeys = {
  all: ["employees"] as const,
  lists: () => [...employeeKeys.all, "list"] as const,
  list: (params: EmployeeListParams) => [...employeeKeys.lists(), params] as const,
  details: () => [...employeeKeys.all, "detail"] as const,
  detail: (id: string) => [...employeeKeys.details(), id] as const,
};

// Get employee list
export function useEmployees(params: EmployeeListParams = {}) {
  return useQuery({
    queryKey: employeeKeys.list(params),
    queryFn: async () => {
      const { data } = await api.get<{
        data: Employee[];
        total: number;
        page: number;
        limit: number;
      }>("/api/v1/employees", { params });
      return data;
    },
  });
}

// Get single employee
export function useEmployee(id: string) {
  return useQuery({
    queryKey: employeeKeys.detail(id),
    queryFn: async () => {
      const { data } = await api.get<Employee>(`/api/v1/employees/${id}`);
      return data;
    },
    enabled: !!id,
  });
}

// Create employee
export function useCreateEmployee() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: CreateEmployeeInput) => {
      const { data } = await api.post<Employee>("/api/v1/employees", input);
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: employeeKeys.lists() });
      toast.success("직원 등록 완료", "새 직원이 등록되었습니다.");
    },
    onError: (error: Error) => {
      toast.error("등록 실패", error.message || "직원 등록 중 오류가 발생했습니다.");
    },
  });
}

// Update employee
export function useUpdateEmployee() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      id,
      ...input
    }: CreateEmployeeInput & { id: string }) => {
      const { data } = await api.put<Employee>(`/api/v1/employees/${id}`, input);
      return data;
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: employeeKeys.lists() });
      queryClient.invalidateQueries({ queryKey: employeeKeys.detail(data.id) });
      toast.success("직원 수정 완료", "직원 정보가 수정되었습니다.");
    },
    onError: (error: Error) => {
      toast.error("수정 실패", error.message || "직원 정보 수정 중 오류가 발생했습니다.");
    },
  });
}

// Delete employee
export function useDeleteEmployee() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/api/v1/employees/${id}`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: employeeKeys.lists() });
      toast.success("직원 삭제 완료", "직원이 삭제되었습니다.");
    },
    onError: (error: Error) => {
      toast.error("삭제 실패", error.message || "직원 삭제 중 오류가 발생했습니다.");
    },
  });
}

// Change employee status
export function useChangeEmployeeStatus() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      id,
      status,
      resignDate,
    }: {
      id: string;
      status: "active" | "leave" | "resigned";
      resignDate?: string;
    }) => {
      const { data } = await api.patch<Employee>(`/api/v1/employees/${id}/status`, {
        status,
        resignDate,
      });
      return data;
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: employeeKeys.lists() });
      queryClient.invalidateQueries({ queryKey: employeeKeys.detail(data.id) });
      toast.success("상태 변경 완료", "직원 상태가 변경되었습니다.");
    },
    onError: (error: Error) => {
      toast.error("상태 변경 실패", error.message || "직원 상태 변경 중 오류가 발생했습니다.");
    },
  });
}
