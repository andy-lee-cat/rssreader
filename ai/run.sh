#!/bin/bash
# AI Agent 服务启动脚本

cd "$(dirname "$0")"

# 加载环境变量
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

# 设置 Python 路径
export PYTHONPATH="${PYTHONPATH}:$(pwd)"

# 开发模式
if [ "$1" = "dev" ]; then
    echo "Starting AI Agent in development mode..."
    python src/api/app.py
# 生产模式
else
    echo "Starting AI Agent in production mode..."
    gunicorn -w 4 -b 0.0.0.0:5000 --timeout 120 src.api.app:app
fi
