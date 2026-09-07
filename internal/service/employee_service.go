package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
)

// ---------------------------------------------------------------------------
// Resident registration number encryption
// ---------------------------------------------------------------------------

// ResidentNumberKeyEnv names the environment variable holding the 256-bit key
// that protects employees.resident_number_enc. The value is 32 bytes encoded
// as hex (64 characters) or standard base64.
const ResidentNumberKeyEnv = "KERP_HR_FIELD_KEY"

// Resident number errors.
var (
	// ErrResidentNumberEncryptionUnavailable is returned whenever a resident
	// number would have to be written or read and no usable key is
	// configured. It exists so the failure is fail-closed: the alternative -
	// storing the number as it arrived - would put a 주민등록번호 in the
	// database in the clear, which is exactly what the BYTEA column and its
	// "_enc" suffix were meant to prevent.
	ErrResidentNumberEncryptionUnavailable = errors.New("resident number encryption is not configured")

	// ErrResidentNumberKeySize is returned when a key is configured but is not
	// 32 bytes after decoding.
	ErrResidentNumberKeySize = errors.New("resident number encryption key must be 32 bytes")

	// ErrResidentNumberCiphertext is returned when stored bytes cannot be
	// decrypted - a wrong key, or a truncated value.
	ErrResidentNumberCiphertext = errors.New("stored resident number cannot be decrypted")

	// ErrResidentNumberFormat is re-exported so handlers can map it without
	// importing the domain package for one value.
	ErrResidentNumberFormat = domain.ErrResidentNumberFormat
)

// ResidentNumberCipher encrypts and decrypts one sensitive field.
//
// The plaintext it handles is a Korean resident registration number. Callers
// must never log the argument to Encrypt or the result of Decrypt, and must
// never place either in an error message or an API response.
type ResidentNumberCipher interface {
	// Available reports whether a key is configured. A caller that is about to
	// accept a resident number checks this first so it can refuse the request
	// with a clear message instead of failing at the write.
	Available() bool
	Encrypt(plaintext string) ([]byte, error)
	Decrypt(ciphertext []byte) (string, error)
}

// unavailableCipher is the fail-closed implementation used when no key is
// configured. Every operation refuses; nothing is ever stored in the clear.
type unavailableCipher struct{}

func (unavailableCipher) Available() bool { return false }

func (unavailableCipher) Encrypt(string) ([]byte, error) {
	return nil, ErrResidentNumberEncryptionUnavailable
}

func (unavailableCipher) Decrypt([]byte) (string, error) {
	return "", ErrResidentNumberEncryptionUnavailable
}

// NewUnavailableResidentNumberCipher returns the fail-closed cipher. Tests and
// deployments without a key use it; it never stores plaintext.
func NewUnavailableResidentNumberCipher() ResidentNumberCipher {
	return unavailableCipher{}
}

// aesGCMCipher is AES-256-GCM with a fresh random nonce per value, stored as
// nonce||ciphertext||tag. GCM is authenticated, so a tampered or truncated
// column fails to decrypt rather than yielding garbage digits.
type aesGCMCipher struct {
	aead cipher.AEAD
}

// NewResidentNumberCipher builds a cipher from a raw 32-byte key.
func NewResidentNumberCipher(key []byte) (ResidentNumberCipher, error) {
	if len(key) != 32 {
		return nil, ErrResidentNumberKeySize
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &aesGCMCipher{aead: aead}, nil
}

// NewResidentNumberCipherFromEnv reads ResidentNumberKeyEnv.
//
// An unset variable is not an error: it yields the fail-closed cipher, and the
// service then refuses to accept resident numbers at all. A variable that is
// set but unusable IS an error, so a typo in a deployment is reported instead
// of silently downgrading the feature.
func NewResidentNumberCipherFromEnv() (ResidentNumberCipher, error) {
	raw := strings.TrimSpace(os.Getenv(ResidentNumberKeyEnv))
	if raw == "" {
		return NewUnavailableResidentNumberCipher(), nil
	}

	if key, err := hex.DecodeString(raw); err == nil && len(key) == 32 {
		return NewResidentNumberCipher(key)
	}
	if key, err := base64.StdEncoding.DecodeString(raw); err == nil && len(key) == 32 {
		return NewResidentNumberCipher(key)
	}

	// Deliberately says nothing about the value itself.
	return NewUnavailableResidentNumberCipher(), ErrResidentNumberKeySize
}

func (c *aesGCMCipher) Available() bool { return true }

// Encrypt seals plaintext, prefixing the nonce.
func (c *aesGCMCipher) Encrypt(plaintext string) ([]byte, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	// Seal appends to nonce, so the result is nonce||ciphertext||tag.
	return c.aead.Seal(nonce, nonce, []byte(plaintext), nil), nil
}

// Decrypt opens a value produced by Encrypt.
func (c *aesGCMCipher) Decrypt(ciphertext []byte) (string, error) {
	nonceSize := c.aead.NonceSize()
	if len(ciphertext) <= nonceSize {
		return "", ErrResidentNumberCiphertext
	}
	plaintext, err := c.aead.Open(nil, ciphertext[:nonceSize], ciphertext[nonceSize:], nil)
	if err != nil {
		return "", ErrResidentNumberCiphertext
	}
	return string(plaintext), nil
}

// ---------------------------------------------------------------------------
// Employee service
// ---------------------------------------------------------------------------

// Employee-related errors. They alias the domain errors so the repository, the
// service and the handler all compare the same values.
var (
	ErrEmployeeNotFound          = domain.ErrEmployeeNotFound
	ErrEmployeeNoExists          = domain.ErrEmployeeNoExists
	ErrEmployeeSelfManager       = domain.ErrEmployeeSelfManager
	ErrEmployeeManagerCycle      = domain.ErrEmployeeManagerCycle
	ErrEmployeeManagerNotFound   = domain.ErrEmployeeManagerNotFound
	ErrEmployeeInvalidTransition = domain.ErrEmployeeInvalidTransition
	ErrEmployeeHasHistory        = domain.ErrEmployeeHasHistory
	ErrEmployeeDepartmentUnknown = domain.ErrEmployeeDepartmentUnknown
	ErrEmployeePositionUnknown   = domain.ErrEmployeePositionUnknown

	// ErrEmployeeHasReports blocks deleting somebody other employees still
	// report to: manager_id would be left dangling at a soft-deleted row.
	ErrEmployeeHasReports = errors.New("employee still manages other employees")
)

// EmployeeFilter is re-exported from repository
type EmployeeFilter = repository.EmployeeFilter

// EmployeeStats holds headcount statistics.
type EmployeeStats struct {
	TotalCount      int64 `json:"total_count"`
	ActiveCount     int64 `json:"active_count"`
	OnLeaveCount    int64 `json:"on_leave_count"`
	ResignedCount   int64 `json:"resigned_count"`
	TerminatedCount int64 `json:"terminated_count"`
}

// EmployeeStatusChange describes a move between employment states.
type EmployeeStatusChange struct {
	Status          domain.EmployeeStatus
	ResignationDate *time.Time
	Reason          string
	ActorID         uuid.UUID
}

// EmployeeService defines the interface for employee business logic.
type EmployeeService interface {
	// Create stores a new employee. residentNumber is the plaintext 주민등록번호
	// or "" for none; it is encrypted here and never reaches the caller again.
	Create(ctx context.Context, employee *domain.Employee, residentNumber string) error

	// Update stores changes. residentNumber is nil to keep whatever is on
	// file, a pointer to "" to erase it, or a pointer to a new value.
	Update(ctx context.Context, employee *domain.Employee, residentNumber *string) error

	// Delete soft-deletes an employee that has no retained records.
	Delete(ctx context.Context, companyID, id uuid.UUID) error

	// ChangeStatus performs a validated employment status transition,
	// including resignation. It never touches payroll or insurance history.
	ChangeStatus(ctx context.Context, companyID, id uuid.UUID, change EmployeeStatusChange) error

	GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Employee, error)
	GetByEmployeeNo(ctx context.Context, companyID uuid.UUID, employeeNo string) (*domain.Employee, error)
	List(ctx context.Context, filter *EmployeeFilter) ([]domain.Employee, int64, error)

	CanDelete(ctx context.Context, companyID, id uuid.UUID) (bool, string, error)
	GetStats(ctx context.Context, companyID uuid.UUID) (*EmployeeStats, error)

	// ResidentNumberAvailable reports whether resident numbers can be accepted
	// at all, so a handler can answer 503 rather than 500.
	ResidentNumberAvailable() bool
}

// employeeService implements EmployeeService.
type employeeService struct {
	repo         repository.EmployeeRepository
	positionRepo repository.PositionRepository
	cipher       ResidentNumberCipher
}

// NewEmployeeService creates a new EmployeeService.
//
// cipher may be the fail-closed one: the service then rejects any request that
// carries a resident number instead of storing it unencrypted.
func NewEmployeeService(
	repo repository.EmployeeRepository,
	positionRepo repository.PositionRepository,
	cipher ResidentNumberCipher,
) EmployeeService {
	if cipher == nil {
		cipher = NewUnavailableResidentNumberCipher()
	}
	return &employeeService{repo: repo, positionRepo: positionRepo, cipher: cipher}
}

// ResidentNumberAvailable reports whether the cipher has a key.
func (s *employeeService) ResidentNumberAvailable() bool {
	return s.cipher.Available()
}

// Create validates and stores a new employee.
func (s *employeeService) Create(ctx context.Context, employee *domain.Employee, residentNumber string) error {
	s.normalize(employee)

	if employee.Status == "" {
		employee.Status = domain.EmployeeStatusActive
	}
	if employee.EmploymentType == "" {
		employee.EmploymentType = domain.EmploymentTypeRegular
	}
	if employee.Nationality == "" {
		employee.Nationality = "KR"
	}

	if err := employee.Validate(); err != nil {
		return err
	}

	exists, err := s.repo.ExistsByEmployeeNo(ctx, employee.CompanyID, employee.EmployeeNo, nil)
	if err != nil {
		return err
	}
	if exists {
		return ErrEmployeeNoExists
	}

	if err := s.validateOrganization(ctx, employee); err != nil {
		return err
	}

	// A brand-new row cannot be its own manager and nothing reports to it yet,
	// so only existence and tenancy of the manager are checked here.
	if err := s.validateManager(ctx, employee.CompanyID, uuid.Nil, employee.ManagerID); err != nil {
		return err
	}

	if err := s.applyResidentNumber(employee, &residentNumber); err != nil {
		return err
	}

	return s.repo.Create(ctx, employee)
}

// Update validates and stores changes to an employee.
//
// The caller is expected to have loaded the stored row and mutated it, exactly
// as the partner and project handlers do. The stored row is read again here to
// check the status transition against what is actually in the database rather
// than against whatever the client echoed back.
func (s *employeeService) Update(ctx context.Context, employee *domain.Employee, residentNumber *string) error {
	s.normalize(employee)

	stored, err := s.repo.GetByID(ctx, employee.CompanyID, employee.ID)
	if err != nil {
		return err
	}

	if err := employee.Validate(); err != nil {
		return err
	}

	if !stored.Status.CanTransitionTo(employee.Status) {
		return ErrEmployeeInvalidTransition
	}

	exists, err := s.repo.ExistsByEmployeeNo(ctx, employee.CompanyID, employee.EmployeeNo, &employee.ID)
	if err != nil {
		return err
	}
	if exists {
		return ErrEmployeeNoExists
	}

	if err := s.validateOrganization(ctx, employee); err != nil {
		return err
	}
	if err := s.validateManager(ctx, employee.CompanyID, employee.ID, employee.ManagerID); err != nil {
		return err
	}

	// Carry the stored ciphertext forward unless the caller asked for a change.
	employee.ResidentNumberEnc = stored.ResidentNumberEnc
	if err := s.applyResidentNumber(employee, residentNumber); err != nil {
		return err
	}

	return s.repo.Update(ctx, employee)
}

// Delete soft-deletes an employee, refusing when anything must be retained.
func (s *employeeService) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	canDelete, _, err := s.CanDelete(ctx, companyID, id)
	if err != nil {
		return err
	}
	if !canDelete {
		return ErrEmployeeHasHistory
	}
	return s.repo.Delete(ctx, companyID, id)
}

// ChangeStatus moves an employee between employment states.
func (s *employeeService) ChangeStatus(ctx context.Context, companyID, id uuid.UUID, change EmployeeStatusChange) error {
	if !change.Status.IsValid() {
		return domain.ErrEmployeeInvalidStatus
	}

	employee, err := s.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return err
	}

	if !employee.Status.CanTransitionTo(change.Status) {
		return ErrEmployeeInvalidTransition
	}

	if change.Status.IsSeparated() {
		// A separation is a record of a fact, so it needs its date. Payroll,
		// salary and insurance rows are untouched: they are retained for three
		// years and the schema enforces it with ON DELETE RESTRICT.
		if change.ResignationDate == nil {
			return domain.ErrEmployeeResignationDate
		}
		employee.ResignationDate = change.ResignationDate
		employee.ResignationReason = change.Reason
	} else {
		// Returning to active service clears the separation, otherwise the row
		// would claim both to be employed and to have left.
		employee.ResignationDate = nil
		employee.ResignationReason = ""
	}

	employee.Status = change.Status
	if change.ActorID != uuid.Nil {
		employee.UpdatedBy = &change.ActorID
	}

	if err := employee.Validate(); err != nil {
		return err
	}

	return s.repo.Update(ctx, employee)
}

// GetByID retrieves one employee.
func (s *employeeService) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Employee, error) {
	return s.repo.GetByID(ctx, companyID, id)
}

// GetByEmployeeNo retrieves one employee by employee number.
func (s *employeeService) GetByEmployeeNo(ctx context.Context, companyID uuid.UUID, employeeNo string) (*domain.Employee, error) {
	return s.repo.GetByEmployeeNo(ctx, companyID, employeeNo)
}

// List retrieves employees with filtering.
func (s *employeeService) List(ctx context.Context, filter *EmployeeFilter) ([]domain.Employee, int64, error) {
	filter.Page, filter.PageSize = clampHRListPage(filter.Page, filter.PageSize)
	return s.repo.List(ctx, filter)
}

// CanDelete reports whether an employee may be deleted, and why not.
func (s *employeeService) CanDelete(ctx context.Context, companyID, id uuid.UUID) (bool, string, error) {
	if _, err := s.repo.GetByID(ctx, companyID, id); err != nil {
		return false, "", err
	}

	reports, err := s.repo.CountDirectReports(ctx, companyID, id)
	if err != nil {
		return false, "", err
	}
	if reports > 0 {
		return false, "employee still manages other employees", nil
	}

	retained, err := s.repo.CountRetainedRecords(ctx, companyID, id)
	if err != nil {
		return false, "", err
	}
	if retained.Any() {
		return false, "employee has payroll, insurance or leave history; use resignation instead", nil
	}

	return true, "", nil
}

// GetStats returns the headcount by status.
func (s *employeeService) GetStats(ctx context.Context, companyID uuid.UUID) (*EmployeeStats, error) {
	counts, err := s.repo.CountByStatus(ctx, companyID)
	if err != nil {
		return nil, err
	}
	return &EmployeeStats{
		TotalCount:      counts.Total,
		ActiveCount:     counts.Active,
		OnLeaveCount:    counts.OnLeave,
		ResignedCount:   counts.Resigned,
		TerminatedCount: counts.Terminated,
	}, nil
}

// normalize trims the free-text identifiers that uniqueness depends on.
func (s *employeeService) normalize(employee *domain.Employee) {
	employee.EmployeeNo = strings.TrimSpace(employee.EmployeeNo)
	employee.Name = strings.TrimSpace(employee.Name)
	employee.Email = strings.TrimSpace(employee.Email)
	employee.WorkEmail = strings.TrimSpace(employee.WorkEmail)
}

// validateOrganization checks that the department and position belong to the
// SAME company. Without this an employee can be filed under another tenant's
// department by passing its id: the foreign keys are company-blind.
func (s *employeeService) validateOrganization(ctx context.Context, employee *domain.Employee) error {
	if employee.DepartmentID != nil {
		ok, err := s.repo.DepartmentExists(ctx, employee.CompanyID, *employee.DepartmentID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrEmployeeDepartmentUnknown
		}
	}

	if employee.PositionID != nil {
		if _, err := s.positionRepo.GetByID(ctx, employee.CompanyID, *employee.PositionID); err != nil {
			if errors.Is(err, domain.ErrPositionNotFound) {
				return ErrEmployeePositionUnknown
			}
			return err
		}
	}

	return nil
}

// validateManager rejects a self-reference and a reporting cycle.
//
// employeeID is uuid.Nil for a row that does not exist yet. The foreign key
// employees.manager_id -> employees(id) is satisfied by a row pointing at
// itself, so nothing below this layer refuses one; left in place it makes
// every walk up the reporting line loop forever. Departments had the same
// defect (see internal/service/department_service.go).
func (s *employeeService) validateManager(ctx context.Context, companyID, employeeID uuid.UUID, managerID *uuid.UUID) error {
	if managerID == nil {
		return nil
	}
	if employeeID != uuid.Nil && *managerID == employeeID {
		return ErrEmployeeSelfManager
	}

	exists, err := s.repo.Exists(ctx, companyID, *managerID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrEmployeeManagerNotFound
	}
	if employeeID == uuid.Nil {
		return nil
	}

	// Walk up from the proposed manager. Reaching this employee would close a
	// cycle; running past the depth bound means the stored data already loops.
	current := managerID
	for depth := 0; depth < domain.MaxManagerChainDepth; depth++ {
		if *current == employeeID {
			return ErrEmployeeManagerCycle
		}
		next, err := s.repo.GetManagerID(ctx, companyID, *current)
		if err != nil {
			if errors.Is(err, domain.ErrEmployeeNotFound) {
				return ErrEmployeeManagerNotFound
			}
			return err
		}
		if next == nil {
			return nil
		}
		current = next
	}

	return ErrEmployeeManagerCycle
}

// applyResidentNumber encrypts a supplied resident number, or leaves whatever
// is on file alone.
//
// It is the single place a 주민등록번호 is handled. The plaintext never leaves
// this function: it is validated, encrypted and dropped. When no key is
// configured the request is refused - storing the digits unencrypted is not an
// available fallback.
func (s *employeeService) applyResidentNumber(employee *domain.Employee, residentNumber *string) error {
	if residentNumber == nil {
		return nil
	}

	if strings.TrimSpace(*residentNumber) == "" {
		employee.ResidentNumberEnc = nil
		return nil
	}

	if !s.cipher.Available() {
		return ErrResidentNumberEncryptionUnavailable
	}

	normalized, err := domain.NormalizeResidentNumber(*residentNumber)
	if err != nil {
		return err
	}

	encrypted, err := s.cipher.Encrypt(normalized)
	if err != nil {
		return err
	}
	employee.ResidentNumberEnc = encrypted
	return nil
}
