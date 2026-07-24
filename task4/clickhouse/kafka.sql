CREATE DATABASE IF NOT EXISTS reports;

CREATE TABLE IF NOT EXISTS reports.clients_kafka
(
    id UInt32,
    username String,
    name String,
    created_at DateTime
)
ENGINE = Kafka
SETTINGS
    kafka_broker_list = 'kafka:9092',
    kafka_topic_list = 'crm.public.clients',
    kafka_group_name = 'clickhouse_reports_clients',
    kafka_format = 'JSONEachRow',
    kafka_num_consumers = 1,
    kafka_skip_broken_messages = 1000;

CREATE TABLE IF NOT EXISTS reports.prostheses_kafka
(
    id String,
    name String,
    created_at DateTime,
    client_id UInt32
)
ENGINE = Kafka
SETTINGS
    kafka_broker_list = 'kafka:9092',
    kafka_topic_list = 'crm.public.prostheses',
    kafka_group_name = 'clickhouse_reports_prostheses',
    kafka_format = 'JSONEachRow',
    kafka_num_consumers = 1,
    kafka_skip_broken_messages = 1000;

CREATE TABLE IF NOT EXISTS reports.telemetry_kafka
(
    id UInt32,
    prothesis_id String,
    rotation_x Decimal(10, 2),
    rotation_y Decimal(10, 2),
    rotation_z Decimal(10, 2),
    signal_force Decimal(10, 2),
    created_at DateTime
)
ENGINE = Kafka
SETTINGS
    kafka_broker_list = 'kafka:9092',
    kafka_topic_list = 'crm.public.telemetry',
    kafka_group_name = 'clickhouse_reports_telemetry',
    kafka_format = 'JSONEachRow',
    kafka_num_consumers = 1,
    kafka_skip_broken_messages = 1000;

CREATE MATERIALIZED VIEW IF NOT EXISTS reports.clients_from_kafka_mv
TO reports.clients
AS
SELECT
    id,
    username,
    name,
    created_at
FROM reports.clients_kafka;

CREATE MATERIALIZED VIEW IF NOT EXISTS reports.prostheses_from_kafka_mv
TO reports.prostheses
AS
SELECT
    id,
    name,
    created_at,
    client_id
FROM reports.prostheses_kafka;

CREATE MATERIALIZED VIEW IF NOT EXISTS reports.telemetry_from_kafka_mv
TO reports.telemetry
AS
SELECT
    id,
    prothesis_id,
    rotation_x,
    rotation_y,
    rotation_z,
    signal_force,
    created_at
FROM reports.telemetry_kafka;
