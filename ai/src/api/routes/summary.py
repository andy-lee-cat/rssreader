"""
Summary API 路由
支持流式和非流式输出
"""
from flask import Blueprint, request, jsonify, Response, stream_with_context
import json

from src.agent.summary.workflow import create_summary_workflow
from src.config.user_model import user_model_store

summary_bp = Blueprint("summary", __name__, url_prefix="/api/summary")


@summary_bp.route("/generate", methods=["POST"])
def generate_summary():
    """
    生成 RSS 文章摘要

    Request Body:
    {
        "user_id": int,          # 用户 ID（必需）
        "content": str,          # 需要总结的文本（必需）
        "streaming": bool        # 是否流式输出，默认 false
    }

    Response (非流式):
    {
        "success": true,
        "summary": "摘要内容..."
    }

    Response (流式): SSE 格式
    data: {"chunk": "文"}
    data: {"chunk": "字"}
    ...
    event: done
    data: {"success": true}
    """
    data = request.get_json()

    # 参数验证
    if not data:
        return jsonify({"success": False, "error": "请求体不能为空"}), 400

    user_id = data.get("user_id")
    content = data.get("content")
    streaming = data.get("streaming", False)

    if user_id is None:
        return jsonify({"success": False, "error": "user_id 是必需的"}), 400
    if not content:
        return jsonify({"success": False, "error": "content 是必需的"}), 400

    # 获取用户模型配置
    user_config = user_model_store.get(user_id)
    if not user_config:
        return jsonify({
            "success": False,
            "error": "用户未配置 AI 模型，请先设置 API Key 和模型"
        }), 400

    try:
        # 创建 workflow
        workflow = create_summary_workflow(user_config)

        if streaming:
            return Response(
                stream_with_context(_stream_summary(workflow, content)),
                mimetype="text/event-stream",
                headers={
                    "Cache-Control": "no-cache",
                    "Connection": "keep-alive",
                    "X-Accel-Buffering": "no",  # 禁用 nginx 缓冲
                }
            )
        else:
            result = _generate_summary_sync(workflow, content)
            return jsonify({"success": True, "summary": result})

    except Exception as e:
        return jsonify({"success": False, "error": str(e)}), 500


def _generate_summary_sync(workflow, content: str) -> str:
    """同步生成摘要"""
    from langchain.messages import HumanMessage

    messages = [HumanMessage(content=content)]
    result = workflow.invoke({"messages": messages})

    # 获取最后一条 AI 消息
    ai_message = result["messages"][-1]
    return ai_message.content


def _stream_summary(workflow, content: str):
    """流式生成摘要（SSE 格式）"""
    from langchain.messages import HumanMessage

    messages = [HumanMessage(content=content)]

    try:
        for message_chunk, metadata in workflow.stream(
            {"messages": messages},
            stream_mode="messages",
        ):
            if message_chunk.content:
                # SSE 格式：data: JSON\n\n
                chunk_data = json.dumps({"chunk": message_chunk.content}, ensure_ascii=False)
                yield f"data: {chunk_data}\n\n"

        # 发送完成事件
        yield "event: done\n"
        yield f"data: {json.dumps({'success': True})}\n\n"

    except Exception as e:
        error_data = json.dumps({"success": False, "error": str(e)}, ensure_ascii=False)
        yield f"event: error\n"
        yield f"data: {error_data}\n\n"
