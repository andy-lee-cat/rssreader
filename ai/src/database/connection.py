"""
数据库连接管理
使用与 rssreader 相同的 PostgreSQL 数据库
"""
import os
import psycopg2
from psycopg2 import pool
from contextlib import contextmanager
from typing import Optional

# 连接池（全局单例）
_connection_pool: Optional[pool.ThreadedConnectionPool] = None


def get_database_url() -> str:
    """获取数据库连接字符串"""
    # 优先从环境变量读取，格式与 rssreader 兼容
    database_url = os.environ.get("DATABASE_URL")

    if database_url:
        return database_url

    # 默认值（与 rssreader 默认配置一致）
    return "user=postgres password=postgres dbname=miniflux2 sslmode=disable"


def init_db(min_conn: int = 1, max_conn: int = 10):
    """
    初始化数据库连接池

    Args:
        min_conn: 最小连接数
        max_conn: 最大连接数
    """
    global _connection_pool

    if _connection_pool is not None:
        return

    database_url = get_database_url()

    _connection_pool = pool.ThreadedConnectionPool(
        minconn=min_conn,
        maxconn=max_conn,
        dsn=database_url
    )

    # 初始化表结构
    _init_tables()


def close_db():
    """关闭数据库连接池"""
    global _connection_pool

    if _connection_pool is not None:
        _connection_pool.closeall()
        _connection_pool = None


def get_db_connection():
    """
    获取数据库连接（从连接池）

    使用方式:
        conn = get_db_connection()
        try:
            with conn.cursor() as cur:
                cur.execute("SELECT ...")
            conn.commit()
        finally:
            release_db_connection(conn)
    """
    global _connection_pool

    if _connection_pool is None:
        init_db()

    return _connection_pool.getconn()


def release_db_connection(conn):
    """归还连接到连接池"""
    global _connection_pool

    if _connection_pool is not None and conn is not None:
        _connection_pool.putconn(conn)


@contextmanager
def db_connection():
    """
    数据库连接上下文管理器

    使用方式:
        with db_connection() as conn:
            with conn.cursor() as cur:
                cur.execute("SELECT ...")
            conn.commit()
    """
    conn = get_db_connection()
    try:
        yield conn
    finally:
        release_db_connection(conn)


def _init_tables():
    """初始化 AI 服务所需的表"""
    conn = get_db_connection()
    try:
        with conn.cursor() as cur:
            # 创建 user_ai_configs 表
            cur.execute("""
                CREATE TABLE IF NOT EXISTS user_ai_configs (
                    id SERIAL PRIMARY KEY,
                    user_id INTEGER NOT NULL UNIQUE,
                    provider VARCHAR(50) NOT NULL,
                    api_key VARCHAR(500) NOT NULL,
                    model_name VARCHAR(100) NOT NULL,
                    temperature REAL DEFAULT 0.3,
                    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                    CONSTRAINT fk_user
                        FOREIGN KEY (user_id)
                        REFERENCES users(id)
                        ON DELETE CASCADE
                )
            """)

            # 创建更新时间触发器（如果不存在）
            cur.execute("""
                CREATE OR REPLACE FUNCTION update_updated_at_column()
                RETURNS TRIGGER AS $$
                BEGIN
                    NEW.updated_at = CURRENT_TIMESTAMP;
                    RETURN NEW;
                END;
                $$ language 'plpgsql'
            """)

            # 检查触发器是否存在
            cur.execute("""
                SELECT EXISTS (
                    SELECT 1 FROM pg_trigger
                    WHERE tgname = 'update_user_ai_configs_updated_at'
                )
            """)
            trigger_exists = cur.fetchone()[0]

            if not trigger_exists:
                cur.execute("""
                    CREATE TRIGGER update_user_ai_configs_updated_at
                        BEFORE UPDATE ON user_ai_configs
                        FOR EACH ROW
                        EXECUTE FUNCTION update_updated_at_column()
                """)

        conn.commit()
    finally:
        release_db_connection(conn)
