//go:build integration

package service_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/saintgo7/saas-kerp/internal/auth"
	"github.com/saintgo7/saas-kerp/internal/config"
	"github.com/saintgo7/saas-kerp/internal/domain"
	"github.com/saintgo7/saas-kerp/internal/repository"
	"github.com/saintgo7/saas-kerp/internal/service"
)

// TestRegisterCreatesStandardChartOfAccounts registers companies through the
// real AuthService against a database built from db/migrations ONLY - no seed -
// so it also proves registration does not depend on db/seed/002 having been run.
//
// Before the fix a registered company had zero accounts and could not write a
// voucher.
func TestRegisterCreatesStandardChartOfAccounts(t *testing.T) {
	ctx := context.Background()

	migrations, err := filepath.Glob("../../db/migrations/*.up.sql")
	require.NoError(t, err)
	require.NotEmpty(t, migrations)

	pg, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithInitScripts(migrations...),
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pg.Terminate(context.Background()) })

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)

	// A pre-existing tenant with its own, user-created chart. Registration of
	// other companies must not touch it.
	otherID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO companies (id, code, name, business_number, representative) VALUES (?, ?, ?, ?, ?)`,
		otherID, "OTHER", "다른회사", "111-11-11111", "대표").Error)
	require.NoError(t, db.Exec(`INSERT INTO accounts (company_id, code, name, account_type, account_nature)
		VALUES (?, '9999', '사용자계정', 'asset', 'debit')`, otherID).Error)

	jwtService := auth.NewJWTService(&config.JWTConfig{
		Secret:          uuid.NewString() + uuid.NewString(),
		AccessTokenTTL:  time.Minute,
		RefreshTokenTTL: time.Hour,
		Issuer:          "kerp-test",
	})
	svc := service.NewAuthService(
		repository.NewUserRepository(db),
		repository.NewRefreshTokenRepository(db),
		jwtService,
		zap.NewNop(),
		repository.NewUnitOfWork(db),
	)

	register := func(company, email string) uuid.UUID {
		t.Helper()
		out, err := svc.Register(ctx, service.RegisterInput{
			CompanyName:    company,
			BusinessNumber: "222-22-" + uuid.NewString()[:5],
			Email:          email,
			Password:       uuid.NewString(), // throwaway, never reused
			Name:           "관리자",
		})
		require.NoError(t, err)
		return out.User.CompanyID
	}

	type chart struct {
		Total, Codes int64
		Has4101      bool
		Parent4101   string
	}
	chartOf := func(companyID uuid.UUID) chart {
		t.Helper()
		var c chart
		require.NoError(t, db.Raw(`
			SELECT count(*) AS total, count(DISTINCT code) AS codes, coalesce(bool_or(code = '4101'), false) AS has4101,
			       coalesce((SELECT p.code FROM accounts a JOIN accounts p ON p.id = a.parent_id
			                 WHERE a.company_id = ? AND a.code = '4101'), '') AS parent4101
			FROM accounts WHERE company_id = ?`, companyID, companyID).Scan(&c).Error)
		return c
	}

	first := register("첫회사", "first@example.test")
	want := chart{Total: 109, Codes: 109, Has4101: true, Parent4101: "41"}
	require.Equal(t, want, chartOf(first), "registered company must get the standard chart")

	second := register("둘째회사", "second@example.test")
	require.Equal(t, want, chartOf(second))
	require.Equal(t, want, chartOf(first), "registering another company must not change the first one's chart")
	require.Equal(t, chart{Total: 1, Codes: 1}, chartOf(otherID), "an existing company's own chart must be untouched")

	// The tree GET /accounts/tree returns: the five classes as roots, every
	// group (11, 21, 41, ...) under its class, and all 109 accounts reachable.
	accounts := repository.NewAccountRepository(db)
	for _, id := range []uuid.UUID{first, second} {
		roots, err := accounts.GetTree(ctx, id)
		require.NoError(t, err)
		var codes []string
		for _, r := range roots {
			codes = append(codes, r.Code)
		}
		require.Equal(t, []string{"1", "2", "3", "4", "5"}, codes, "tree roots")
		require.Equal(t, 109, countNodes(roots), "every account is in the tree")

		var misplaced []string
		require.NoError(t, db.Raw(`
			SELECT a.code FROM accounts a LEFT JOIN accounts p ON p.id = a.parent_id
			WHERE a.company_id = ? AND length(a.code) > 1
			  AND p.code IS DISTINCT FROM (CASE WHEN length(a.code) = 2 THEN left(a.code, 1)
			                                    ELSE left(a.code, length(a.code) - 2) END)`, id).Scan(&misplaced).Error)
		require.Empty(t, misplaced, "accounts not under their standard parent")
	}
}

func countNodes(accounts []domain.Account) int {
	n := len(accounts)
	for _, a := range accounts {
		n += countNodes(a.Children)
	}
	return n
}
