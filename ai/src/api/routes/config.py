"""
用户模型配置 API 路由
"""
from flask import Blueprint, request, jsonify

from src.config.user_model import user_model_store, UserModelConfig

config_bp = Blueprint("config", __name__, url_prefix="/api/config")


@config_bp.route("/model", methods=["GET"])
def get_model_config():
    """
    获取用户模型配置

    Query Params:
        user_id: int

    Response:
    {
        "success": true,
        "config": {
            "user_id": 1,
            "provider": "tongyi",
            "model_name": "qwen-max",
            "temperature": 0.3
            // 注意：不返回 api_key
        }
    }
    """
    user_id = request.args.get("user_id", type=int)
    if user_id is None:
        return jsonify({"success": False, "error": "user_id 是必需的"}), 400

    config = user_model_store.get(user_id)
    if not config:
        return jsonify({"success": False, "error": "用户配置不存在"}), 404

    # 返回配置（隐藏 api_key）
    config_dict = config.to_dict()
    config_dict["api_key"] = "***"  # 隐藏敏感信息
    return jsonify({"success": True, "config": config_dict})


@config_bp.route("/model", methods=["POST"])
def set_model_config():
    """
    设置用户模型配置

    Request Body:
    {
        "user_id": int,
        "provider": str,      # "tongyi", "openai", "deepseek" 等
        "api_key": str,
        "model_name": str,    # "qwen-max", "gpt-4", "deepseek-chat" 等
        "temperature": float  # 可选，默认 0.3
    }

    Response:
    {
        "success": true,
        "message": "配置已保存"
    }
    """
    data = request.get_json()
    if not data:
        return jsonify({"success": False, "error": "请求体不能为空"}), 400

    required_fields = ["user_id", "provider", "api_key", "model_name"]
    for field in required_fields:
        if field not in data:
            return jsonify({"success": False, "error": f"{field} 是必需的"}), 400

    try:
        config = UserModelConfig(
            user_id=int(data["user_id"]),
            provider=data["provider"],
            api_key=data["api_key"],
            model_name=data["model_name"],
            temperature=float(data.get("temperature", 0.3))
        )
        user_model_store.set(config)
        return jsonify({"success": True, "message": "配置已保存"})
    except Exception as e:
        return jsonify({"success": False, "error": str(e)}), 500


@config_bp.route("/model", methods=["DELETE"])
def delete_model_config():
    """
    删除用户模型配置

    Query Params:
        user_id: int

    Response:
    {
        "success": true,
        "message": "配置已删除"
    }
    """
    user_id = request.args.get("user_id", type=int)
    if user_id is None:
        return jsonify({"success": False, "error": "user_id 是必需的"}), 400

    if user_model_store.delete(user_id):
        return jsonify({"success": True, "message": "配置已删除"})
    else:
        return jsonify({"success": False, "error": "用户配置不存在"}), 404


@config_bp.route("/providers", methods=["GET"])
def list_providers():
    """
    列出支持的模型提供商

    Response:
    {
        "success": true,
        "providers": [
            {
                "name": "tongyi",
                "display_name": "阿里通义千问",
                "models": ["qwen-max", "qwen-plus", "qwen-turbo"]
            },
            ...
        ]
    }
    """
    providers = [
        {
            "name": "tongyi",
            "display_name": "阿里通义千问",
            "models": ["qwen-max", "qwen-plus", "qwen-turbo"]
        },
        {
            "name": "openai",
            "display_name": "OpenAI",
            "models": ["gpt-4", "gpt-4-turbo", "gpt-3.5-turbo"]
        },
        {
            "name": "deepseek",
            "display_name": "DeepSeek",
            "models": ["deepseek-chat", "deepseek-reasoner"]
        },
    ]
    return jsonify({"success": True, "providers": providers})
