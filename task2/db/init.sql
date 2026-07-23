CREATE DATABASE sample;
GRANT ALL PRIVILEGES ON DATABASE sample TO airflow; 

\connect sample

CREATE TABLE clients (
    id SERIAL PRIMARY KEY,
    username TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE prostheses (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    client_id INTEGER NOT NULL REFERENCES clients(id)
);

CREATE TABLE telemetry (
    id SERIAL PRIMARY KEY,
    prothesis_id TEXT NOT NULL REFERENCES prostheses(id),
    rotation_x DECIMAL(10, 2) NOT NULL,
    rotation_y DECIMAL(10, 2) NOT NULL,
    rotation_z DECIMAL(10, 2) NOT NULL,
    signal_force DECIMAL(10, 2) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE VIEW aggregated_data AS
SELECT
    c.username,
    c.name,
    c.created_at,
    COALESCE(array_agg(DISTINCT p.id) FILTER (WHERE p.id IS NOT NULL), '{}') AS prothesis_ids,
    COUNT(t.id)::bigint AS signals_count
FROM clients c
LEFT JOIN prostheses p ON c.id = p.client_id
LEFT JOIN telemetry t ON p.id = t.prothesis_id
GROUP BY c.username, c.name, c.created_at;