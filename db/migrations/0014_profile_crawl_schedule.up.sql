ALTER TABLE company_profile ADD COLUMN crawl_frequency TEXT NOT NULL DEFAULT 'harian'
    CHECK (crawl_frequency IN ('harian', '2-3x', 'mingguan'));
ALTER TABLE company_profile ADD COLUMN crawl_enabled BOOLEAN NOT NULL DEFAULT false;
