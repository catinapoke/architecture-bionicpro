from airflow import DAG
from airflow.hooks.base import BaseHook
from airflow.operators.python import PythonOperator

from datetime import datetime
import csv
from decimal import Decimal

import clickhouse_connect

# Аргументы по умолчанию: владелец процесса и время отсчёта для задачи
default_args = {
    'owner': 'airflow',
    'start_date': datetime(2024, 12, 1),
}

DATETIME_FORMAT = '%Y-%m-%d %H:%M:%S'
CLICKHOUSE_CONN_ID = 'clickhouse_default'


def clickhouse_client():
    conn = BaseHook.get_connection(CLICKHOUSE_CONN_ID)
    return clickhouse_connect.get_client(
        host=conn.host,
        port=conn.port or 8123,
        database=conn.schema or 'reports',
        username=conn.login or 'default',
        password=conn.password or '',
    )


def parse_datetime(value):
    return datetime.strptime(value, DATETIME_FORMAT)


def read_csv(path):
    with open(path, 'r', newline='') as csvfile:
        return list(csv.DictReader(csvfile))


def load_raw_data():
    client = clickhouse_client()

    client.command('TRUNCATE TABLE aggregated_data')
    client.command('TRUNCATE TABLE telemetry')
    client.command('TRUNCATE TABLE prostheses')
    client.command('TRUNCATE TABLE clients')

    clients = [
        [
            int(row['id']),
            row['username'],
            row['name'],
            parse_datetime(row['created_at']),
        ]
        for row in read_csv('data/clients.csv')
    ]
    prostheses = [
        [
            row['id'],
            row['name'],
            parse_datetime(row['created_at']),
            int(row['client_id']),
        ]
        for row in read_csv('data/sales.csv')
    ]
    telemetry = [
        [
            int(row['id']),
            row['prothesis_id'],
            Decimal(row['rotation_x']),
            Decimal(row['rotation_y']),
            Decimal(row['rotation_z']),
            Decimal(row['signal_force']),
            parse_datetime(row['created_at']),
        ]
        for row in read_csv('data/signals.csv')
    ]

    client.insert(
        'clients',
        data=clients,
        column_names=['id', 'username', 'name', 'created_at'],
    )
    client.insert(
        'prostheses',
        data=prostheses,
        column_names=['id', 'name', 'created_at', 'client_id'],
    )
    client.insert(
        'telemetry',
        data=telemetry,
        column_names=[
            'id',
            'prothesis_id',
            'rotation_x',
            'rotation_y',
            'rotation_z',
            'signal_force',
            'created_at',
        ],
    )


def build_mart():
    client = clickhouse_client()
    # aggregated_data is filled by aggregated_data_mv during telemetry inserts.
    client.command('OPTIMIZE TABLE aggregated_data FINAL')


# Определяем DAG
with DAG('prothesis_dag',
         default_args=default_args, #аргументы по умолчанию в начале скрипта
         schedule_interval='@daily',
         catchup=False) as dag: #предотвращает повторное выполнение DAG для пропущенных расписаний.

    load_raw = PythonOperator(
        task_id='load_raw_data',
        python_callable=load_raw_data
    )

    rebuild_mart = PythonOperator(
        task_id='build_mart',
        python_callable=build_mart
    )

    load_raw >> rebuild_mart