from airflow import DAG
from airflow.operators.python import PythonOperator
from airflow.providers.postgres.operators.postgres import SQLExecuteQueryOperator

from datetime import datetime
import csv

# Аргументы по умолчанию: владелец процесса и время отсчёта для задачи
default_args = {
    'owner': 'airflow',
    'start_date': datetime(2024, 12, 1),
}

def sql_value(value):
    return "'" + value.replace("'", "''") + "'"

def generate_insert_queries_clients():
    CSV_FILE_PATH = 'data/clients.csv'
    with open( CSV_FILE_PATH, 'r') as csvfile:
        csvreader = csv.reader(csvfile)

        # Генерим запросы
        insert_queries = []
        is_header = True
        for row in csvreader:
            if is_header:
                is_header = False
                continue
            insert_query = f"INSERT INTO clients (id,username,name,created_at) VALUES ({row[0]}, {sql_value(row[1])}, {sql_value(row[2])}, {sql_value(row[3])}) ON CONFLICT DO NOTHING;"
            insert_queries.append(insert_query)

    return insert_queries

def generate_insert_queries_prostheses():
    CSV_FILE_PATH = 'data/sales.csv'
    with open( CSV_FILE_PATH, 'r') as csvfile:
        csvreader = csv.reader(csvfile)

        # Генерим запросы
        insert_queries = []
        is_header = True
        for row in csvreader:
            if is_header:
                is_header = False
                continue
            insert_query = f"INSERT INTO prostheses (id,name,created_at,client_id) VALUES ({sql_value(row[0])}, {sql_value(row[1])}, {sql_value(row[2])}, {row[3]}) ON CONFLICT DO NOTHING;"
            insert_queries.append(insert_query)

    return insert_queries

def generate_insert_queries_signals():
    CSV_FILE_PATH = 'data/signals.csv'
    with open( CSV_FILE_PATH, 'r') as csvfile:
        csvreader = csv.reader(csvfile)

        # Генерим запросы
        insert_queries = []
        is_header = True
        for row in csvreader:
            if is_header:
                is_header = False
                continue
            insert_query = f"INSERT INTO telemetry (id,prothesis_id,rotation_x,rotation_y,rotation_z,signal_force,created_at) VALUES ({row[0]}, {sql_value(row[1])}, {row[2]}, {row[3]}, {row[4]}, {row[5]}, {sql_value(row[6])}) ON CONFLICT DO NOTHING;"
            insert_queries.append(insert_query)

    return insert_queries

def generate_insert_queries():
    insert_queries_clients = generate_insert_queries_clients()
    insert_queries_prostheses = generate_insert_queries_prostheses()
    insert_queries_signals = generate_insert_queries_signals()

    # Сохраняем запросы
    with open('./dags/sql/insert_queries.sql', 'w') as f:
        for query in insert_queries_clients:
            f.write(f"{query}\n")
        for query in insert_queries_prostheses:
            f.write(f"{query}\n")
        for query in insert_queries_signals:
            f.write(f"{query}\n")

# Определяем DAG
with DAG('prothesis_dag',
         default_args=default_args, #аргументы по умолчанию в начале скрипта
         schedule_interval='@once', #запускаем один раз
         catchup=False) as dag: #предотвращает повторное выполнение DAG для пропущенных расписаний.

    #Опеределяем оператор для вставки данных
    generate_queries = PythonOperator(
        task_id='generate_insert_queries',
        python_callable=generate_insert_queries
    )

    #Запускаем выполнение оператора SQLExecuteQueryOperator
    run_insert_queries = SQLExecuteQueryOperator(
        task_id='run_insert_queries',
        conn_id='data_postgres',  # Название подключения к PostgreSQL в Airflow UI
        sql='sql/insert_queries.sql'
    )
    
    generate_queries>>run_insert_queries
    # Тут дальше можно продолжать пайплайн 