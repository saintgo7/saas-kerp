//go:build integration

package repository_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const chartMigration = "../../db/migrations/000024_standard_chart_of_accounts"

// TestChartOfAccountsBackfillMigration rebuilds production as it was before
// 000024 - every earlier migration, then seeds 001-003 (004 is the demo company
// and is not loaded in production) - adds companies the way registration left
// them, and applies 000024 on top.
func TestChartOfAccountsBackfillMigration(t *testing.T) {
	ctx := context.Background()

	all, err := filepath.Glob("../../db/migrations/*.up.sql")
	require.NoError(t, err)
	var scripts []string
	for _, f := range all {
		if !strings.HasPrefix(filepath.Base(f), "000024_") {
			scripts = append(scripts, f)
		}
	}
	// Init scripts run in file-name order: 0000NN_* before 00N_*.
	for _, s := range []string{"001_permissions", "002_chart_of_accounts", "003_insurance_rates"} {
		scripts = append(scripts, "../../db/seed/"+s+".sql")
	}

	pg, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithInitScripts(scripts...),
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
	sqlDB, err := db.DB()
	require.NoError(t, err)

	// Like cmd/migrate: the whole file in one transaction.
	apply := func(direction string) {
		t.Helper()
		body, err := os.ReadFile(chartMigration + "." + direction + ".sql")
		require.NoError(t, err)
		tx, err := sqlDB.BeginTx(ctx, nil)
		require.NoError(t, err)
		_, err = tx.ExecContext(ctx, string(body))
		if err != nil {
			_ = tx.Rollback()
		}
		require.NoError(t, err, direction)
		require.NoError(t, tx.Commit())
	}

	// empty: what registration produced before the fix. custom: a company that
	// already built its own chart, which the backfill must leave alone.
	empty, custom := uuid.New(), uuid.New()
	for i, id := range []uuid.UUID{empty, custom} {
		require.NoError(t, db.Exec(`INSERT INTO companies (id, code, name, business_number, representative) VALUES (?, ?, ?, ?, ?)`,
			id, "MIG"+string(rune('A'+i)), "회사", "333-33-3333"+string(rune('0'+i)), "대표").Error)
	}
	require.NoError(t, db.Exec(`INSERT INTO accounts (company_id, code, name, account_type, account_nature)
		VALUES (?, '9999', '사용자계정', 'asset', 'debit')`, custom).Error)

	type chart struct {
		Total   int64
		Has4101 bool
	}
	chartOf := func(id uuid.UUID) chart {
		t.Helper()
		var c chart
		require.NoError(t, db.Raw(`SELECT count(*) AS total, coalesce(bool_or(code = '4101'), false) AS has4101
			FROM accounts WHERE company_id = ?`, id).Scan(&c).Error)
		return c
	}
	standard, own := chart{Total: 109, Has4101: true}, chart{Total: 1}

	apply("up")
	require.Equal(t, standard, chartOf(empty), "a company without accounts gets the standard chart")
	require.Equal(t, own, chartOf(custom), "a company with its own chart is left alone")

	apply("up") // idempotent
	require.Equal(t, standard, chartOf(empty))
	require.Equal(t, own, chartOf(custom))

	apply("down") // must not delete anything
	require.Equal(t, standard, chartOf(empty))
	require.Equal(t, own, chartOf(custom))

	var fn int
	require.NoError(t, db.Raw(`SELECT count(*) FROM pg_proc WHERE proname = 'create_standard_accounts'`).Scan(&fn).Error)
	require.Equal(t, 1, fn, "down keeps the function")
}
