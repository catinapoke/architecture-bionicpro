CREATE DATABASE sample;
GRANT ALL PRIVILEGES ON DATABASE sample TO airflow; 

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
    client_id TEXT NOT NULL REFERENCES clients(id)
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
    p.name AS prosthesis_name,
    COUNT(t.id) AS signals_count
FROM clients c
JOIN prostheses p ON c.id = p.client_id
JOIN telemetry t ON p.id = t.prothesis_id
GROUP BY c.username, c.name, c.created_at, p.name;