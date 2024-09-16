CREATE TABLE registry (
    "registry"           TEXT    NOT NULL,
    "node"               TEXT    NOT NULL,
    "transport"          TEXT    NOT NULL,
    "transport_details"  TEXT    NOT NULL,
    PRIMARY KEY ("registry","node","transport")
);

CREATE UNIQUE INDEX node_transport ON registry("node","transport");
