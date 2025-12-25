"""
Summary Workflow 工厂
根据用户配置创建 LangGraph 工作流
"""
from typing import Optional
from typing_extensions import TypedDict, Annotated
import operator
import os

from langchain.messages import AnyMessage, SystemMessage
from langgraph.graph import StateGraph, START, END

from src.config.user_model import UserModelConfig


# 系统提示词
SUMMARY_SYSTEM_PROMPT = """你是一个专业的助手，请对于用户输入的内容（rss新闻）进行简要总结，内容在100字左右，不超过200字。如果用户输入本身就小于100字，请直接返回用户输入。"""


class MessagesState(TypedDict):
    """消息状态"""
    messages: Annotated[list[AnyMessage], operator.add]


def _create_model(config: UserModelConfig):
    """根据配置创建 LLM 模型"""
    provider = config.provider.lower()

    if provider == "tongyi":
        from langchain_community.chat_models.tongyi import ChatTongyi
        # 设置环境变量
        os.environ["DASHSCOPE_API_KEY"] = config.api_key
        return ChatTongyi(
            model=config.model_name,
            streaming=True,
            model_kwargs={
                "temperature": config.temperature,
                "enable_thinking": False,
            }
        )

    elif provider == "openai":
        from langchain_openai import ChatOpenAI
        return ChatOpenAI(
            model=config.model_name,
            api_key=config.api_key,
            temperature=config.temperature,
            streaming=True,
        )

    elif provider == "deepseek":
        from langchain_openai import ChatOpenAI
        return ChatOpenAI(
            model=config.model_name,
            api_key=config.api_key,
            base_url="https://api.deepseek.com/v1",
            temperature=config.temperature,
            streaming=True,
        )

    else:
        raise ValueError(f"不支持的模型提供商: {provider}")


def create_summary_workflow(config: UserModelConfig):
    """
    创建摘要工作流

    Args:
        config: 用户模型配置

    Returns:
        编译后的 LangGraph 工作流
    """
    model = _create_model(config)

    def llm_node(state: dict):
        """LLM 节点"""
        return {
            "messages": [
                model.invoke(
                    [SystemMessage(content=SUMMARY_SYSTEM_PROMPT)]
                    + state["messages"]
                )
            ]
        }

    # 构建工作流
    workflow_builder = StateGraph(MessagesState)
    workflow_builder.add_node("llm_node", llm_node)
    workflow_builder.add_edge(START, "llm_node")
    workflow_builder.add_edge("llm_node", END)

    return workflow_builder.compile()
