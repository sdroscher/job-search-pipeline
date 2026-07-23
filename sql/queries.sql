-- name: GetProfile :one
SELECT * FROM user_profile WHERE id = 1;

-- name: UpsertProfile :one
INSERT INTO user_profile (
  id, resume_md, cover_letter_sample, salary_min, salary_max, salary_target,
  remote_pref, location, industries, green_flags, red_flags, tech_prefs,
  writing_voice_md, profile_hash, updated_at
) VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(id) DO UPDATE SET
  resume_md = excluded.resume_md,
  cover_letter_sample = excluded.cover_letter_sample,
  salary_min = excluded.salary_min,
  salary_max = excluded.salary_max,
  salary_target = excluded.salary_target,
  remote_pref = excluded.remote_pref,
  location = excluded.location,
  industries = excluded.industries,
  green_flags = excluded.green_flags,
  red_flags = excluded.red_flags,
  tech_prefs = excluded.tech_prefs,
  writing_voice_md = excluded.writing_voice_md,
  profile_hash = excluded.profile_hash,
  updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: ListJobs :many
SELECT * FROM jobs ORDER BY
  CASE verdict WHEN 'green' THEN 0 WHEN 'yellow' THEN 1 ELSE 2 END,
  fit_score DESC,
  salary_min DESC;

-- name: GetJob :one
SELECT * FROM jobs WHERE id = ?;

-- name: CreateJob :one
INSERT INTO jobs (
  id, company, role, stage, verdict, salary, salary_min, remote,
  source, source_url, raw_jd, added, last_activity, fit_score,
  summary, positives, concerns, my_notes, company_values, networking, role_details
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateJob :one
UPDATE jobs SET
  stage         = COALESCE(sqlc.narg(stage), stage),
  verdict       = COALESCE(sqlc.narg(verdict), verdict),
  salary        = COALESCE(sqlc.narg(salary), salary),
  salary_min    = COALESCE(sqlc.narg(salary_min), salary_min),
  fit_score     = COALESCE(sqlc.narg(fit_score), fit_score),
  summary       = COALESCE(sqlc.narg(summary), summary),
  positives     = COALESCE(sqlc.narg(positives), positives),
  concerns      = COALESCE(sqlc.narg(concerns), concerns),
  my_notes       = COALESCE(sqlc.narg(my_notes), my_notes),
  company_values = COALESCE(sqlc.narg(company_values), company_values),
  networking     = COALESCE(sqlc.narg(networking), networking),
  role_details   = COALESCE(sqlc.narg(role_details), role_details),
  last_activity  = CURRENT_DATE,
  updated_at    = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteJob :exec
UPDATE jobs SET stage = 'Won''t Apply', updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: ListActivityLog :many
SELECT * FROM activity_log WHERE job_id = ? ORDER BY date DESC, id DESC;

-- name: CreateActivityEntry :one
INSERT INTO activity_log (job_id, date, action, notes)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: ListArtifacts :many
SELECT * FROM artifacts WHERE job_id = ? ORDER BY created_at DESC;

-- name: CreateArtifact :one
INSERT INTO artifacts (job_id, type, filepath, profile_hash)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: MarkArtifactsStale :exec
UPDATE artifacts SET stale = 1
WHERE profile_hash != (SELECT profile_hash FROM user_profile WHERE id = 1);

-- name: GetArtifact :one
SELECT * FROM artifacts WHERE id = ? AND job_id = ?;
