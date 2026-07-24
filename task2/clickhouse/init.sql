CREATE DATABASE IF NOT EXISTS reports;

CREATE TABLE IF NOT EXISTS reports.clients
(
    id UInt32,
    username String,
    name String,
    created_at DateTime
)
ENGINE = ReplacingMergeTree
ORDER BY id;

CREATE TABLE IF NOT EXISTS reports.prostheses
(
    id String,
    name String,
    created_at DateTime,
    client_id UInt32
)
ENGINE = ReplacingMergeTree
ORDER BY (client_id, id);

CREATE TABLE IF NOT EXISTS reports.telemetry
(
    id UInt32,
    prothesis_id String,
    rotation_x Decimal(10, 2),
    rotation_y Decimal(10, 2),
    rotation_z Decimal(10, 2),
    signal_force Decimal(10, 2),
    created_at DateTime
)
ENGINE = ReplacingMergeTree
ORDER BY (prothesis_id, id);

CREATE TABLE IF NOT EXISTS reports.aggregated_data
(
    username String,
    name String,
    created_at DateTime,
    prothesis_ids_state AggregateFunction(groupUniqArray, String),
    signals_count_state AggregateFunction(count)
)
ENGINE = AggregatingMergeTree
ORDER BY (username, name, created_at);

CREATE MATERIALIZED VIEW IF NOT EXISTS reports.aggregated_data_mv
TO reports.aggregated_data
AS
SELECT
    c.username AS username,
    c.name AS name,
    c.created_at AS created_at,
    groupUniqArrayState(p.id) AS prothesis_ids_state,
    countState() AS signals_count_state
FROM reports.telemetry AS t
INNER JOIN reports.prostheses AS p ON t.prothesis_id = p.id
INNER JOIN reports.clients AS c ON p.client_id = c.id
GROUP BY c.username, c.name, c.created_at;
