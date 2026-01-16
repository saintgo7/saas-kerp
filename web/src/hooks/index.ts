// Auth hooks
export {
  authKeys,
  useCurrentUser,
  useLogin,
  useLogout,
  useRegister,
} from "./useAuth";

// Voucher hooks
export {
  voucherKeys,
  useVouchers,
  useVoucher,
  useCreateVoucher,
  useUpdateVoucher,
  useDeleteVoucher,
  useApproveVoucher,
  type Voucher,
  type VoucherEntry,
  type VoucherListParams,
  type CreateVoucherInput,
} from "./useVoucher";

// Invoice hooks
export {
  invoiceKeys,
  useInvoices,
  useInvoice,
  useCreateInvoice,
  useUpdateInvoice,
  useIssueInvoice,
  useCancelInvoice,
  useDeleteInvoice,
  type Invoice,
  type InvoiceItem,
  type InvoiceListParams,
  type CreateInvoiceInput,
} from "./useInvoice";

// Employee hooks
export {
  employeeKeys,
  useEmployees,
  useEmployee,
  useCreateEmployee,
  useUpdateEmployee,
  useDeleteEmployee,
  useChangeEmployeeStatus,
  type Employee,
  type EmployeeListParams,
  type CreateEmployeeInput,
} from "./useEmployee";
