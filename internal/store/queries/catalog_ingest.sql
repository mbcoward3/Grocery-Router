-- name: UpsertIngestSite :exec
INSERT INTO ingest.sites (
    id, attribution_name, registry_position, include_patterns, exclude_patterns, min_interval_ms,
    measurement_convention, active, eligibility_intended_use, eligibility_reviewed_on,
    eligibility_notes
) VALUES (
    sqlc.arg(id), sqlc.arg(attribution_name), sqlc.arg(registry_position),
    sqlc.arg(include_patterns), sqlc.arg(exclude_patterns), sqlc.narg(min_interval_ms),
    sqlc.arg(measurement_convention), sqlc.arg(active), sqlc.narg(eligibility_intended_use),
    sqlc.narg(eligibility_reviewed_on), sqlc.arg(eligibility_notes)
)
ON CONFLICT (id) DO UPDATE SET
    attribution_name = excluded.attribution_name,
    registry_position = excluded.registry_position,
    include_patterns = excluded.include_patterns,
    exclude_patterns = excluded.exclude_patterns,
    min_interval_ms = excluded.min_interval_ms,
    measurement_convention = excluded.measurement_convention,
    active = excluded.active,
    eligibility_intended_use = excluded.eligibility_intended_use,
    eligibility_reviewed_on = excluded.eligibility_reviewed_on,
    eligibility_notes = excluded.eligibility_notes,
    removed_at = NULL,
    updated_at = clock_timestamp();

-- name: DeleteAllIngestSiteHosts :exec
DELETE FROM ingest.site_hosts;

-- name: InsertIngestSiteHost :exec
INSERT INTO ingest.site_hosts (host, site_id, position)
VALUES (sqlc.arg(host), sqlc.arg(site_id), sqlc.arg(position));

-- name: DeleteIngestSiteSitemaps :exec
DELETE FROM ingest.site_sitemaps WHERE site_id = sqlc.arg(site_id);

-- name: InsertIngestSiteSitemap :exec
INSERT INTO ingest.site_sitemaps (site_id, position, url)
VALUES (sqlc.arg(site_id), sqlc.arg(position), sqlc.arg(url));

-- Deactivates a site the registry no longer lists. Its evidence, pages, and pauses stay.
-- name: RetireIngestSite :execrows
UPDATE ingest.sites SET
    active = FALSE,
    removed_at = clock_timestamp(),
    updated_at = clock_timestamp()
WHERE id = sqlc.arg(id) AND removed_at IS NULL;

-- name: ListIngestSites :many
SELECT * FROM ingest.sites ORDER BY removed_at IS NOT NULL, registry_position, id;

-- name: GetIngestSite :one
SELECT * FROM ingest.sites WHERE id = sqlc.arg(id);

-- name: ListIngestSiteHosts :many
SELECT * FROM ingest.site_hosts ORDER BY site_id, position;

-- name: ListIngestSiteSitemaps :many
SELECT * FROM ingest.site_sitemaps ORDER BY site_id, position;

-- name: GetIngestSiteByHost :one
SELECT s.*
FROM ingest.site_hosts AS h
INNER JOIN ingest.sites AS s ON h.site_id = s.id
WHERE h.host = sqlc.arg(host);

-- Sites that may be fetched now: listed, active, eligible, and not paused.
-- name: ListFetchableIngestSites :many
SELECT s.*
FROM ingest.sites AS s
WHERE
    s.active
    AND s.removed_at IS NULL
    AND s.eligibility_intended_use IS NOT NULL
    AND NOT EXISTS (
        SELECT 1 FROM ingest.site_pauses AS p
        WHERE p.site_id = s.id AND p.resumed_at IS NULL
    )
ORDER BY s.registry_position, s.id;


-- Opens a persistent pause. Returns 0 rows affected when the site is already paused; the first
-- open pause and its reason are kept.
-- name: PauseIngestSite :execrows
INSERT INTO ingest.site_pauses (site_id, reason, detail, fetch_attempt_id, run_id)
VALUES (
    sqlc.arg(site_id), sqlc.arg(reason), sqlc.arg(detail), sqlc.narg(fetch_attempt_id),
    sqlc.narg(run_id)
)
ON CONFLICT (site_id) WHERE resumed_at IS NULL DO NOTHING;

-- name: ResumeIngestSite :execrows
UPDATE ingest.site_pauses SET
    resumed_at = clock_timestamp(),
    resume_note = sqlc.arg(resume_note)::TEXT
WHERE site_id = sqlc.arg(site_id) AND resumed_at IS NULL;

-- name: GetOpenIngestSitePause :one
SELECT * FROM ingest.site_pauses WHERE site_id = sqlc.arg(site_id) AND resumed_at IS NULL;

-- name: ListOpenIngestSitePauses :many
SELECT * FROM ingest.site_pauses WHERE resumed_at IS NULL ORDER BY paused_at, id;


-- Atomically takes the host's single request lease when the lease is free (or expired) and the
-- next-allowed time has passed. Returns sql.ErrNoRows when the caller must wait; read
-- GetIngestHostRateLimit to learn how long. Safe across concurrent processes.
-- name: AcquireIngestHostSlot :one
INSERT INTO ingest.host_rate_limits AS r (host, lease_token, lease_expires_at)
VALUES (
    sqlc.arg(host), gen_random_uuid(),
    clock_timestamp() + sqlc.arg(lease_ms)::BIGINT * INTERVAL '1 millisecond'
)
ON CONFLICT (host) DO UPDATE SET
    lease_token = excluded.lease_token,
    lease_expires_at = excluded.lease_expires_at,
    updated_at = clock_timestamp()
WHERE
    r.next_allowed_at <= clock_timestamp()
    AND (r.lease_token IS NULL OR r.lease_expires_at <= clock_timestamp())
RETURNING r.lease_token, r.lease_expires_at;

-- Releases a lease and sets the next allowed request time at least interval_ms (never less than
-- 5 seconds) from now. Returns 0 rows affected when the lease was lost to expiry.
-- name: ReleaseIngestHostSlot :execrows
UPDATE ingest.host_rate_limits SET
    lease_token = NULL,
    lease_expires_at = NULL,
    next_allowed_at = greatest(
        next_allowed_at,
        clock_timestamp() + greatest(sqlc.arg(interval_ms)::BIGINT, 5000) * INTERVAL '1 millisecond'
    ),
    updated_at = clock_timestamp()
WHERE host = sqlc.arg(host) AND lease_token = sqlc.arg(lease_token)::UUID;

-- Pushes the next allowed time later (for example after a retryable failure); never earlier.
-- name: DeferIngestHost :exec
INSERT INTO ingest.host_rate_limits AS r (host, next_allowed_at)
VALUES (sqlc.arg(host), sqlc.arg(not_before))
ON CONFLICT (host) DO UPDATE SET
    next_allowed_at = greatest(r.next_allowed_at, excluded.next_allowed_at),
    updated_at = clock_timestamp();

-- name: GetIngestHostRateLimit :one
SELECT
    host,
    next_allowed_at,
    lease_expires_at,
    clock_timestamp()::TIMESTAMPTZ AS database_now
FROM ingest.host_rate_limits
WHERE host = sqlc.arg(host);

-- Stage locks (§7.3). Session-level: call on one *sql.Conn and unlock on the same connection.
-- name: TryIngestStageLock :one
SELECT pg_try_advisory_lock(sqlc.arg(lock_key)::BIGINT) AS acquired;

-- name: ReleaseIngestStageLock :one
SELECT pg_advisory_unlock(sqlc.arg(lock_key)::BIGINT) AS released;


-- name: StartIngestRun :one
INSERT INTO ingest.runs (command, site_id, parameters)
VALUES (sqlc.arg(command), sqlc.narg(site_id), sqlc.arg(parameters))
RETURNING *;

-- name: UpdateIngestRunCounters :exec
UPDATE ingest.runs SET counters = sqlc.arg(counters)
WHERE id = sqlc.arg(id) AND state = 'running';

-- name: FinishIngestRun :execrows
UPDATE ingest.runs SET
    state = sqlc.arg(state),
    counters = sqlc.arg(counters),
    error = sqlc.narg(error),
    finished_at = clock_timestamp()
WHERE id = sqlc.arg(id) AND state = 'running';

-- name: GetIngestRun :one
SELECT * FROM ingest.runs WHERE id = sqlc.arg(id);

-- name: ListRunningIngestRuns :many
SELECT * FROM ingest.runs WHERE state = 'running' ORDER BY started_at, id;


-- Records a page found in a registered sitemap. A known URL keeps its queue state; only sighting
-- metadata changes. Returns the page and whether it was newly inserted.
-- name: UpsertIngestSourcePage :one
INSERT INTO ingest.source_pages AS p (site_id, url, discovered_via, first_run_id, sitemap_lastmod)
VALUES (
    sqlc.arg(site_id), sqlc.arg(url), sqlc.arg(discovered_via), sqlc.narg(first_run_id),
    sqlc.narg(sitemap_lastmod)
)
ON CONFLICT (url) DO UPDATE SET
    last_seen_at = clock_timestamp(),
    sitemap_lastmod = coalesce(excluded.sitemap_lastmod, p.sitemap_lastmod),
    updated_at = clock_timestamp()
WHERE p.site_id = excluded.site_id
RETURNING p.id, p.state, (p.xmax = 0)::BOOLEAN AS inserted;

-- name: GetIngestSourcePage :one
SELECT * FROM ingest.source_pages WHERE id = sqlc.arg(id);

-- Next queued pages for one site whose retry time has passed, oldest first.
-- name: ListDueIngestSourcePages :many
SELECT * FROM ingest.source_pages
WHERE
    site_id = sqlc.arg(site_id)
    AND state = 'queued'
    AND next_attempt_at <= clock_timestamp()
ORDER BY next_attempt_at, id
LIMIT sqlc.arg(row_limit);

-- name: MarkIngestSourcePageFetched :exec
UPDATE ingest.source_pages SET
    state = 'fetched',
    state_reason = NULL,
    attempt_count = attempt_count + 1,
    latest_evidence_id = sqlc.arg(evidence_id),
    updated_at = clock_timestamp()
WHERE id = sqlc.arg(id);

-- A retryable failure: stays queued until next_attempt_at.
-- name: MarkIngestSourcePageRetry :exec
UPDATE ingest.source_pages SET
    state = 'queued',
    state_reason = sqlc.arg(state_reason),
    attempt_count = attempt_count + 1,
    next_attempt_at = sqlc.arg(next_attempt_at),
    updated_at = clock_timestamp()
WHERE id = sqlc.arg(id);

-- name: MarkIngestSourcePageFailed :exec
UPDATE ingest.source_pages SET
    state = 'failed',
    state_reason = sqlc.arg(state_reason),
    attempt_count = attempt_count + 1,
    updated_at = clock_timestamp()
WHERE id = sqlc.arg(id);

-- Explicitly queues a page again; a re-fetch creates new evidence and never edits the old.
-- name: RequeueIngestSourcePage :exec
UPDATE ingest.source_pages SET
    state = 'queued',
    state_reason = NULL,
    attempt_count = 0,
    next_attempt_at = clock_timestamp(),
    updated_at = clock_timestamp()
WHERE id = sqlc.arg(id);

-- name: CountIngestSourcePagesByState :many
SELECT site_id, state, count(*) AS page_count
FROM ingest.source_pages
GROUP BY site_id, state
ORDER BY site_id, state;


-- name: InsertIngestFetchAttempt :one
INSERT INTO ingest.fetch_attempts (
    run_id, site_id, source_page_id, kind, requested_url, final_url, http_status, outcome, error,
    started_at, finished_at
) VALUES (
    sqlc.narg(run_id), sqlc.arg(site_id), sqlc.narg(source_page_id), sqlc.arg(kind),
    sqlc.arg(requested_url), sqlc.narg(final_url), sqlc.narg(http_status), sqlc.arg(outcome),
    sqlc.narg(error), sqlc.arg(started_at), sqlc.arg(finished_at)
)
RETURNING id;

-- name: InsertIngestEvidence :one
INSERT INTO ingest.evidence (
    fetch_attempt_id, site_id, source_page_id, requested_url, final_url, redirect_chain,
    http_status, fetched_at, media_type, body_sha256, body_size, body_encoding, body_compressed,
    canonical_link
) VALUES (
    sqlc.arg(fetch_attempt_id), sqlc.arg(site_id), sqlc.arg(source_page_id),
    sqlc.arg(requested_url), sqlc.arg(final_url), sqlc.arg(redirect_chain), sqlc.arg(http_status),
    sqlc.arg(fetched_at), sqlc.arg(media_type), sqlc.arg(body_sha256), sqlc.arg(body_size),
    sqlc.arg(body_encoding), sqlc.arg(body_compressed), sqlc.narg(canonical_link)
)
RETURNING id;

-- name: GetIngestEvidence :one
SELECT * FROM ingest.evidence WHERE id = sqlc.arg(id);

-- Evidence in ID order for offline extraction replay; pages through with after_id.
-- name: ListIngestEvidenceForReplay :many
SELECT * FROM ingest.evidence
WHERE
    id > sqlc.arg(after_id)
    AND (sqlc.narg(site_id)::TEXT IS NULL OR site_id = sqlc.narg(site_id)::TEXT)
ORDER BY id
LIMIT sqlc.arg(row_limit);

-- name: InsertIngestJSONLDBlock :one
INSERT INTO ingest.evidence_jsonld_blocks (evidence_id, position, raw_text, parse_status, parse_error)
VALUES (
    sqlc.arg(evidence_id), sqlc.arg(position), sqlc.arg(raw_text), sqlc.arg(parse_status),
    sqlc.narg(parse_error)
)
RETURNING id;

-- name: ListIngestJSONLDBlocks :many
SELECT * FROM ingest.evidence_jsonld_blocks WHERE evidence_id = sqlc.arg(evidence_id) ORDER BY position;

-- name: InsertIngestSourceNode :one
INSERT INTO ingest.source_nodes (
    site_id, evidence_id, jsonld_block_id, location, node_id, normalization_version,
    content_digest, raw_node
) VALUES (
    sqlc.arg(site_id), sqlc.arg(evidence_id), sqlc.arg(jsonld_block_id), sqlc.arg(location),
    sqlc.narg(node_id), sqlc.arg(normalization_version), sqlc.arg(content_digest),
    sqlc.arg(raw_node)
)
RETURNING id;

-- name: ListIngestSourceNodesForEvidence :many
SELECT * FROM ingest.source_nodes WHERE evidence_id = sqlc.arg(evidence_id) ORDER BY jsonld_block_id, location;

-- name: CountIngestSourceNodesBySite :many
SELECT
    site_id,
    count(*) AS node_count,
    count(DISTINCT content_digest) AS distinct_digest_count
FROM ingest.source_nodes
GROUP BY site_id
ORDER BY site_id;

-- Fetched pages whose latest evidence holds no Recipe source node (the no_recipe outcome).
-- name: CountIngestNoRecipePagesBySite :many
SELECT p.site_id, count(*) AS page_count
FROM ingest.source_pages AS p
WHERE
    p.state = 'fetched'
    AND NOT EXISTS (
        SELECT 1 FROM ingest.source_nodes AS n
        WHERE n.evidence_id = p.latest_evidence_id
    )
GROUP BY p.site_id
ORDER BY p.site_id;
