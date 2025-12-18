from typing import Literal
from typing_extensions import TypedDict, Annotated
import operator

from langchain.messages import AnyMessage, SystemMessage, HumanMessage
from langgraph.graph import StateGraph, START, END
from langchain_community.chat_models.tongyi import ChatTongyi

model = ChatTongyi(  # type: ignore
    model="qwen-max",
    streaming=True,
    model_kwargs={
        "temperature": 0.3,
        "enable_thinking": False,
    }    
)

class MessagesState(TypedDict):
    messages: Annotated[list[AnyMessage], operator.add]

def llm_node(state: dict):
    """LLM Node"""
    return {
        "messages": [
            model.invoke(
                [
                    SystemMessage(
                        content="你是一个专业的助手，请对于用户输入的内容（rss新闻）进行简要总结，内容在100字左右，不超过200字。如果用户输入本身就小于100字，请直接返回用户输入。"
                    )
                ] 
                + state["messages"]
            )
        ]
    }

workflow_builder = StateGraph(MessagesState)
workflow_builder.add_node("llm_node", llm_node)
workflow_builder.add_edge(START, "llm_node")
workflow_builder.add_edge("llm_node", END)
workflow = workflow_builder.compile()

if __name__ == "__main__":
    from prompt.test_input import test_input
    messages = [HumanMessage(content=test_input)]
    
    testcase = 0

    if testcase == 0:
        # 非流式输出
        messages = workflow.invoke({"messages": messages})
        for m in messages["messages"]:
            m.pretty_print()
    else: 
        # 流式输出
        for message_chunk, metadata in workflow.stream(
            {"messages": messages},
            stream_mode="messages",  
        ):
            if message_chunk.content:
                print(message_chunk.content, end="|", flush=True)