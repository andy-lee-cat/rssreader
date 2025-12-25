"""
Flask API 服务入口
用于提供 LangGraph AI Agent 服务
"""
import os
import sys
import atexit

# 确保 src 目录在 Python 路径中
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))

from flask import Flask, jsonify
from flask_cors import CORS

from src.api.routes.summary import summary_bp
from src.api.routes.config import config_bp
from src.database import init_db, close_db


def create_app():
    """创建 Flask 应用"""
    app = Flask(__name__)

    # 初始化数据库连接池
    init_db()

    # 注册关闭时清理数据库连接
    atexit.register(close_db)

    # 跨域支持（允许 rssreader 前端调用）
    CORS(app, resources={
        r"/api/*": {
            "origins": ["http://localhost:8080", "http://127.0.0.1:8080"],
            "methods": ["GET", "POST", "DELETE", "OPTIONS"],
            "allow_headers": ["Content-Type"],
        }
    })

    # 注册蓝图
    app.register_blueprint(summary_bp)
    app.register_blueprint(config_bp)

    # 健康检查端点
    @app.route("/health", methods=["GET"])
    def health():
        return jsonify({"status": "ok", "service": "ai-agent"})

    # 错误处理
    @app.errorhandler(404)
    def not_found(e):
        return jsonify({"success": False, "error": "Not Found"}), 404

    @app.errorhandler(500)
    def internal_error(e):
        return jsonify({"success": False, "error": "Internal Server Error"}), 500

    return app


# 应用实例
app = create_app()


if __name__ == "__main__":
    # 开发模式运行
    # 生产环境请使用 gunicorn: gunicorn -w 4 -b 0.0.0.0:5000 src.api.app:app
    app.run(
        host="0.0.0.0",
        port=5000,
        debug=True,
        threaded=True,  # 支持多线程处理流式响应
    )
