package domain_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/saintgo7/saas-kerp/internal/domain"
)

// ============================================================================
// AccountType Tests
// ============================================================================

func TestAccountType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		aType    domain.AccountType
		expected bool
	}{
		{"asset is valid", domain.AccountTypeAsset, true},
		{"liability is valid", domain.AccountTypeLiability, true},
		{"equity is valid", domain.AccountTypeEquity, true},
		{"revenue is valid", domain.AccountTypeRevenue, true},
		{"expense is valid", domain.AccountTypeExpense, true},
		{"invalid type", domain.AccountType("invalid"), false},
		{"empty type", domain.AccountType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.aType.IsValid())
		})
	}
}

// ============================================================================
// AccountNature Tests
// ============================================================================

func TestAccountNature_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		nature   domain.AccountNature
		expected bool
	}{
		{"debit is valid", domain.AccountNatureDebit, true},
		{"credit is valid", domain.AccountNatureCredit, true},
		{"invalid nature", domain.AccountNature("invalid"), false},
		{"empty nature", domain.AccountNature(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.nature.IsValid())
		})
	}
}

// ============================================================================
// Account Validation Tests
// ============================================================================

func TestAccount_Validate(t *testing.T) {
	tests := []struct {
		name    string
		account *domain.Account
		wantErr error
	}{
		{
			name: "valid account",
			account: &domain.Account{
				Code:          "101",
				Name:          "Cash",
				AccountType:   domain.AccountTypeAsset,
				AccountNature: domain.AccountNatureDebit,
			},
			wantErr: nil,
		},
		{
			name: "missing code",
			account: &domain.Account{
				Code:          "",
				Name:          "Cash",
				AccountType:   domain.AccountTypeAsset,
				AccountNature: domain.AccountNatureDebit,
			},
			wantErr: domain.ErrAccountCodeRequired,
		},
		{
			name: "missing name",
			account: &domain.Account{
				Code:          "101",
				Name:          "",
				AccountType:   domain.AccountTypeAsset,
				AccountNature: domain.AccountNatureDebit,
			},
			wantErr: domain.ErrAccountNameRequired,
		},
		{
			name: "invalid account type",
			account: &domain.Account{
				Code:          "101",
				Name:          "Cash",
				AccountType:   domain.AccountType("invalid"),
				AccountNature: domain.AccountNatureDebit,
			},
			wantErr: domain.ErrInvalidAccountType,
		},
		{
			name: "invalid account nature",
			account: &domain.Account{
				Code:          "101",
				Name:          "Cash",
				AccountType:   domain.AccountTypeAsset,
				AccountNature: domain.AccountNature("invalid"),
			},
			wantErr: domain.ErrInvalidAccountNature,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.account.Validate()
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

// ============================================================================
// Account SetDefaults Tests
// ============================================================================

func TestAccount_SetDefaults(t *testing.T) {
	t.Run("asset account defaults to debit nature", func(t *testing.T) {
		a := &domain.Account{
			AccountType:   domain.AccountTypeAsset,
			AccountNature: "",
		}

		a.SetDefaults()

		assert.Equal(t, domain.AccountNatureDebit, a.AccountNature)
		assert.Equal(t, 1, a.Level)
	})

	t.Run("expense account defaults to debit nature", func(t *testing.T) {
		a := &domain.Account{
			AccountType:   domain.AccountTypeExpense,
			AccountNature: "",
		}

		a.SetDefaults()

		assert.Equal(t, domain.AccountNatureDebit, a.AccountNature)
	})

	t.Run("liability account defaults to credit nature", func(t *testing.T) {
		a := &domain.Account{
			AccountType:   domain.AccountTypeLiability,
			AccountNature: "",
		}

		a.SetDefaults()

		assert.Equal(t, domain.AccountNatureCredit, a.AccountNature)
	})

	t.Run("equity account defaults to credit nature", func(t *testing.T) {
		a := &domain.Account{
			AccountType:   domain.AccountTypeEquity,
			AccountNature: "",
		}

		a.SetDefaults()

		assert.Equal(t, domain.AccountNatureCredit, a.AccountNature)
	})

	t.Run("revenue account defaults to credit nature", func(t *testing.T) {
		a := &domain.Account{
			AccountType:   domain.AccountTypeRevenue,
			AccountNature: "",
		}

		a.SetDefaults()

		assert.Equal(t, domain.AccountNatureCredit, a.AccountNature)
	})

	t.Run("does not override existing nature", func(t *testing.T) {
		a := &domain.Account{
			AccountType:   domain.AccountTypeAsset,
			AccountNature: domain.AccountNatureCredit, // Contra-asset
		}

		a.SetDefaults()

		assert.Equal(t, domain.AccountNatureCredit, a.AccountNature)
	})

	t.Run("sets default level to 1", func(t *testing.T) {
		a := &domain.Account{
			AccountType: domain.AccountTypeAsset,
			Level:       0,
		}

		a.SetDefaults()

		assert.Equal(t, 1, a.Level)
	})

	t.Run("does not override existing level", func(t *testing.T) {
		a := &domain.Account{
			AccountType: domain.AccountTypeAsset,
			Level:       3,
		}

		a.SetDefaults()

		assert.Equal(t, 3, a.Level)
	})
}

// ============================================================================
// Account CanPost Tests
// ============================================================================

func TestAccount_CanPost(t *testing.T) {
	tests := []struct {
		name               string
		isActive           bool
		allowDirectPosting bool
		isControlAccount   bool
		expected           bool
	}{
		{
			name:               "active, allows posting, not control - can post",
			isActive:           true,
			allowDirectPosting: true,
			isControlAccount:   false,
			expected:           true,
		},
		{
			name:               "inactive account cannot post",
			isActive:           false,
			allowDirectPosting: true,
			isControlAccount:   false,
			expected:           false,
		},
		{
			name:               "direct posting disabled cannot post",
			isActive:           true,
			allowDirectPosting: false,
			isControlAccount:   false,
			expected:           false,
		},
		{
			name:               "control account cannot post",
			isActive:           true,
			allowDirectPosting: true,
			isControlAccount:   true,
			expected:           false,
		},
		{
			name:               "all restrictions - cannot post",
			isActive:           false,
			allowDirectPosting: false,
			isControlAccount:   true,
			expected:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &domain.Account{
				IsActive:           tt.isActive,
				AllowDirectPosting: tt.allowDirectPosting,
				IsControlAccount:   tt.isControlAccount,
			}

			assert.Equal(t, tt.expected, a.CanPost())
		})
	}
}

// ============================================================================
// Account Nature Tests
// ============================================================================

func TestAccount_IsDebitNature(t *testing.T) {
	t.Run("debit nature returns true", func(t *testing.T) {
		a := &domain.Account{AccountNature: domain.AccountNatureDebit}
		assert.True(t, a.IsDebitNature())
	})

	t.Run("credit nature returns false", func(t *testing.T) {
		a := &domain.Account{AccountNature: domain.AccountNatureCredit}
		assert.False(t, a.IsDebitNature())
	})
}

func TestAccount_IsCreditNature(t *testing.T) {
	t.Run("credit nature returns true", func(t *testing.T) {
		a := &domain.Account{AccountNature: domain.AccountNatureCredit}
		assert.True(t, a.IsCreditNature())
	})

	t.Run("debit nature returns false", func(t *testing.T) {
		a := &domain.Account{AccountNature: domain.AccountNatureDebit}
		assert.False(t, a.IsCreditNature())
	})
}

// ============================================================================
// Account Label Tests
// ============================================================================

func TestAccount_GetTypeLabel(t *testing.T) {
	tests := []struct {
		aType    domain.AccountType
		expected string
	}{
		{domain.AccountTypeAsset, "자산"},
		{domain.AccountTypeLiability, "부채"},
		{domain.AccountTypeEquity, "자본"},
		{domain.AccountTypeRevenue, "수익"},
		{domain.AccountTypeExpense, "비용"},
		{domain.AccountType("unknown"), ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.aType), func(t *testing.T) {
			a := &domain.Account{AccountType: tt.aType}
			assert.Equal(t, tt.expected, a.GetTypeLabel())
		})
	}
}

func TestAccount_GetNatureLabel(t *testing.T) {
	tests := []struct {
		nature   domain.AccountNature
		expected string
	}{
		{domain.AccountNatureDebit, "차변"},
		{domain.AccountNatureCredit, "대변"},
		{domain.AccountNature("unknown"), ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.nature), func(t *testing.T) {
			a := &domain.Account{AccountNature: tt.nature}
			assert.Equal(t, tt.expected, a.GetNatureLabel())
		})
	}
}

// ============================================================================
// Account Hierarchy Tests
// ============================================================================

func TestAccount_Hierarchy(t *testing.T) {
	t.Run("account with parent", func(t *testing.T) {
		parentID := uuid.New()
		a := &domain.Account{
			Code:     "10101",
			Name:     "Petty Cash",
			ParentID: &parentID,
			Level:    2,
		}

		require.NotNil(t, a.ParentID)
		assert.Equal(t, parentID, *a.ParentID)
		assert.Equal(t, 2, a.Level)
	})

	t.Run("root account without parent", func(t *testing.T) {
		a := &domain.Account{
			Code:     "101",
			Name:     "Cash",
			ParentID: nil,
			Level:    1,
		}

		assert.Nil(t, a.ParentID)
		assert.Equal(t, 1, a.Level)
	})
}

// ============================================================================
// Account Type/Nature Consistency Tests (K-IFRS)
// ============================================================================

func TestAccountTypeNatureConsistency(t *testing.T) {
	// K-IFRS standard: Asset and Expense are Debit nature
	// Liability, Equity, Revenue are Credit nature

	t.Run("asset type should be debit nature by default", func(t *testing.T) {
		a := &domain.Account{AccountType: domain.AccountTypeAsset}
		a.SetDefaults()
		assert.Equal(t, domain.AccountNatureDebit, a.AccountNature)
	})

	t.Run("expense type should be debit nature by default", func(t *testing.T) {
		a := &domain.Account{AccountType: domain.AccountTypeExpense}
		a.SetDefaults()
		assert.Equal(t, domain.AccountNatureDebit, a.AccountNature)
	})

	t.Run("liability type should be credit nature by default", func(t *testing.T) {
		a := &domain.Account{AccountType: domain.AccountTypeLiability}
		a.SetDefaults()
		assert.Equal(t, domain.AccountNatureCredit, a.AccountNature)
	})

	t.Run("equity type should be credit nature by default", func(t *testing.T) {
		a := &domain.Account{AccountType: domain.AccountTypeEquity}
		a.SetDefaults()
		assert.Equal(t, domain.AccountNatureCredit, a.AccountNature)
	})

	t.Run("revenue type should be credit nature by default", func(t *testing.T) {
		a := &domain.Account{AccountType: domain.AccountTypeRevenue}
		a.SetDefaults()
		assert.Equal(t, domain.AccountNatureCredit, a.AccountNature)
	})
}
