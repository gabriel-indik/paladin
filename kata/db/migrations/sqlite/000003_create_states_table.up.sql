CREATE TABLE states (
    hash_l      UUID,
    hash_h      UUID,
    created_at  BIGINT,
    domain_id   VARCHAR,
    schema_l    UUID,
    schema_h    UUID,
    data        VARCHAR,
    PRIMARY KEY (hash_l, hash_h),
    FOREIGN KEY (schema_l, schema_h) REFERENCES schemas (hash_l, hash_h) ON DELETE CASCADE
);

CREATE TABLE state_labels (
    state_l     UUID,
    state_h     UUID,   
    label       VARCHAR,
    value       VARCHAR,
    PRIMARY KEY (state_l, state_h, label),
    FOREIGN KEY (state_l, state_h) REFERENCES states (hash_l, hash_h) ON DELETE CASCADE
);
CREATE INDEX state_labels_value ON state_labels(value);