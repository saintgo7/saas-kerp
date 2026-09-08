-- Restore the 000004 column names on partners.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema = current_schema() AND table_name = 'partners'
                 AND column_name = 'name') THEN
        ALTER TABLE partners RENAME COLUMN name TO partner_name;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema = current_schema() AND table_name = 'partners'
                 AND column_name = 'name_en') THEN
        ALTER TABLE partners RENAME COLUMN name_en TO partner_name_en;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema = current_schema() AND table_name = 'partners'
                 AND column_name = 'payment_term_days') THEN
        ALTER TABLE partners RENAME COLUMN payment_term_days TO payment_terms;
    END IF;
END
$$;
