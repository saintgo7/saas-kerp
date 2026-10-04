//go:build integration

package repository_test

import (
	"context"
	"os"
	"path/filepath"
	"sort"
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

const migrationsDir = "../../db/migrations/"

// TestChartOfAccountsBackfillMigration rebuilds production as it was before
// 000024 - every earlier migration, then seeds 001-003 (004 is the demo company
// and is not loaded in production) - adds companies the way registration left
// them, and applies 000024 and then 000025 on top, as production received them.
func TestChartOfAccountsBackfillMigration(t *testing.T) {
	ctx := context.Background()

	all, err := filepath.Glob(migrationsDir + "*.up.sql")
	require.NoError(t, err)
	var scripts []string
	for _, f := range all {
		if filepath.Base(f) < "000024_" {
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
	apply := func(migration, direction string) {
		t.Helper()
		body, err := os.ReadFile(migrationsDir + migration + "." + direction + ".sql")
		require.NoError(t, err)
		tx, err := sqlDB.BeginTx(ctx, nil)
		require.NoError(t, err)
		_, err = tx.ExecContext(ctx, string(body))
		if err != nil {
			_ = tx.Rollback()
		}
		require.NoError(t, err, migration+" "+direction)
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

	const chart24, tree25 = "000024_standard_chart_of_accounts", "000025_account_class_parents"

	apply(chart24, "up")
	require.Equal(t, standard, chartOf(empty), "a company without accounts gets the standard chart")
	require.Equal(t, own, chartOf(custom), "a company with its own chart is left alone")

	apply(chart24, "up") // idempotent
	require.Equal(t, standard, chartOf(empty))
	require.Equal(t, own, chartOf(custom))

	apply(chart24, "down") // must not delete anything
	require.Equal(t, standard, chartOf(empty))
	require.Equal(t, own, chartOf(custom))

	var fn int
	require.NoError(t, db.Raw(`SELECT count(*) FROM pg_proc WHERE proname = 'create_standard_accounts'`).Scan(&fn).Error)
	require.Equal(t, 1, fn, "down keeps the function")

	// --- 000025: the 2-digit groups under their 1-digit class ---

	// edited: a company registered while 000024's function was live, whose
	// user then moved 22 under 21 and deleted the class 5 account.
	edited := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO companies (id, code, name, business_number, representative) VALUES (?, 'MIGE', '회사', '333-33-33339', '대표')`, edited).Error)
	require.NoError(t, db.Exec(`SELECT create_standard_accounts(?)`, edited).Error)
	require.NoError(t, db.Exec(`UPDATE accounts SET parent_id = (SELECT id FROM accounts WHERE company_id = ? AND code = '21')
		WHERE company_id = ? AND code = '22'`, edited, edited).Error)
	require.NoError(t, db.Exec(`DELETE FROM accounts WHERE company_id = ? AND code = '5'`, edited).Error)
	// custom gets a class account and a same-coded group of its own making
	// (level 1, no path), which is not the standard row and must stay a root.
	require.NoError(t, db.Exec(`INSERT INTO accounts (company_id, code, name, account_type, account_nature, level, path)
		VALUES (?, '1', '내자산', 'asset', 'debit', 1, '1'), (?, '11', '내유동', 'asset', 'debit', 1, NULL)`, custom, custom).Error)

	require.Equal(t, []string{"1", "11", "12", "2", "21", "22", "3", "31", "32", "33", "4", "41", "42", "5", "51", "52", "53", "54"},
		rootsOf(parentsOf(t, db, empty)), "the defect: after 000024 the 13 groups are roots too")

	editedWant := standardParents(parentsOf(t, db, edited))
	editedWant["22"] = "21"
	for _, c := range []string{"51", "52", "53", "54"} {
		editedWant[c] = ""
	}
	customWant := map[string]string{"1": "", "11": "", "9999": ""}

	check := func(step string) {
		t.Helper()
		got := parentsOf(t, db, empty)
		require.Len(t, got, 109, step)
		require.Equal(t, standardParents(got), got, step+": every account under its standard parent")
		require.Equal(t, []string{"1", "2", "3", "4", "5"}, rootsOf(got), step)
		require.Zero(t, inconsistentRows(t, db, empty), step+": level/path agree with parent_id")
		require.Equal(t, editedWant, parentsOf(t, db, edited), step+": a chosen parent is kept, a missing class is not invented")
		require.Equal(t, customWant, parentsOf(t, db, custom), step+": user-created accounts are untouched")
	}
	apply(tree25, "up")
	check("000025 up")
	apply(tree25, "up")
	check("000025 up again")
	apply(tree25, "down")
	check("000025 down")

	// The replaced function builds the right tree for a new company.
	fresh := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO companies (id, code, name, business_number, representative) VALUES (?, 'MIGF', '회사', '333-33-33338', '대표')`, fresh).Error)
	require.NoError(t, db.Exec(`SELECT create_standard_accounts(?)`, fresh).Error)
	got := parentsOf(t, db, fresh)
	require.Len(t, got, 109)
	require.Equal(t, standardParents(got), got)
	require.Equal(t, []string{"1", "2", "3", "4", "5"}, rootsOf(got))
	require.Zero(t, inconsistentRows(t, db, fresh))
}

// parentsOf maps each account code of the company to its parent's code ("" for a root).
func parentsOf(t *testing.T, db *gorm.DB, companyID uuid.UUID) map[string]string {
	t.Helper()
	var rows []struct{ Code, Parent string }
	require.NoError(t, db.Raw(`SELECT a.code, coalesce(p.code, '') AS parent
		FROM accounts a LEFT JOIN accounts p ON p.id = a.parent_id WHERE a.company_id = ?`, companyID).Scan(&rows).Error)
	m := make(map[string]string, len(rows))
	for _, r := range rows {
		m[r.Code] = r.Parent
	}
	return m
}

// standardParents gives, for the codes in m, the parent the standard chart
// prescribes: none for a class (1), the class for a group (11), and the code
// without its last two digits below that (1101 -> 11, 110101 -> 1101).
func standardParents(m map[string]string) map[string]string {
	want := make(map[string]string, len(m))
	for code := range m {
		switch {
		case len(code) == 1:
			want[code] = ""
		case len(code) == 2:
			want[code] = code[:1]
		default:
			want[code] = code[:len(code)-2]
		}
	}
	return want
}

func rootsOf(m map[string]string) []string {
	var roots []string
	for code, parent := range m {
		if parent == "" {
			roots = append(roots, code)
		}
	}
	sort.Strings(roots)
	return roots
}

// inconsistentRows counts children whose level or path disagrees with their parent's.
func inconsistentRows(t *testing.T, db *gorm.DB, companyID uuid.UUID) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Raw(`SELECT count(*) FROM accounts a JOIN accounts p ON p.id = a.parent_id
		WHERE a.company_id = ? AND (a.level <> p.level + 1 OR a.path::text <> p.path::text || '.' || a.code)`, companyID).Scan(&n).Error)
	return n
}
