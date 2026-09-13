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
  useRejectVoucher,
  toVoucher,
  type Voucher,
  type VoucherEntry,
  type VoucherStatus,
  type VoucherType,
  type VoucherListParams,
  type VoucherListResult,
  type CreateVoucherInput,
} from "./useVoucher";

// NOTE: useInvoice / useEmployee / useDashboard were removed. They called
// /invoices, /employees and /api/v1/dashboard/*, none of which the Go backend
// registers (internal/router/v1.go). They had no consumers and only advertised
// endpoints that do not exist.
