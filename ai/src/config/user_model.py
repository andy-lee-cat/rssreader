"""
用户模型配置存储
使用 PostgreSQL 数据库存储（与 rssreader 共用同一数据库）
"""
from typing import Optional
from dataclasses import dataclass, asdict

from src.database import db_connection


@dataclass
class UserModelConfig:
    """用户模型配置"""
    user_id: int
    provider: str  # tongyi, openai, deepseek 等
    api_key: str
    model_name: str
    temperature: float = 0.3

    def to_dict(self) -> dict:
        return asdict(self)

    @classmethod
    def from_dict(cls, data: dict) -> "UserModelConfig":
        return cls(**data)

    @classmethod
    def from_row(cls, row: tuple) -> "UserModelConfig":
        """从数据库行创建配置对象"""
        # row: (user_id, provider, api_key, model_name, temperature)
        return cls(
            user_id=row[0],
            provider=row[1],
            api_key=row[2],
            model_name=row[3],
            temperature=row[4] if row[4] is not None else 0.3
        )


class UserModelStore:
    """用户模型配置存储（PostgreSQL 实现）"""

    def get(self, user_id: int) -> Optional[UserModelConfig]:
        """获取用户配置"""
        with db_connection() as conn:
            with conn.cursor() as cur:
                cur.execute(
                    """
                    SELECT user_id, provider, api_key, model_name, temperature
                    FROM user_ai_configs
                    WHERE user_id = %s
                    """,
                    (user_id,)
                )
                row = cur.fetchone()
                if row:
                    return UserModelConfig.from_row(row)
                return None

    def set(self, config: UserModelConfig) -> None:
        """设置用户配置（存在则更新，不存在则插入）"""
        with db_connection() as conn:
            with conn.cursor() as cur:
                cur.execute(
                    """
                    INSERT INTO user_ai_configs (user_id, provider, api_key, model_name, temperature)
                    VALUES (%s, %s, %s, %s, %s)
                    ON CONFLICT (user_id)
                    DO UPDATE SET
                        provider = EXCLUDED.provider,
                        api_key = EXCLUDED.api_key,
                        model_name = EXCLUDED.model_name,
                        temperature = EXCLUDED.temperature
                    """,
                    (config.user_id, config.provider, config.api_key, config.model_name, config.temperature)
                )
            conn.commit()

    def delete(self, user_id: int) -> bool:
        """删除用户配置"""
        with db_connection() as conn:
            with conn.cursor() as cur:
                cur.execute(
                    "DELETE FROM user_ai_configs WHERE user_id = %s",
                    (user_id,)
                )
                deleted = cur.rowcount > 0
            conn.commit()
            return deleted

    def get_or_default(self, user_id: int) -> Optional[UserModelConfig]:
        """获取用户配置，如果不存在则返回 None"""
        return self.get(user_id)


# 全局单例
user_model_store = UserModelStore()
