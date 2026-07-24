CREATE TABLE IF NOT EXISTS clients
(
    id INTEGER PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS prostheses
(
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    client_id INTEGER NOT NULL REFERENCES clients(id)
);

CREATE TABLE IF NOT EXISTS telemetry
(
    id INTEGER PRIMARY KEY,
    prothesis_id TEXT NOT NULL REFERENCES prostheses(id),
    rotation_x NUMERIC(10, 2) NOT NULL,
    rotation_y NUMERIC(10, 2) NOT NULL,
    rotation_z NUMERIC(10, 2) NOT NULL,
    signal_force NUMERIC(10, 2) NOT NULL,
    created_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_prostheses_client_id ON prostheses(client_id);
CREATE INDEX IF NOT EXISTS idx_telemetry_prothesis_id ON telemetry(prothesis_id);

CREATE USER debezium WITH REPLICATION LOGIN PASSWORD 'debezium';

GRANT CONNECT ON DATABASE crm TO debezium;
GRANT USAGE ON SCHEMA public TO debezium;
GRANT SELECT ON TABLE clients, prostheses, telemetry TO debezium;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO debezium;

CREATE PUBLICATION debezium_publication FOR TABLE clients, prostheses, telemetry;
