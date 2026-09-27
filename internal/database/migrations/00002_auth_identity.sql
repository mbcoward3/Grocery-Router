-- +goose Up
CREATE TABLE app_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    primary_email TEXT NOT NULL CHECK (trim(primary_email) <> '' AND primary_email = lower(primary_email)),
    display_name TEXT NOT NULL CHECK (trim(display_name) <> ''),
    avatar_url TEXT CHECK (avatar_url IS NULL OR trim(avatar_url) <> ''),
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    disabled_at TIMESTAMPTZ,
    UNIQUE (primary_email)
);

CREATE TABLE user_identities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES app_users(id) ON UPDATE RESTRICT ON DELETE CASCADE,
    provider TEXT NOT NULL CHECK (trim(provider) <> ''),
    issuer TEXT NOT NULL CHECK (trim(issuer) <> ''),
    subject TEXT NOT NULL CHECK (trim(subject) <> ''),
    provider_email TEXT NOT NULL CHECK (trim(provider_email) <> '' AND provider_email = lower(provider_email)),
    provider_email_verified BOOLEAN NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (issuer, subject)
);

CREATE TABLE households (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL CHECK (trim(name) <> ''),
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

CREATE TABLE household_memberships (
    household_id UUID NOT NULL REFERENCES households(id) ON UPDATE RESTRICT ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES app_users(id) ON UPDATE RESTRICT ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'member')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (household_id, user_id)
);

CREATE TABLE auth_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES app_users(id) ON UPDATE RESTRICT ON DELETE CASCADE,
    token_digest BYTEA NOT NULL UNIQUE CHECK (octet_length(token_digest) = 32),
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    idle_expires_at TIMESTAMPTZ NOT NULL,
    absolute_expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    client_metadata TEXT CHECK (client_metadata IS NULL OR length(client_metadata) <= 512),
    CHECK (idle_expires_at > created_at),
    CHECK (absolute_expires_at > idle_expires_at)
);

CREATE INDEX auth_sessions_user_id_idx ON auth_sessions(user_id);
CREATE INDEX household_memberships_user_id_idx ON household_memberships(user_id);

-- The fixed ID makes the one approved household an explicit deployment invariant, not a query default.
INSERT INTO households (id, name)
VALUES ('c0a7a2d8-669b-4e47-91c1-4d9a32f339d5', 'Coward');

-- A household may not lose its final owner.
-- +goose StatementBegin
CREATE FUNCTION prevent_last_household_owner() RETURNS TRIGGER AS $$
BEGIN
    IF OLD.role = 'owner' AND (TG_OP = 'DELETE' OR NEW.role <> 'owner') AND NOT EXISTS (
        SELECT 1 FROM household_memberships
        WHERE household_id = OLD.household_id AND role = 'owner' AND user_id <> OLD.user_id
    ) THEN
        RAISE EXCEPTION 'a household must retain at least one owner';
    END IF;
    RETURN CASE WHEN TG_OP = 'DELETE' THEN OLD ELSE NEW END;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER household_memberships_retain_owner
BEFORE DELETE OR UPDATE OF role ON household_memberships
FOR EACH ROW EXECUTE FUNCTION prevent_last_household_owner();

-- +goose Down
DROP TRIGGER IF EXISTS household_memberships_retain_owner ON household_memberships;
DROP FUNCTION IF EXISTS prevent_last_household_owner();
DROP TABLE IF EXISTS auth_sessions;
DROP TABLE IF EXISTS household_memberships;
DROP TABLE IF EXISTS households;
DROP TABLE IF EXISTS user_identities;
DROP TABLE IF EXISTS app_users;
