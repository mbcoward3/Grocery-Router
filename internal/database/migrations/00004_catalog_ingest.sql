-- +goose Up
-- Operator-only acquisition data for the shared recipe catalog (CATALOG_INGEST_SPEC.md slice 2).
-- Nothing here is household data: no table carries household_id, and no HTTP request writes it.
CREATE SCHEMA ingest;

-- One row per registry entry in catalog/sites.yaml. Removed entries stay as inactive history so
-- their evidence keeps its site; removed_at records when the registry stopped listing them.
CREATE TABLE ingest.sites (
    id TEXT PRIMARY KEY CHECK (id ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND length(id) <= 64),
    attribution_name TEXT NOT NULL CHECK (trim(attribution_name) <> ''),
    registry_position INTEGER NOT NULL CHECK (registry_position >= 0),
    include_patterns JSONB NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(include_patterns) = 'array'),
    exclude_patterns JSONB NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(exclude_patterns) = 'array'),
    -- Only a stricter interval than the 5-second floor may be configured.
    min_interval_ms INTEGER CHECK (min_interval_ms IS NULL OR min_interval_ms > 5000),
    measurement_convention TEXT NOT NULL CHECK (measurement_convention IN ('us', 'metric')),
    active BOOLEAN NOT NULL,
    eligibility_intended_use TEXT CHECK (eligibility_intended_use IS NULL OR trim(eligibility_intended_use) <> ''),
    eligibility_reviewed_on DATE,
    eligibility_notes TEXT NOT NULL DEFAULT '',
    removed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    CHECK ((eligibility_intended_use IS NULL) = (eligibility_reviewed_on IS NULL)),
    CHECK (NOT active OR eligibility_intended_use IS NOT NULL),
    CHECK (removed_at IS NULL OR NOT active)
);

-- Explicit host list; the primary key makes a host belong to at most one site.
CREATE TABLE ingest.site_hosts (
    host TEXT PRIMARY KEY CHECK (
        host ~ '^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)+$'
    ),
    site_id TEXT NOT NULL REFERENCES ingest.sites (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    position INTEGER NOT NULL CHECK (position >= 0),
    UNIQUE (site_id, position)
);

CREATE TABLE ingest.site_sitemaps (
    site_id TEXT NOT NULL REFERENCES ingest.sites (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    position INTEGER NOT NULL CHECK (position >= 0),
    url TEXT NOT NULL CHECK (url ~ '^https://[a-z0-9.-]+/[^[:space:]]*$'),
    PRIMARY KEY (site_id, position),
    UNIQUE (site_id, url)
);

-- Shared by every command and request type: one in-flight request per host (the lease) and a
-- next-allowed time. Rows are created on first acquisition.
CREATE TABLE ingest.host_rate_limits (
    host TEXT PRIMARY KEY CHECK (
        host ~ '^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)+$'
    ),
    next_allowed_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    lease_token UUID,
    lease_expires_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    CHECK ((lease_token IS NULL) = (lease_expires_at IS NULL))
);

CREATE TABLE ingest.runs (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    command TEXT NOT NULL CHECK (command ~ '^catalog-[a-z]+(-[a-z]+)*$'),
    site_id TEXT REFERENCES ingest.sites (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    parameters JSONB NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(parameters) = 'object'),
    counters JSONB NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(counters) = 'object'),
    state TEXT NOT NULL DEFAULT 'running' CHECK (state IN ('running', 'completed', 'failed', 'interrupted')),
    error TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    finished_at TIMESTAMPTZ,
    CHECK ((state = 'running') = (finished_at IS NULL)),
    CHECK (finished_at IS NULL OR finished_at >= started_at),
    CHECK (error IS NULL OR state IN ('failed', 'interrupted'))
);

-- A page URL found by sitemap discovery, keyed by its requested URL. Its state is the resumable
-- fetch queue; evidence for every fetch stays in ingest.evidence.
CREATE TABLE ingest.source_pages (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    site_id TEXT NOT NULL REFERENCES ingest.sites (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    url TEXT NOT NULL UNIQUE CHECK (url ~ '^https://[a-z0-9.-]+/[^[:space:]]*$'),
    discovered_via TEXT NOT NULL CHECK (discovered_via ~ '^https://[a-z0-9.-]+/[^[:space:]]*$'),
    first_run_id BIGINT REFERENCES ingest.runs (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    sitemap_lastmod TIMESTAMPTZ,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    state TEXT NOT NULL DEFAULT 'queued' CHECK (state IN ('queued', 'fetched', 'failed')),
    state_reason TEXT CHECK (state_reason IS NULL OR trim(state_reason) <> ''),
    attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    latest_evidence_id BIGINT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (id, site_id),
    CHECK (state <> 'failed' OR state_reason IS NOT NULL),
    CHECK (state <> 'fetched' OR latest_evidence_id IS NOT NULL)
);

CREATE INDEX source_pages_queue_idx ON ingest.source_pages (site_id, next_attempt_at, id)
WHERE state = 'queued';

-- Every network attempt (robots, sitemap, or page), successful or not. Recorded once complete.
CREATE TABLE ingest.fetch_attempts (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    run_id BIGINT REFERENCES ingest.runs (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    site_id TEXT NOT NULL REFERENCES ingest.sites (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    source_page_id BIGINT,
    kind TEXT NOT NULL CHECK (kind IN ('robots', 'sitemap', 'page')),
    requested_url TEXT NOT NULL CHECK (requested_url ~ '^https://[a-z0-9.-]+/[^[:space:]]*$'),
    final_url TEXT CHECK (final_url IS NULL OR final_url ~ '^https://[a-z0-9.-]+/[^[:space:]]*$'),
    http_status INTEGER CHECK (http_status IS NULL OR http_status BETWEEN 100 AND 599),
    outcome TEXT NOT NULL CHECK (outcome IN (
        'success', 'http_status', 'timeout', 'network_error', 'bot_challenge', 'policy_rejected',
        'body_too_large'
    )),
    error TEXT,
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ NOT NULL,
    UNIQUE (id, source_page_id),
    FOREIGN KEY (source_page_id, site_id) REFERENCES ingest.source_pages (id, site_id)
    ON UPDATE RESTRICT ON DELETE RESTRICT,
    CHECK ((kind = 'page') = (source_page_id IS NOT NULL)),
    CHECK (finished_at >= started_at),
    CHECK (outcome <> 'success' OR (http_status BETWEEN 200 AND 299 AND final_url IS NOT NULL))
);

CREATE INDEX fetch_attempts_source_page_idx ON ingest.fetch_attempts (source_page_id)
WHERE source_page_id IS NOT NULL;
CREATE INDEX fetch_attempts_run_idx ON ingest.fetch_attempts (run_id) WHERE run_id IS NOT NULL;

-- One immutable successful page fetch. body_sha256 is the digest of the decoded (uncompressed)
-- body; canonical_link is the page's asserted rel=canonical exactly as found, valid or not.
CREATE TABLE ingest.evidence (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    fetch_attempt_id BIGINT NOT NULL UNIQUE,
    site_id TEXT NOT NULL,
    source_page_id BIGINT NOT NULL,
    requested_url TEXT NOT NULL CHECK (requested_url ~ '^https://[a-z0-9.-]+/[^[:space:]]*$'),
    final_url TEXT NOT NULL CHECK (final_url ~ '^https://[a-z0-9.-]+/[^[:space:]]*$'),
    redirect_chain JSONB NOT NULL DEFAULT '[]' CHECK (
        jsonb_typeof(redirect_chain) = 'array' AND jsonb_array_length(redirect_chain) <= 5
    ),
    http_status INTEGER NOT NULL CHECK (http_status BETWEEN 200 AND 299),
    fetched_at TIMESTAMPTZ NOT NULL,
    media_type TEXT NOT NULL CHECK (media_type ~ '^[a-z0-9][a-z0-9!#$&^_.+-]*/[a-z0-9][a-z0-9!#$&^_.+-]*$'),
    body_sha256 BYTEA NOT NULL CHECK (octet_length(body_sha256) = 32),
    body_size INTEGER NOT NULL CHECK (body_size BETWEEN 0 AND 5242880),
    body_encoding TEXT NOT NULL CHECK (body_encoding = 'gzip'),
    body_compressed BYTEA NOT NULL,
    canonical_link TEXT CHECK (canonical_link IS NULL OR length(canonical_link) BETWEEN 1 AND 4096),
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (id, site_id),
    UNIQUE (id, source_page_id),
    FOREIGN KEY (fetch_attempt_id, source_page_id) REFERENCES ingest.fetch_attempts (id, source_page_id)
    ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (source_page_id, site_id) REFERENCES ingest.source_pages (id, site_id)
    ON UPDATE RESTRICT ON DELETE RESTRICT
);

CREATE INDEX evidence_source_page_idx ON ingest.evidence (source_page_id, fetched_at);
CREATE INDEX evidence_body_sha256_idx ON ingest.evidence (body_sha256);

ALTER TABLE ingest.source_pages ADD CONSTRAINT source_pages_latest_evidence_fkey
FOREIGN KEY (latest_evidence_id, id) REFERENCES ingest.evidence (id, source_page_id)
ON UPDATE RESTRICT ON DELETE RESTRICT;

-- Every <script type="application/ld+json"> block in evidence, by document position.
CREATE TABLE ingest.evidence_jsonld_blocks (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    evidence_id BIGINT NOT NULL REFERENCES ingest.evidence (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    position INTEGER NOT NULL CHECK (position >= 0),
    raw_text TEXT NOT NULL,
    parse_status TEXT NOT NULL CHECK (parse_status IN ('parsed', 'invalid_json')),
    parse_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (evidence_id, position),
    UNIQUE (evidence_id, id),
    CHECK ((parse_status = 'parsed') = (parse_error IS NULL))
);

-- One Schema.org Recipe node. Identity is the normalized content digest plus @id; location is a
-- JSON Pointer into its block and is never identity.
CREATE TABLE ingest.source_nodes (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    site_id TEXT NOT NULL,
    evidence_id BIGINT NOT NULL,
    jsonld_block_id BIGINT NOT NULL,
    location TEXT NOT NULL CHECK (location = '' OR location ~ '^/'),
    node_id TEXT CHECK (node_id IS NULL OR trim(node_id) <> ''),
    normalization_version INTEGER NOT NULL CHECK (normalization_version > 0),
    content_digest BYTEA NOT NULL CHECK (octet_length(content_digest) = 32),
    raw_node JSONB NOT NULL CHECK (jsonb_typeof(raw_node) = 'object'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (jsonld_block_id, location),
    FOREIGN KEY (evidence_id, site_id) REFERENCES ingest.evidence (id, site_id)
    ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (evidence_id, jsonld_block_id) REFERENCES ingest.evidence_jsonld_blocks (evidence_id, id)
    ON UPDATE RESTRICT ON DELETE RESTRICT
);

CREATE INDEX source_nodes_identity_idx ON ingest.source_nodes (site_id, content_digest, node_id);
CREATE INDEX source_nodes_evidence_idx ON ingest.source_nodes (evidence_id);

-- Pauses persist until explicitly resumed. At most one open pause per site; history is kept.
CREATE TABLE ingest.site_pauses (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    site_id TEXT NOT NULL REFERENCES ingest.sites (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    reason TEXT NOT NULL CHECK (reason IN (
        'robots_unavailable', 'http_403', 'http_429', 'bot_challenge', 'operator'
    )),
    detail TEXT NOT NULL CHECK (trim(detail) <> ''),
    fetch_attempt_id BIGINT REFERENCES ingest.fetch_attempts (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    run_id BIGINT REFERENCES ingest.runs (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    paused_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    resumed_at TIMESTAMPTZ,
    resume_note TEXT,
    CHECK ((resumed_at IS NULL) = (resume_note IS NULL)),
    CHECK (resume_note IS NULL OR trim(resume_note) <> ''),
    CHECK (resumed_at IS NULL OR resumed_at >= paused_at)
);

CREATE UNIQUE INDEX site_pauses_one_open_idx ON ingest.site_pauses (site_id) WHERE resumed_at IS NULL;

-- +goose StatementBegin
CREATE FUNCTION ingest.reject_mutation() RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'ingest.% rows are immutable', TG_TABLE_NAME
        USING ERRCODE = 'restrict_violation';
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER fetch_attempts_immutable
BEFORE UPDATE OR DELETE ON ingest.fetch_attempts
FOR EACH ROW EXECUTE FUNCTION ingest.reject_mutation();

CREATE TRIGGER evidence_immutable
BEFORE UPDATE OR DELETE ON ingest.evidence
FOR EACH ROW EXECUTE FUNCTION ingest.reject_mutation();

CREATE TRIGGER evidence_jsonld_blocks_immutable
BEFORE UPDATE OR DELETE ON ingest.evidence_jsonld_blocks
FOR EACH ROW EXECUTE FUNCTION ingest.reject_mutation();

CREATE TRIGGER source_nodes_immutable
BEFORE UPDATE OR DELETE ON ingest.source_nodes
FOR EACH ROW EXECUTE FUNCTION ingest.reject_mutation();

-- A pause may only be closed once, by setting resumed_at and resume_note; it is never deleted.
-- +goose StatementBegin
CREATE FUNCTION ingest.enforce_site_pause_history() RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'ingest.site_pauses rows are never deleted' USING ERRCODE = 'restrict_violation';
    END IF;
    IF OLD.resumed_at IS NOT NULL THEN
        RAISE EXCEPTION 'site pause % is already resumed', OLD.id USING ERRCODE = 'restrict_violation';
    END IF;
    IF NEW.id IS DISTINCT FROM OLD.id
        OR NEW.site_id IS DISTINCT FROM OLD.site_id
        OR NEW.reason IS DISTINCT FROM OLD.reason
        OR NEW.detail IS DISTINCT FROM OLD.detail
        OR NEW.fetch_attempt_id IS DISTINCT FROM OLD.fetch_attempt_id
        OR NEW.run_id IS DISTINCT FROM OLD.run_id
        OR NEW.paused_at IS DISTINCT FROM OLD.paused_at THEN
        RAISE EXCEPTION 'site pause % may only be resumed', OLD.id USING ERRCODE = 'restrict_violation';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER site_pauses_history
BEFORE UPDATE OR DELETE ON ingest.site_pauses
FOR EACH ROW EXECUTE FUNCTION ingest.enforce_site_pause_history();

-- +goose Down
DROP SCHEMA IF EXISTS ingest CASCADE;
