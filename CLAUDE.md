# AI 总结功能开发指南

本文档记录了 RSS Reader (Miniflux) 项目中 AI 总结功能的开发过程和架构设计，用于指导后续开发。

## 一、功能概述

为 RSS 阅读器添加 AI 文章总结功能：
- 在单条文章页面 `/history/entry/{entryID}` 添加"AI总结"按钮
- 点击后流式显示 AI 生成的摘要
- 摘要保存到数据库，再次查看直接显示
- AI 服务独立运行，通过 API 与主服务交互

## 二、系统架构

```
┌─────────────────────────────────────────────────────────────────┐
│                         用户浏览器                               │
│                    (JavaScript EventSource)                      │
└──────────────────────────┬──────────────────────────────────────┘
                           │ SSE Stream
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Miniflux Go 服务 (:8080)                      │
│  ┌─────────────┐    ┌──────────────────┐    ┌───────────────┐  │
│  │   UI 路由   │───▶│ entry_ai_summary │───▶│   Storage     │  │
│  │  /entry/    │    │    (SSE Handler) │    │  (PostgreSQL) │  │
│  │ ai-summary  │    └────────┬─────────┘    └───────────────┘  │
│  └─────────────┘             │                                  │
└──────────────────────────────┼──────────────────────────────────┘
                               │ HTTP API (待接入)
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                   AI Agent 服务 (:5000)                          │
│  ┌─────────────┐    ┌──────────────────┐    ┌───────────────┐  │
│  │  Flask API  │───▶│ LangGraph Agent  │───▶│ LLM Provider  │  │
│  │ /api/summary│    │   (summary.py)   │    │(Tongyi/OpenAI)│  │
│  └─────────────┘    └──────────────────┘    └───────────────┘  │
│                              │                                  │
│                     ┌────────▼────────┐                         │
│                     │  user_ai_configs │ (用户API Key存储)      │
│                     │   (PostgreSQL)   │                        │
│                     └─────────────────┘                         │
└─────────────────────────────────────────────────────────────────┘
```

## 三、代码结构

### 3.1 Go 端 (Miniflux 主服务)

```
internal/
├── ai/                          # AI 服务抽象层
│   ├── ai.go                    # SummaryProvider 接口定义
│   ├── mock.go                  # Mock 实现（返回前100字）
│   └── provider.go              # Provider 工厂
├── model/
│   └── entry.go                 # Entry 模型 (+AISummary 字段)
├── storage/
│   ├── entry.go                 # +UpdateEntryAISummary()
│   └── entry_query_builder.go   # +ai_summary 字段查询
├── ui/
│   ├── ui.go                    # +路由 /entry/ai-summary/{entryID}
│   ├── entry_ai_summary.go      # SSE 流式响应 Handler
│   └── static/
│       ├── js/app.js            # +AiSummary 模块
│       └── css/common.css       # +.entry-ai-summary 样式
├── template/templates/views/
│   └── entry.html               # +AI总结按钮和显示区域
└── locale/translations/
    ├── en_US.json               # +entry.ai_summary.* 翻译
    └── zh_CN.json
```

### 3.2 Python 端 (AI Agent 服务)

```
ai/
├── .env.example                 # 环境变量模板
├── requirements.txt             # Python 依赖
├── run.sh                       # 启动脚本
└── src/
    ├── agent/summary/
    │   ├── summary.py           # LangGraph 工作流（原始版本）
    │   └── workflow.py          # 工作流工厂（支持动态配置）
    ├── api/
    │   ├── app.py               # Flask 应用入口
    │   └── routes/
    │       ├── summary.py       # POST /api/summary/generate
    │       └── config.py        # 用户配置 CRUD API
    ├── config/
    │   └── user_model.py        # 用户模型配置存储
    └── database/
        └── connection.py        # PostgreSQL 连接池
```

## 四、关键接口

### 4.1 Go 端 - SummaryProvider 接口

```go
// internal/ai/ai.go
type SummaryProvider interface {
    // 流式生成摘要，返回 channel 逐块输出
    GenerateSummary(ctx context.Context, content string) (<-chan string, error)
    Name() string
}
```

### 4.2 Go 端 - SSE Handler

```go
// internal/ui/entry_ai_summary.go
// 路由: GET /entry/ai-summary/{entryID}
func (h *handler) streamAISummary(w http.ResponseWriter, r *http.Request) {
    // 1. 获取 entry
    // 2. 如果已有摘要，直接流式返回
    // 3. 否则调用 AI Provider 生成
    // 4. 保存到数据库
    // 5. 发送 SSE done 事件
}
```

### 4.3 Python 端 - Flask API

```
POST /api/summary/generate
Request:
{
    "user_id": 1,
    "content": "文章内容...",
    "streaming": true
}

Response (streaming=false):
{"success": true, "summary": "摘要..."}

Response (streaming=true): SSE
data: {"chunk": "文"}
data: {"chunk": "字"}
event: done
data: {"success": true}
```

### 4.4 Python 端 - 用户配置 API

```
POST   /api/config/model          # 设置用户 AI 配置
GET    /api/config/model?user_id= # 获取配置
DELETE /api/config/model?user_id= # 删除配置
GET    /api/config/providers      # 列出支持的 LLM 提供商
```

## 五、数据库表

### 5.1 entries 表 (已有，新增字段)

```sql
ALTER TABLE entries ADD COLUMN ai_summary TEXT DEFAULT '';
```

### 5.2 user_ai_configs 表 (新增)

```sql
CREATE TABLE user_ai_configs (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL UNIQUE,
    provider VARCHAR(50) NOT NULL,      -- tongyi, openai, deepseek
    api_key VARCHAR(500) NOT NULL,
    model_name VARCHAR(100) NOT NULL,
    temperature REAL DEFAULT 0.3,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
```

## 六、前端实现

### 6.1 HTML 结构 (entry.html)

```html
<!-- AI 总结按钮 -->
<button class="page-button" data-ai-summary="true"
        data-ai-summary-url="{{ route "aiSummary" "entryID" .entry.ID }}"
        data-label-loading="{{ t "entry.state.loading" }}">
    {{ icon "sparkle" }}<span class="icon-label">{{ t "entry.ai_summary.label" }}</span>
</button>

<!-- 摘要显示区域（默认隐藏） -->
<div class="entry-ai-summary" id="ai-summary-container">
    <div class="ai-summary-header">
        <span class="ai-summary-title">{{ t "entry.ai_summary.title" }}</span>
        <button class="ai-summary-close" id="ai-summary-close">&times;</button>
    </div>
    <div class="ai-summary-content" id="ai-summary-content"></div>
</div>
```

### 6.2 JavaScript 模块 (app.js)

```javascript
const AiSummary = (function() {
    let eventSource = null;
    let isStreaming = false;
    let savedButtonElement = null;
    let savedOriginalButtonElement = null;

    function stopStreaming() {
        if (eventSource) { eventSource.close(); eventSource = null; }
        isStreaming = false;
        // 恢复按钮状态
    }

    function handleSummaryAction() {
        const isVisible = window.getComputedStyle(container).display !== 'none';
        if (isVisible) { stopStreaming(); container.style.display = 'none'; return; }

        // SSE 连接
        eventSource = new EventSource(url);
        eventSource.onmessage = (e) => { content.textContent += e.data; };
        eventSource.addEventListener('done', () => { stopStreaming(); });
        eventSource.onerror = () => { stopStreaming(); };
    }

    return { init, stopTyping: stopStreaming };
})();
```

## 七、待完成任务

### 7.1 接入真实 AI 服务

当前 Go 端使用 Mock Provider，需要修改为调用 Python AI Agent 服务：

```go
// internal/ai/langgraph.go (待实现)
type LangGraphProvider struct {
    baseURL string  // http://localhost:5000
    userID  int64
}

func (p *LangGraphProvider) GenerateSummary(ctx context.Context, content string) (<-chan string, error) {
    // POST /api/summary/generate with streaming=true
    // 解析 SSE 响应，转发到 channel
}
```

### 7.2 用户配置页面

在 Miniflux 设置页面添加 AI 配置入口：
- Provider 选择（通义千问/OpenAI/DeepSeek）
- API Key 输入
- 模型选择
- Temperature 调整

### 7.3 错误处理增强

- API Key 未配置时的友好提示
- 网络错误重试机制
- Token 限制处理

## 八、开发与调试

### 8.1 环境要求

- Go 1.22+
- Python 3.11+
- PostgreSQL (本地安装)
- 本地数据库: `miniflux2`

### 8.2 启动服务

开发时使用本地 PostgreSQL，两个终端分别启动服务：

```bash
# 终端 1: Miniflux Go 服务
make run

# 终端 2: AI Agent Python 服务
cd ai
pip install -r requirements.txt
./run.sh dev
```

### 8.3 环境变量配置

AI Agent 服务需要配置 `ai/.env` 文件（参考 `ai/.env.example`）：

```bash
# 数据库连接（本地 PostgreSQL）
DATABASE_URL=user=postgres password=postgres dbname=miniflux2 host=localhost sslmode=disable

# Flask 配置
FLASK_ENV=development
FLASK_DEBUG=1
```

### 8.4 测试 AI API

```bash
# 设置用户配置
curl -X POST http://localhost:5000/api/config/model \
  -H "Content-Type: application/json" \
  -d '{"user_id":1,"provider":"tongyi","api_key":"sk-xxx","model_name":"qwen-max"}'

# 测试摘要生成
curl -X POST http://localhost:5000/api/summary/generate \
  -H "Content-Type: application/json" \
  -d '{"user_id":1,"content":"测试内容...","streaming":false}'
```

### 8.5 SSE 调试

浏览器控制台：
```javascript
const es = new EventSource('/entry/ai-summary/123');
es.onmessage = e => console.log('chunk:', e.data);
es.addEventListener('done', () => console.log('完成'));
```

## 九、已修复的 Bug

1. **初始显示问题**: 页面加载时摘要区域应隐藏
   - 修复: CSS 设置 `display: none`，JS 使用 `getComputedStyle()` 检测

2. **按钮状态卡住**: 流式输出中隐藏面板，按钮保持"加载中"
   - 修复: 将按钮状态保存到模块级变量，`stopStreaming()` 时恢复

## 十、参考文档

- `doc/CLAUDE_REF/项目架构.md` - Miniflux 完整架构分析
- `doc/CLAUDE/251211.md` - 初始功能开发记录
- `doc/CLAUDE/251212.md` - SSE 流式输出实现
- `doc/CLAUDE/251225.md` - Flask API 服务搭建
