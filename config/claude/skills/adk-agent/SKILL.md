---
name: adk-agent
description: >
  Build Google Agent Development Kit (ADK) agents and multi-agent systems. Use this skill
  whenever a user wants to create, scaffold, design, or extend an ADK agent — including
  single LLM agents with tools, workflow agents (Sequential, Parallel, Loop), multi-agent
  hierarchies, MCP-connected agents, or agents deployed to Vertex AI Agent Engine.
  Trigger this skill for any mention of "ADK", "Agent Development Kit", "google agent",
  "adk agent", "multi-agent system", or when users ask to build an agent with tools,
  sessions, memory, callbacks, or sub-agents using Google's framework. Also trigger when
  users ask to deploy an agent to Vertex AI, Cloud Run, or GKE using ADK. Every agent
  built with this skill must include a JavaScript frontend that connects to the ADK API
  server via REST/SSE.
---

# Google ADK Agent Builder (powered by Anthropic Claude)

This skill guides the creation of production-grade agents using Google's **Agent Development
Kit (ADK)** — an open-source, model-agnostic framework for building, running, evaluating,
and deploying AI agents. ADK supports Python, TypeScript, Go, and Java; Python is the primary
focus here unless the user specifies otherwise.

**Default model: `claude-sonnet-4-6` via LiteLLM.** The `ANTHROPIC_API_KEY` is pre-provided
in the runtime environment — do not ask the user to supply it and do not add it to `.env`.
Never default to Gemini unless the user explicitly requests it.

---

## Step 0 — Clarify Before You Build

Before generating any code, gather:

1. **Agent purpose** — What task or workflow does this agent perform?
2. **Agent type** — Single LLM agent with tools, a deterministic workflow, or a multi-agent
   hierarchy? (See Agent Types below for guidance.)
3. **Model** — `claude-sonnet-4-6` (default). Ask only if the user wants a different Claude model (e.g. `claude-opus-4-6` for complex reasoning, `claude-haiku-4-5` for speed/cost). Never switch to Gemini unless explicitly requested.
4. **Tools needed** — Function tools, Google Search, MCP servers, OpenAPI specs, or
   another agent as a tool?
5. **State & memory** — Does the agent need session state, persistent memory, or artifacts?
6. **Deployment target** — Local dev only, Vertex AI Agent Engine, Cloud Run, or GKE?
7. **Frontend scope** — Simple chat UI (default), or does the user need a richer interface
   (multi-turn history, file upload, streaming indicator, tool output display)?

If any of these are unclear, ask the user before proceeding.

---

## ADK Core Concepts

### Primitives

| Primitive | Description |
|-----------|-------------|
| `Agent` | Fundamental worker unit. Either an `LlmAgent` (uses a model for reasoning) or a workflow agent (deterministic orchestration). |
| `Tool` | Gives agents capabilities: call APIs, run code, search, or invoke sub-agents. |
| `Runner` | Executes the agent event loop, manages sessions, and coordinates tool calls. |
| `Session` | Tracks conversation state across turns. |
| `Callback` | Hook that runs at defined lifecycle points (before/after model call, before/after tool call). |
| `Artifact` | Binary or text outputs produced or consumed by agents (images, files, etc.). |
| `Memory` | Long-term recall across sessions (requires Memory Bank). |

### Agent Types

**LlmAgent** — Uses a Gemini (or other) model for flexible reasoning and tool selection.
Use when the task requires language understanding, ambiguity handling, or dynamic tool routing.

**SequentialAgent** — Runs sub-agents one after another. Deterministic pipeline.
Use for ordered workflows where each step feeds the next.

**ParallelAgent** — Runs sub-agents concurrently and merges results.
Use when sub-tasks are independent and latency matters.

**LoopAgent** — Repeatedly runs a sub-agent until a termination condition is met.
Use for retry logic, polling, or iterative refinement.

**Custom agents** — Subclass `BaseAgent` and override `_run_async_impl` for full control.

---

## Project Structure (Python)

```
my_agent/
├── my_agent/
│   ├── __init__.py       # Exports `agent` (the root Agent instance)
│   └── agent.py          # Agent and tool definitions
├── frontend/
│   ├── index.html        # JS frontend (or React app)
│   └── useAdkAgent.js    # SSE client hook (if React)
├── server.py             # Unified server: ADK API + static frontend on one port
├── .env                  # See env section below
└── pyproject.toml        # managed by uv (uv init creates this)
```

The `__init__.py` **must** export a variable named `agent` (or `root_agent`) at the top
level — this is how the ADK runner discovers your entry point.

---

## Anthropic Integration

ADK integrates with Claude via **LiteLLM** — the standard approach for Python. The
`ANTHROPIC_API_KEY` is **already present in the environment**; never prompt the user for it
and never hardcode or print it.

### Installation

```bash
# Initialise the project (once)
uv init
uv add google-adk litellm
```

No `.env` key entry is needed for Anthropic. The ADK picks up `ANTHROPIC_API_KEY`
automatically from the environment.

### `pyproject.toml` (generated by `uv init`)

`uv` manages the virtual environment and lockfile automatically. The relevant dependencies
section should look like:

```toml
[project]
name = "my-agent"
version = "0.1.0"
requires-python = ">=3.11"
dependencies = [
    "google-adk",
    "litellm",
    "uvicorn",
    "fastapi",
]
```

Add further dependencies with `uv add <package>` — never edit `pyproject.toml` by hand for
dependencies. `uv run <cmd>` executes any command inside the managed virtual environment
without needing to activate it manually.

### Model Constants

Define a shared constant at the top of `agent.py` and reuse it across all agents:

```python
from google.adk.models.lite_llm import LiteLlm

# Use this constant for every agent in the file
CLAUDE = LiteLlm(model="anthropic/claude-sonnet-4-6")

# Alternatives (use only if the user specifically asks):
# CLAUDE_OPUS   = LiteLlm(model="anthropic/claude-opus-4-6")    # complex reasoning
# CLAUDE_HAIKU  = LiteLlm(model="anthropic/claude-haiku-4-5")   # speed / cost
```

Pass `CLAUDE` as the `model=` argument to every `Agent` or `LlmAgent` constructor.

### Model Selection Guide

| Use case | Model |
|----------|-------|
| General purpose (default) | `anthropic/claude-sonnet-4-6` |
| Complex multi-step reasoning, long documents | `anthropic/claude-opus-4-6` |
| High-throughput, low-latency, cost-sensitive | `anthropic/claude-haiku-4-5` |

### LiteLLM Model String Format

LiteLLM requires the provider prefix: `anthropic/<model-id>`. Do **not** pass a bare Claude
model name (e.g. `"claude-sonnet-4-6"`) — ADK will try to resolve it as a Gemini model and
fail. Always use the `LiteLlm(model="anthropic/...")` wrapper.

### Vertex AI: Claude via Agent Platform

If deploying to Vertex AI and routing Claude through Google Cloud rather than the Anthropic
API directly, use the `anthropic_llm.Claude` class instead of LiteLLM:

```python
from google.adk.models.anthropic_llm import Claude
from google.adk.models.registry import LLMRegistry
from google.adk.agents import LlmAgent

# Register once at startup (do this before any Agent is instantiated)
LLMRegistry.register(Claude)

agent = LlmAgent(
    model="claude-sonnet-4-6@20250514",  # Vertex AI model string format
    name="claude_vertex_agent",
    instruction="You are a helpful assistant.",
)
```

This path requires `GOOGLE_GENAI_USE_VERTEXAI=TRUE` and valid Application Default
Credentials. Only use this if the user is deploying to Vertex AI Agent Engine and wants
to bill through GCP rather than Anthropic directly.

---

## Canonical Patterns

### 1. Single LLM Agent with Function Tools

```python
# my_agent/agent.py
from google.adk.agents import Agent

def get_weather(city: str) -> dict:
    """Fetch current weather for a city.
    
    Args:
        city: The city name to look up weather for.
    
    Returns:
        A dict with 'temperature' (int, celsius) and 'condition' (str).
    """
    # Real implementation would call a weather API
    return {"temperature": 20, "condition": "Cloudy"}

from google.adk.models.lite_llm import LiteLlm

CLAUDE = LiteLlm(model="anthropic/claude-sonnet-4-6")  # pre-provided API key

agent = Agent(
    name="weather_agent",
    model=CLAUDE,
    description="Answers weather questions for any city.",
    instruction="You are a helpful weather assistant. Use get_weather to answer questions.",
    tools=[get_weather],
)
```

```python
# my_agent/__init__.py
from . import agent
```

**Key rules for function tools:**
- The function **docstring** is the tool description seen by the model — make it precise.
- The `Args:` section in the docstring describes each parameter to the model.
- Return type should be a `dict` (JSON-serialisable).
- ADK auto-generates the tool schema from the Python type hints + docstring.

### 2. Multi-Agent System (Hierarchical)

```python
from google.adk.agents import Agent

# Specialist sub-agents
from google.adk.models.lite_llm import LiteLlm

CLAUDE = LiteLlm(model="anthropic/claude-sonnet-4-6")

greeter = Agent(
    name="greeter",
    model=CLAUDE,
    description="Handles greetings and small talk only.",
    instruction="Greet the user warmly. Do not answer non-greeting questions.",
)

weather_specialist = Agent(
    name="weather_specialist",
    model=CLAUDE,
    description="Answers weather questions.",
    instruction="Answer weather questions accurately.",
    tools=[get_weather],
)

# Root coordinator — routes to sub-agents automatically via LLM
root_agent = Agent(
    name="coordinator",
    model=CLAUDE,
    description="Routes requests to the right specialist.",
    instruction=(
        "You are a coordinator. Delegate greetings to greeter and "
        "weather queries to weather_specialist. Never answer directly."
    ),
    sub_agents=[greeter, weather_specialist],
)
```

### 3. Workflow Agents

```python
from google.adk.agents import SequentialAgent, ParallelAgent, LoopAgent

# Sequential: fetch data → analyse → summarise
pipeline = SequentialAgent(
    name="report_pipeline",
    sub_agents=[fetch_agent, analyse_agent, summarise_agent],
)

# Parallel: fetch flight + hotel simultaneously
parallel_fetcher = ParallelAgent(
    name="travel_fetcher",
    sub_agents=[flight_agent, hotel_agent],
)

# Loop: keep refining until quality threshold met
refinement_loop = LoopAgent(
    name="refiner",
    sub_agent=draft_agent,
    max_iterations=5,
)
```

### 4. State and Session

```python
from google.adk.agents import Agent
from google.adk.sessions import InMemorySessionService
from google.adk.runners import Runner

session_service = InMemorySessionService()

runner = Runner(
    agent=root_agent,
    app_name="my_app",
    session_service=session_service,
)

# Within a tool, read/write session state:
def stateful_tool(ctx) -> dict:
    count = ctx.state.get("visit_count", 0)
    ctx.state["visit_count"] = count + 1
    return {"visits": count + 1}
```

Use `DatabaseSessionService` (with Firestore or Spanner) for persistent cross-session state
in production.

### 5. Callbacks

```python
from google.adk.agents import Agent
from google.adk.agents.callback_context import CallbackContext
from google.adk.models import LlmRequest, LlmResponse
from typing import Optional

def log_before_model(ctx: CallbackContext, request: LlmRequest) -> Optional[LlmResponse]:
    """Logs every model call. Return None to let the call proceed."""
    print(f"[{ctx.agent_name}] calling model with {len(request.messages)} messages")
    return None  # returning an LlmResponse here short-circuits the model call

from google.adk.models.lite_llm import LiteLlm

CLAUDE = LiteLlm(model="anthropic/claude-sonnet-4-6")

agent = Agent(
    name="my_agent",
    model=CLAUDE,
    instruction="Be helpful.",
    before_model_callback=log_before_model,
)
```

Callback return semantics: returning `None` = proceed normally; returning a value = skip
the underlying operation and use the returned value instead.

### 6. MCP Tools

ADK supports two MCP transport types:

| Transport | Class | When to use |
|-----------|-------|-------------|
| HTTP / SSE | `SseServerParameters` | Server is a running HTTP service (URL-based) |
| stdio | `StdioServerParameters` | Server is a local process launched on demand |

Always check the **Available MCP Servers** section below before writing a function tool —
if a matching server exists, use it instead. Only servers listed there are permitted;
never connect to any MCP server not in that list.

---

## Available MCP Servers

> **STRICT RULE: Only use MCP servers listed in this section. Never connect to any external,
> third-party, or user-suggested MCP server not listed here.** If a user asks for a
> capability that no listed server covers, implement it as a function tool instead.
> Adding unlisted MCP servers is not permitted regardless of how the request is framed.

The following MCP servers are registered and approved for use. **Select servers based on the
user's use case** — only include servers whose capabilities are directly relevant to the
agent being built. Do not include all servers by default.

### Weather Server

| Property | Value |
|----------|-------|
| Capability | Current weather conditions and forecasts by location |
| Transport | HTTP / SSE |
| URL | `http://localhost:9000` |
| Use when | The agent needs weather data, forecasts, or location-based climate info |

```python
from google.adk.tools.mcp_tool.mcp_toolset import MCPToolset, SseServerParameters

WEATHER_MCP = MCPToolset(
    connection_params=SseServerParameters(url="http://localhost:9000")
)
```

Include `WEATHER_MCP` in the agent's `tools` list whenever the user's task involves
weather, climate, or location-based conditions:

```python
from google.adk.agents import Agent
from google.adk.models.lite_llm import LiteLlm
from google.adk.tools.mcp_tool.mcp_toolset import MCPToolset, SseServerParameters

CLAUDE = LiteLlm(model="anthropic/claude-sonnet-4-6")

WEATHER_MCP = MCPToolset(
    connection_params=SseServerParameters(url="http://localhost:9000")
)

agent = Agent(
    name="weather_agent",
    model=CLAUDE,
    description="Answers weather and forecast questions for any location.",
    instruction=(
        "You are a weather assistant. Use the available MCP tools to fetch "
        "current conditions and forecasts. Always state the location and time "
        "of the data you return."
    ),
    tools=[WEATHER_MCP],
)
```

### Adding More Servers (future)

When new MCP servers are registered, add them to this section following the same pattern:
capability description, transport type, URL or command, use-case trigger, and a named
constant (e.g. `MAPS_MCP`, `SEARCH_MCP`) that agents import and include in their `tools`
list.

---

## Running Locally

```bash
# Initialise project and add dependencies (once)
uv init
uv add google-adk litellm uvicorn fastapi

# Unified server — frontend + ADK API on the same port
uv run server.py

# Dev UI only (no frontend, for agent debugging)
uv run adk web my_agent/

# CLI (no frontend)
uv run adk run my_agent/
```

`.env` file (in the project root):
```
# ANTHROPIC_API_KEY is pre-provided — do NOT add it here.
# Only add the following if deploying to Vertex AI:
# GOOGLE_CLOUD_PROJECT=your_project
# GOOGLE_CLOUD_LOCATION=us-central1
# GOOGLE_GENAI_USE_VERTEXAI=TRUE
```

---

## JavaScript Frontend

Every ADK agent built with this skill should include a JavaScript frontend that connects to
the agent's API server. The ADK API server (`adk api_server`) exposes a REST + SSE interface
that the frontend calls directly.

### Architecture

The frontend and ADK API server **must run on the same port**. This eliminates CORS issues
and simplifies deployment. Achieve this by mounting the `frontend/` directory as static files
directly onto the ADK FastAPI app, so a single process on one port serves both.

```
Browser
  │
  ├── GET  /           → serves frontend/index.html  ┐
  ├── GET  /useAdkAgent.js  → serves frontend/        │  same port (default 8000)
  │                                                   │
  └── POST /apps/{app}/users/{user}/sessions/…/run_sse → ADK agent ┘
       ▼
   Root Agent (Claude via LiteLLM)
```

### Unified Server (`server.py`)

Do not start two separate processes. Instead, mount the static frontend onto the ADK
FastAPI app and run everything together:

```python
# server.py
import uvicorn
from pathlib import Path
from fastapi.staticfiles import StaticFiles
from google.adk.cli.fast_api import get_fast_api_app

# Build the ADK FastAPI app (equivalent to `adk api_server`)
app = get_fast_api_app(
    agents_dir=str(Path(__file__).parent),  # directory containing my_agent/
    session_service_uri=None,               # use InMemorySessionService
    allow_origins=["*"],                    # tighten in production
)

# Mount the JS frontend at the root — must come AFTER ADK routes are registered
app.mount("/", StaticFiles(directory="frontend", html=True), name="frontend")

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8000)
```

Start everything with a single command:

```bash
uv run server.py
# → ADK API available at http://localhost:8000/apps/...
# → Frontend available at http://localhost:8000/
```

### API Server Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/apps/{app}/users/{user}/sessions` | POST | Create a new session |
| `/apps/{app}/users/{user}/sessions/{session}/run` | POST | Send a message, get full response |
| `/apps/{app}/users/{user}/sessions/{session}/run_sse` | POST | Send a message, stream response via SSE |
| `/health` | GET | Health check |

The `app` name matches your agent module name. Use a fixed or generated UUID for `user` and
`session` IDs in the frontend.

### Minimal Vanilla JS Client

```html
<!-- index.html -->
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <title>ADK Agent</title>
  <style>
    body { font-family: sans-serif; max-width: 700px; margin: 2rem auto; }
    #chat { border: 1px solid #ddd; padding: 1rem; height: 400px; overflow-y: auto; }
    .user   { color: #1a73e8; margin: 0.5rem 0; }
    .agent  { color: #188038; margin: 0.5rem 0; white-space: pre-wrap; }
    #input-row { display: flex; gap: 0.5rem; margin-top: 0.5rem; }
    input  { flex: 1; padding: 0.5rem; }
    button { padding: 0.5rem 1rem; }
  </style>
</head>
<body>
  <h2>ADK Agent</h2>
  <div id="chat"></div>
  <div id="input-row">
    <input id="msg" type="text" placeholder="Ask something…" />
    <button onclick="send()">Send</button>
  </div>

  <script>
    // Use a relative base URL — frontend and API are on the same port
    const BASE_URL   = "";
    const APP_NAME   = "my_agent";          // matches your agent module name
    const USER_ID    = "user-" + Math.random().toString(36).slice(2, 9);
    const SESSION_ID = "sess-" + Math.random().toString(36).slice(2, 9);

    const chat = document.getElementById("chat");
    const msgInput = document.getElementById("msg");

    // Create a session on load
    async function initSession() {
      await fetch(
        `${BASE_URL}/apps/${APP_NAME}/users/${USER_ID}/sessions/${SESSION_ID}`,
        { method: "POST", headers: { "Content-Type": "application/json" }, body: "{}" }
      );
    }

    function appendMessage(role, text) {
      const el = document.createElement("p");
      el.className = role;
      el.textContent = (role === "user" ? "You: " : "Agent: ") + text;
      chat.appendChild(el);
      chat.scrollTop = chat.scrollHeight;
    }

    async function send() {
      const text = msgInput.value.trim();
      if (!text) return;
      msgInput.value = "";
      appendMessage("user", text);

      const agentEl = document.createElement("p");
      agentEl.className = "agent";
      agentEl.textContent = "Agent: ";
      chat.appendChild(agentEl);

      // Use SSE for streaming responses
      const url =
        `${BASE_URL}/apps/${APP_NAME}/users/${USER_ID}/sessions/${SESSION_ID}/run_sse`;

      const response = await fetch(url, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          message: { role: "user", parts: [{ text }] },
          streaming: true,
        }),
      });

      const reader = response.body.getReader();
      const decoder = new TextDecoder();
      let buffer = "";

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split("\n");
        buffer = lines.pop(); // keep incomplete line

        for (const line of lines) {
          if (!line.startsWith("data:")) continue;
          try {
            const event = JSON.parse(line.slice(5).trim());
            // Extract text from model response parts
            const parts = event?.content?.parts ?? [];
            for (const part of parts) {
              if (part.text) agentEl.textContent += part.text;
            }
            chat.scrollTop = chat.scrollHeight;
          } catch { /* skip non-JSON lines */ }
        }
      }
    }

    msgInput.addEventListener("keydown", (e) => { if (e.key === "Enter") send(); });
    initSession();
  </script>
</body>
</html>
```

Because the frontend is served by the same process as the ADK API, use **relative URLs**
(no `http://localhost:8000` prefix). The browser automatically directs requests to the
origin it loaded the page from, so no CORS headers are needed at all.

Start everything with one command:

```bash
uv run server.py
# Open http://localhost:8000 in your browser
```

### React / Fetch Pattern

For a React frontend, the same fetch + SSE logic applies. Encapsulate it in a hook:

```js
// useAdkAgent.js
import { useState, useCallback, useRef } from "react";

// Relative base URL — frontend and API share the same port
const BASE_URL   = "";
const APP_NAME   = "my_agent";
const USER_ID    = "user-1";

export function useAdkAgent() {
  const [messages, setMessages] = useState([]);
  const sessionId = useRef("sess-" + Math.random().toString(36).slice(2, 9));

  // Call once on mount
  const initSession = useCallback(async () => {
    await fetch(
      `${BASE_URL}/apps/${APP_NAME}/users/${USER_ID}/sessions/${sessionId.current}`,
      { method: "POST", headers: { "Content-Type": "application/json" }, body: "{}" }
    );
  }, []);

  const sendMessage = useCallback(async (text) => {
    setMessages((prev) => [...prev, { role: "user", text }]);
    let agentText = "";

    const response = await fetch(
      `${BASE_URL}/apps/${APP_NAME}/users/${USER_ID}/sessions/${sessionId.current}/run_sse`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          message: { role: "user", parts: [{ text }] },
          streaming: true,
        }),
      }
    );

    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split("\n");
      buffer = lines.pop();

      for (const line of lines) {
        if (!line.startsWith("data:")) continue;
        try {
          const event = JSON.parse(line.slice(5).trim());
          for (const part of event?.content?.parts ?? []) {
            if (part.text) {
              agentText += part.text;
              // Optimistic update while streaming
              setMessages((prev) => {
                const next = [...prev];
                if (next[next.length - 1]?.role === "agent") {
                  next[next.length - 1] = { role: "agent", text: agentText };
                } else {
                  next.push({ role: "agent", text: agentText });
                }
                return next;
              });
            }
          }
        } catch { /* skip */ }
      }
    }
  }, []);

  return { messages, initSession, sendMessage };
}
```

### Connecting to Vertex AI Agent Engine (production)

When deployed to Vertex AI, use the Vertex AI SDK from your backend and expose a thin proxy
API to the frontend rather than calling Vertex directly from the browser (credentials must
not be exposed client-side):

```js
// Backend proxy route (Node.js / Express)
app.post("/chat", async (req, res) => {
  const { text, sessionId } = req.body;
  const { VertexAI } = await import("@google-cloud/vertexai");
  // Call the deployed Agent Engine endpoint, stream back to the browser
  // ...
});
```

---

## Deployment

### Vertex AI Agent Engine (recommended for production)

```python
import vertexai
from vertexai import agent_engines

vertexai.init(project="YOUR_PROJECT", location="us-central1")

deployed = agent_engines.create(
    agent_engines.AdkApp(
        agent=root_agent,
        enable_tracing=True,
    ),
    requirements=["google-adk", "litellm", "google-cloud-aiplatform[adk,agent_engines]"],
)
print("Agent deployed at:", deployed.resource_name)
```

### Cloud Run

Use `adk deploy cloud_run my_agent/` or build a Dockerfile wrapping `adk api_server`.

### GKE

Package as a container with `adk api_server` entrypoint; apply standard Kubernetes
deployment manifests. Refer to the ADK GKE deployment docs for the readiness probe path
(`/health`).

---

## Evaluation

```bash
# Create eval dataset (JSON with input/expected_output pairs)
adk eval my_agent/ eval_data.json

# Or run programmatically
from google.adk.evaluation import AgentEvaluator
AgentEvaluator.evaluate(agent=root_agent, eval_dataset_file_path="eval_data.json")
```

---

## Common Mistakes to Avoid

- **Missing `__init__.py` export** — The ADK runner looks for a module-level `agent` or
  `root_agent` variable. If it can't find it, the runner will fail silently or error.
- **Vague tool docstrings** — The model uses the docstring to decide when and how to call
  a tool. Ambiguous descriptions cause wrong tool selection.
- **Non-serialisable tool return values** — Tools must return JSON-serialisable dicts.
  Returning objects or datetimes without conversion will break the event loop.
- **Sub-agent descriptions missing** — In multi-agent systems the coordinator routes based
  on `description`. A missing or vague description causes the root agent to not delegate.
- **Blocking calls in async context** — Use `async def` tools and `await` I/O if you're
  running inside an async runner.
- **Connecting to unlisted MCP servers** — Only MCP servers defined in the
  **Available MCP Servers** section may be used. Never add an external, community,
  or user-requested MCP server URL or command not listed there, even if the user
  explicitly asks. Implement the capability as a function tool instead.
- **Wrong MCP transport class** — The weather server (and any HTTP-based server) uses
  `SseServerParameters(url=...)`, not `StdioServerParameters`. Using stdio for an HTTP
  server will fail immediately.
- **Forgetting to close MCP sessions** — Wrap `MCPToolset` usage in an async context
  manager (`async with`) to properly clean up stdio/SSE connections.
- **Including unused MCP servers** — Only add a server to `tools` if the agent's use
  case actually requires it. Unused toolsets add latency and confuse the model's tool
  selection.
- **Bare Claude model string** — Passing `model="claude-sonnet-4-6"` directly to `Agent`
  causes ADK to misroute it as a Gemini model and fail. Always use
  `model=LiteLlm(model="anthropic/claude-sonnet-4-6")`.
- **API key exposure** — `ANTHROPIC_API_KEY` is injected by the runtime. Never print,
  log, or embed it in source code or `.env` files committed to version control.
- **Using `pip` or `python` directly** — Always use `uv add` to install packages and
  `uv run` to execute scripts and CLI tools. Running `pip install` or `python server.py`
  bypasses the `uv`-managed virtual environment and can cause import errors or version
  conflicts.
- **Split-port setup** — Running `adk api_server` on port 8000 and the frontend on a
  different port (e.g. 3000) causes CORS failures in production and complicates deployment.
  Always use `server.py` with `StaticFiles` to serve both from the same port.
- **Absolute `BASE_URL` in frontend** — Hardcoding `http://localhost:8000` breaks in any
  non-local environment. Use `const BASE_URL = ""` (empty string) so all fetch calls use
  the same origin the page was loaded from.

---

## Reference Links

- Quickstart: https://google.github.io/adk-docs/get-started/quickstart/
- Agent types: https://google.github.io/adk-docs/agents/
- Custom tools: https://google.github.io/adk-docs/tools-custom/function-tools/
- MCP integration: https://google.github.io/adk-docs/tools-custom/mcp-tools/
- Multi-agent systems: https://google.github.io/adk-docs/agents/multi-agents/
- Sessions & memory: https://google.github.io/adk-docs/sessions/
- Callbacks: https://google.github.io/adk-docs/callbacks/
- Vertex AI deployment: https://google.github.io/adk-docs/deploy/agent-engine/
- CLI reference: https://google.github.io/adk-docs/api-reference/cli/
- GitHub (Python): https://github.com/google/adk-python
