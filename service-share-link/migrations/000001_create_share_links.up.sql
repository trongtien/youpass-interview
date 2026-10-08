CREATE TABLE IF NOT EXISTS share_links (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code varchar(8) NOT NULL,
    owner_id bigint NOT NULL,
    resource_id bigint NOT NULL,
    status text NOT NULL DEFAULT 'active',
    view_count bigint NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    CONSTRAINT share_links_code_key UNIQUE (code),
    CONSTRAINT share_links_status_check CHECK (status IN ('active', 'disabled', 'deleted'))
);
CREATE INDEX IF NOT EXISTS share_links_owner_id_idx ON share_links (owner_id);
CREATE UNIQUE INDEX IF NOT EXISTS share_links_live_resource_key ON share_links (resource_id)
WHERE status <> 'deleted';