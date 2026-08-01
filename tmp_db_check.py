import os
from dotenv import load_dotenv
import psycopg2

load_dotenv('.env')
hosts = ['localhost','127.0.0.1','host.docker.internal','postgres']
for h in hosts:
    try:
        conn = psycopg2.connect(
            host=h,
            port=os.getenv('KAGGLE_DB_PORT','5432'),
            dbname=os.getenv('KAGGLE_DB_NAME','statgate_ml_staging'),
            user=os.getenv('KAGGLE_DB_USER','Kaggle'),
            password=os.getenv('KAGGLE_DB_PASSWORD','Statgate_kaggle'),
            connect_timeout=5,
        )
        print(h, 'OK')
        conn.close()
    except Exception as e:
        print(h, 'ERR', repr(e))
