import os
import re
import time
import logging
import threading
try:
    from prometheus_client import Gauge
except ImportError:
    Gauge = None
import psycopg2
import psycopg2.extras
import psycopg2.pool
from dotenv import load_dotenv

load_dotenv(os.path.join(os.path.dirname(__file__), '..', '.env'))

DEFAULT_DB_HOST = 'host.docker.internal' if os.path.exists('/.dockerenv') else 'localhost'
DB_HOST = os.getenv('KAGGLE_DB_HOST', DEFAULT_DB_HOST)
DB_PORT = os.getenv('KAGGLE_DB_PORT', '5432')
DB_NAME = os.getenv('KAGGLE_DB_NAME', 'statgate_ml_staging')
DB_USER = os.getenv('KAGGLE_DB_USER', 'Kaggle')
DB_PASSWORD = os.getenv('KAGGLE_DB_PASSWORD')
DB_SCHEMA = os.getenv('KAGGLE_STAGING_SCHEMA', 'ml_staging')
POOL_MIN = int(os.getenv('KAGGLE_DB_POOL_MIN', '1'))
POOL_MAX = int(os.getenv('KAGGLE_DB_POOL_MAX', '6'))
IDENTIFIER_PATTERN = re.compile(r'^[A-Za-z_][A-Za-z0-9_]{0,62}$')
DATASET_NAME_PATTERN = re.compile(r'^[a-z][a-z0-9_]{0,62}$')
KNOWN_ANALYTICS_DATASETS = [
    'covid_19_data',
    'gdp_by_region',
    'education_by_region',
    'renewable_energy_jobs_by_country',
    'global_health',
    'the_titanic_dataset',
    'billionaire_list_20yrs',
    'co2_emissions_by_country',
    'world_development_data_imputed',
]
ALLOW_FALLBACK = os.getenv('KAGGLE_ALLOW_FALLBACK', 'false').lower() in ('true', '1', 'yes')
PRODUCTION_ENV = os.getenv('STATGATE_ENV', 'development').lower() == 'production'

logger = logging.getLogger('kaggle_connector')
if not logger.handlers:
    handler = logging.StreamHandler()
    handler.setFormatter(logging.Formatter('%(asctime)s %(levelname)s %(name)s: %(message)s'))
    logger.addHandler(handler)
    logger.setLevel(logging.INFO)

# Module-level connection pool (initialized lazily)
_pool = None
_pool_in_use = 0
_pool_lock = threading.Lock()

class _NoopGauge:
    def __init__(self, *args, **kwargs):
        pass
    def set(self, value):
        pass


def _safe_gauge(name, help_text):
    if Gauge is None:
        return _NoopGauge()
    from prometheus_client import REGISTRY
    if name in REGISTRY._names_to_collectors:
        return REGISTRY._names_to_collectors[name]
    return Gauge(name, help_text)


# Prometheus gauges for DB pool
DB_POOL_IN_USE = _safe_gauge('statgate_db_pool_in_use', 'Connections currently checked out from DB pool')
DB_POOL_MAX = _safe_gauge('statgate_db_pool_max', 'Configured DB pool max size')


def _init_pool():
    global _pool
    if _pool is not None:
        return
    if not DB_PASSWORD:
        logger.warning('KAGGLE_DB_PASSWORD not set; DB pool will not be initialized')
        return
    try:
        _pool = psycopg2.pool.SimpleConnectionPool(
            POOL_MIN,
            POOL_MAX,
            host=DB_HOST,
            port=int(DB_PORT),
            dbname=DB_NAME,
            user=DB_USER,
            password=DB_PASSWORD,
            connect_timeout=5,
        )
        logger.info('Postgres connection pool initialized (min=%d max=%d)', POOL_MIN, POOL_MAX)
        try:
            DB_POOL_MAX.set(int(POOL_MAX))
        except Exception:
            pass
    except Exception as e:
        logger.exception('Failed to initialize Postgres pool: %s', e)
        _pool = None


class TableNotFoundError(RuntimeError):
    """Raised when a requested table does not exist in the target schema."""


def validate_identifier(identifier, label='identifier'):
    if not isinstance(identifier, str) or not IDENTIFIER_PATTERN.fullmatch(identifier):
        raise ValueError(f'Invalid {label}.')
    return identifier


def validate_dataset_name(dataset_name):
    if not isinstance(dataset_name, str) or not DATASET_NAME_PATTERN.fullmatch(dataset_name):
        raise ValueError('Invalid analytics dataset name. Use lowercase letters, digits, or underscores, starting with a letter.')
    return dataset_name


def _get_conn_from_pool():
    _init_pool()
    if _pool:
        try:
            conn = _pool.getconn()
            # track in-use count
            try:
                with _pool_lock:
                    global _pool_in_use
                    _pool_in_use += 1
                    DB_POOL_IN_USE.set(_pool_in_use)
            except Exception:
                pass
            return conn
        except Exception:
            logger.exception('Pool getconn failed; falling back to direct connection')
    # Fallback to direct connection if pool unavailable
    if not DB_PASSWORD:
        raise RuntimeError('KAGGLE_DB_PASSWORD must be configured before connecting to PostgreSQL.')
    return psycopg2.connect(
        host=DB_HOST,
        port=int(DB_PORT),
        dbname=DB_NAME,
        user=DB_USER,
        password=DB_PASSWORD,
        connect_timeout=5,
    )


def _put_conn_back(conn):
    if conn is None:
        return
    if _pool and isinstance(_pool, psycopg2.pool.AbstractConnectionPool):
        try:
            _pool.putconn(conn)
            try:
                with _pool_lock:
                    global _pool_in_use
                    if _pool_in_use > 0:
                        _pool_in_use -= 1
                    DB_POOL_IN_USE.set(_pool_in_use)
            except Exception:
                pass
            return
        except Exception:
            logger.exception('Failed to return connection to pool; closing')
    try:
        conn.close()
    except Exception:
        pass


def _retry(fn, attempts=3, backoff=0.5, *args, **kwargs):
    last_exc = None
    for i in range(attempts):
        try:
            return fn(*args, **kwargs)
        except Exception as e:
            if isinstance(e, TableNotFoundError):
                raise
            if isinstance(e, psycopg2.errors.UndefinedTable):
                raise TableNotFoundError(str(e))
            last_exc = e
            logger.debug('Attempt %d/%d failed: %s', i + 1, attempts, e)
            time.sleep(backoff * (2 ** i))
    raise last_exc


def init_pool_or_fail(timeout=30, interval=2):
    """Initialize the connection pool and verify DB connectivity, failing fast if unreachable.

    Raises RuntimeError if the pool cannot be initialized or no connection is possible
    within `timeout` seconds. Useful for container startup readiness (fail-fast).
    """
    # Ensure the pool object is created (or attempted)
    _init_pool()

    if not DB_PASSWORD:
        raise RuntimeError('KAGGLE_DB_PASSWORD not configured; cannot initialize DB pool')

    deadline = time.time() + float(timeout)
    last_err = None
    while time.time() < deadline:
        conn = None
        try:
            conn = _get_conn_from_pool()
            cur = conn.cursor()
            cur.execute('SELECT 1')
            _ = cur.fetchone()
            try:
                cur.close()
            except Exception:
                pass
            _put_conn_back(conn)
            logger.info('DB connectivity verified during startup')
            return True
        except Exception as e:
            last_err = e
            logger.warning('DB not ready yet (will retry): %s', e)
            try:
                _put_conn_back(conn)
            except Exception:
                pass
            time.sleep(float(interval))

    raise RuntimeError(f'Unable to connect to DB within {timeout} seconds during startup: {last_err}')


def is_db_connected():
    """Lightweight check used by readiness probes."""
    conn = None
    try:
        conn = _get_conn_from_pool()
        cur = conn.cursor()
        cur.execute('SELECT 1')
        _ = cur.fetchone()
        try:
            cur.close()
        except Exception:
            pass
        return True
    except Exception:
        logger.exception('DB connectivity check failed')
        return False
    finally:
        _put_conn_back(conn)


def list_kaggle_tables():
    """
    Dynamically discovers ALL tables in the ml_staging schema via information_schema.
    Returns bare table names (no schema prefix). Zero hardcoding.
    """
    def _list():
        conn = None
        try:
            conn = _get_conn_from_pool()
            cur = conn.cursor()
            cur.execute(
                """
                SELECT table_name
                FROM information_schema.tables
                WHERE table_schema = %s
                  AND table_type = 'BASE TABLE'
                ORDER BY table_name;
                """,
                (DB_SCHEMA,)
            )
            rows = cur.fetchall()
            try:
                cur.close()
            except Exception:
                pass
            if rows:
                return [r[0] for r in rows if DATASET_NAME_PATTERN.fullmatch(r[0])]
            return []
        finally:
            _put_conn_back(conn)

    try:
        return _retry(_list, attempts=3, backoff=0.3)
    except Exception as e:
        logger.exception('[kaggle_connector] list_kaggle_tables error: %s', e)
        if PRODUCTION_ENV:
            raise RuntimeError('Database unavailable; production mode forbids silent dataset fallback.')
        if not ALLOW_FALLBACK:
            raise RuntimeError('Database unavailable and KAGGLE_ALLOW_FALLBACK is disabled.')

    fallback = [
        "covid_19_data", "gdp_by_region", "global_health",
        "the_titanic_dataset", "billionaire_list_20yrs",
        "co2_emissions_by_country", "education_by_region",
        "unemployment_by_region", "renewable_energy_jobs_by_country",
        "world_development_data_imputed",
    ]
    if PRODUCTION_ENV:
        raise RuntimeError('Database unavailable; production mode forbids silent dataset fallback.')
    return [name for name in fallback if DATASET_NAME_PATTERN.fullmatch(name)]


def query_kaggle_table(table_name, limit=100):
    """
    Executes SELECT * FROM ml_staging.<table_name> LIMIT <limit> on the live database.
    Returns a list of plain dicts (column-name -> value). Zero hardcoded columns.
    """
    def _query():
        conn = None
        try:
            table_name_valid = validate_dataset_name(table_name)
            lim = max(1, min(int(limit), 1000))
            conn = _get_conn_from_pool()
            cur = conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor)
            sql = f'SELECT * FROM {DB_SCHEMA}."{table_name_valid}" LIMIT %s;'
            cur.execute(sql, (lim,))
            rows = cur.fetchall()
            try:
                cur.close()
            except Exception:
                pass
            return [dict(r) for r in rows]
        finally:
            _put_conn_back(conn)

    try:
        return _retry(_query, attempts=3, backoff=0.2)
    except TableNotFoundError:
        raise
    except Exception as e:
        logger.exception('[kaggle_connector] query_kaggle_table(%s) error: %s', table_name, e)
        if PRODUCTION_ENV:
            raise RuntimeError(f'Database unavailable while querying table {table_name}; production mode forbids silent fallback.')
        if not ALLOW_FALLBACK:
            raise RuntimeError(f'Database unavailable while querying table {table_name} and KAGGLE_ALLOW_FALLBACK is disabled.')
        return []


def get_table_schema(table_name):
    """
    Returns column metadata from information_schema.columns for the given table.
    Filters strictly by table_schema = ml_staging so results are unambiguous.
    Each entry: {column_name, data_type, is_nullable}
    """
    def _schema():
        conn = None
        try:
            table_name_valid = validate_dataset_name(table_name)
            conn = _get_conn_from_pool()
            cur = conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor)
            cur.execute(
                """
                SELECT column_name, data_type, is_nullable
                FROM information_schema.columns
                WHERE table_schema = %s
                  AND table_name   = %s
                ORDER BY ordinal_position;
                """,
                (DB_SCHEMA, table_name_valid)
            )
            rows = cur.fetchall()
            try:
                cur.close()
            except Exception:
                pass
            if rows:
                return [dict(r) for r in rows]
            raise TableNotFoundError(f'Table "{table_name_valid}" not found in schema "{DB_SCHEMA}".')
        finally:
            _put_conn_back(conn)

    try:
        res = _retry(_schema, attempts=3, backoff=0.2)
        return res
    except TableNotFoundError:
        raise
    except Exception as e:
        logger.exception('[kaggle_connector] get_table_schema(%s) error: %s', table_name, e)
        if PRODUCTION_ENV:
            raise RuntimeError(f'Database unavailable while reading schema for table {table_name}; production mode forbids silent fallback.')
        if not ALLOW_FALLBACK:
            raise RuntimeError(f'Database unavailable while reading schema for table {table_name} and KAGGLE_ALLOW_FALLBACK is disabled.')

    return [
        {"column_name": "id", "data_type": "integer", "is_nullable": "NO"},
        {"column_name": "value", "data_type": "double precision", "is_nullable": "YES"},
        {"column_name": "created_at", "data_type": "timestamp", "is_nullable": "YES"},
    ]
