CREATE TABLE IF NOT EXISTS resources (
    id              BIGSERIAL PRIMARY KEY,
    namespace       TEXT NOT NULL,
    pod_name        TEXT NOT NULL,
    deployment      TEXT NOT NULL DEFAULT '',
    cpu_request     BIGINT NOT NULL DEFAULT 0,
    cpu_usage       BIGINT NOT NULL DEFAULT 0,
    memory_request  BIGINT NOT NULL DEFAULT 0,
    memory_usage    BIGINT NOT NULL DEFAULT 0,
    collected_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_resources_namespace ON resources (namespace);
CREATE INDEX idx_resources_deployment ON resources (deployment);
CREATE INDEX idx_resources_collected_at ON resources (collected_at);

CREATE TABLE IF NOT EXISTS recommendations (
    id                       BIGSERIAL PRIMARY KEY,
    namespace                TEXT NOT NULL,
    deployment               TEXT NOT NULL,
    current_request_cpu      BIGINT NOT NULL DEFAULT 0,
    current_request_memory   BIGINT NOT NULL DEFAULT 0,
    recommended_cpu          BIGINT NOT NULL DEFAULT 0,
    recommended_memory       BIGINT NOT NULL DEFAULT 0,
    monthly_savings_jpy      BIGINT NOT NULL DEFAULT 0,
    message                  TEXT NOT NULL DEFAULT '',
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_recommendations_namespace ON recommendations (namespace);
CREATE INDEX idx_recommendations_deployment ON recommendations (deployment);
CREATE INDEX idx_recommendations_created_at ON recommendations (created_at);
