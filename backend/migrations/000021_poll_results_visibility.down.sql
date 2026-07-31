ALTER TABLE polls DROP CONSTRAINT IF EXISTS chk_polls_results_visibility;
ALTER TABLE polls DROP COLUMN IF EXISTS results_published_at;
ALTER TABLE polls DROP COLUMN IF EXISTS results_visibility;
