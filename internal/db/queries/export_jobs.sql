-- name: CreateExportJob :one
INSERT INTO export_jobs (requested_by_user_id, job_type, site_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetExportJob :one
SELECT * FROM export_jobs WHERE id = $1;

-- name: ListExportJobsForUser :many
SELECT * FROM export_jobs WHERE requested_by_user_id = $1 ORDER BY created_at DESC LIMIT $2;

-- name: MarkExportJobRunning :exec
UPDATE export_jobs SET status = 'running' WHERE id = $1;

-- name: MarkExportJobCompleted :exec
UPDATE export_jobs SET status = 'completed', result_key = $2, completed_at = now() WHERE id = $1;

-- name: MarkExportJobFailed :exec
UPDATE export_jobs SET status = 'failed', error = $2, completed_at = now() WHERE id = $1;
