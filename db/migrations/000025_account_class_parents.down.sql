-- Intentionally a no-op.
--
-- The up migration fixes a defect: it links the 13 standard 2-digit accounts
-- to their class and corrects create_standard_accounts() to do the same. An
-- exact inverse would have to unlink exactly the rows the up linked, but the
-- up records nothing that tells them apart from rows linked by the corrected
-- function, or by a user, afterwards - and unlinking them only brings back the
-- 18-root tree. Code that predates 000025 runs correctly on the repaired data:
-- the old function only ever set parent_id where it was NULL.
SELECT 1;
