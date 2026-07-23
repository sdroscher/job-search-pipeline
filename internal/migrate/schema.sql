-- Authoritative schema source. Embedded via //go:embed in internal/migrate/migrate.go
-- and re-executed on every startup. All CREATE TABLE statements use IF NOT EXISTS.
-- To add a new table: append here.
-- To add a new column to an existing table: add it here AND add a call to
-- addColumnIfMissing in migrate.go so existing databases are upgraded.

CREATE TABLE IF NOT EXISTS user_profile (
  id                    INTEGER PRIMARY KEY DEFAULT 1,
  resume_md             TEXT NOT NULL DEFAULT '',
  cover_letter_sample   TEXT,
  salary_min            INTEGER,
  salary_max            INTEGER,
  salary_target         INTEGER,
  remote_pref           TEXT,
  location              TEXT,
  industries            TEXT,
  green_flags           TEXT,
  red_flags             TEXT,
  tech_prefs            TEXT,
  writing_voice_md      TEXT,
  achievements_md       TEXT,
  profile_hash          TEXT NOT NULL DEFAULT '',
  updated_at            DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS jobs (
  id            TEXT PRIMARY KEY,
  company       TEXT NOT NULL,
  role          TEXT NOT NULL,
  stage         TEXT NOT NULL DEFAULT 'Evaluated',
  verdict       TEXT NOT NULL DEFAULT 'yellow',
  salary        TEXT,
  salary_min    INTEGER,
  remote        TEXT,
  source        TEXT,
  source_url    TEXT,
  raw_jd        TEXT,
  added         DATE NOT NULL,
  last_activity DATE NOT NULL,
  fit_score     INTEGER,
  summary       TEXT,
  positives     TEXT,
  concerns      TEXT,
  my_notes      TEXT,
  company_values TEXT,
  networking    TEXT,
  role_details  TEXT,
  created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS activity_log (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  job_id     TEXT NOT NULL REFERENCES jobs(id),
  date       DATE NOT NULL,
  action     TEXT NOT NULL,
  notes      TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS artifacts (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  job_id       TEXT NOT NULL REFERENCES jobs(id),
  type         TEXT NOT NULL,
  filepath     TEXT NOT NULL,
  profile_hash TEXT NOT NULL,
  stale        INTEGER NOT NULL DEFAULT 0,
  created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);
