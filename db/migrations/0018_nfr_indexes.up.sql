-- EP-17 ST-17.3: covers two gaps not already served by existing indexes
-- (tender(status)/(submission_deadline)/(recommended_action)/(origin),
-- prospect(stage)/(owner_user_id) — see internal/repository/tender_repo.go
-- and prospect_repo.go for the exact filter/sort predicates these serve):
--
-- 1. Discovery Inbox's hottest query filters on
--    (origin = 'discovery' AND status = 'IDENTIFIED' AND reviewed_at IS NULL)
--    — a composite index matches this exactly instead of relying on
--    bitmap-AND across three separate single-column indexes.
-- 2. Every TenderRepo.List/ProspectRepo.List call ends with
--    ORDER BY created_at DESC — without an index backing that sort,
--    Postgres must sort in memory after applying filters.
CREATE INDEX ON tender(origin, status, reviewed_at);
CREATE INDEX ON tender(created_at DESC);
CREATE INDEX ON prospect(created_at DESC);
