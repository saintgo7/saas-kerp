-- ============================================================================
-- PARTNER COLUMN CONVERGENCE
-- ============================================================================
-- 000018_column_convergence aligned companies and projects with their Go/GORM
-- models but skipped partners, which carries the same class of defect:
--
--   domain.Partner.Name            -> no `column:` tag, so GORM writes "name"
--   domain.Partner.NameEn          -> ... "name_en"
--   domain.Partner.PaymentTermDays -> ... "payment_term_days"
--
-- while 000004 created the columns as partner_name / partner_name_en /
-- payment_terms. GORM builds its INSERT and UPDATE column lists from the model,
-- so every partner Create/Update failed with
--   ERROR: column "name" of relation "partners" does not exist  (SQLSTATE 42703)
-- and every SELECT silently returned an empty Name, because nothing in the
-- result set matched the field.
--
-- The renames follow the direction 000018 already established (physical schema
-- converges on the model) rather than pinning `column:` tags onto the model, so
-- there is exactly one naming convention across companies, projects and
-- partners. Indexes and constraints follow a RENAME COLUMN automatically, so
-- idx_partners_name and idx_partners_not_deleted keep working untouched.
--
-- Guarded with IF EXISTS so the migration is a no-op on a database that was
-- already converged by hand.

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema = current_schema() AND table_name = 'partners'
                 AND column_name = 'partner_name') THEN
        ALTER TABLE partners RENAME COLUMN partner_name TO name;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema = current_schema() AND table_name = 'partners'
                 AND column_name = 'partner_name_en') THEN
        ALTER TABLE partners RENAME COLUMN partner_name_en TO name_en;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema = current_schema() AND table_name = 'partners'
                 AND column_name = 'payment_terms') THEN
        ALTER TABLE partners RENAME COLUMN payment_terms TO payment_term_days;
    END IF;
END
$$;
