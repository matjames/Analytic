import os
import duckdb
import pandas as pd
from kaggle_connector import query_kaggle_table, get_table_schema

class StatGateSDK:
    def __init__(self):
        self.duck_conn = duckdb.connect(database=':memory:')

    def load_dataset(self, table_name, limit=1000):
        """
        SDK entry point: loads dataset into a Pandas DataFrame using DuckDB in-memory engine.
        Usage:
            from statgate import engine
            df = engine.load_dataset('housing_prices_dataset')
        """
        records = query_kaggle_table(table_name, limit=limit)
        if records:
            df = pd.DataFrame(records)
            self.duck_conn.register(table_name, df)
            return df
        return pd.DataFrame()

    def get_schema(self, table_name):
        """
        Returns dynamic schema attributes from information_schema.
        """
        return get_table_schema(table_name)

    def execute_sql(self, query):
        """
        Executes arbitrary SQL on loaded DuckDB in-memory tables.
        """
        return self.duck_conn.execute(query).df()

engine = StatGateSDK()
