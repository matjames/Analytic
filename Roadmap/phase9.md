# PHASE 9 — ARTIFICIAL INTELLIGENCE, MACHINE LEARNING & INTELLIGENT AUTOMATION

## Objective

Build the complete Artificial Intelligence ecosystem that powers every component of StatGate.

AI functions as an intelligent assistant — not a replacement — for statisticians, researchers, administrators, policymakers, and decision-makers.

Every module developed in previous phases must expose AI capabilities through a common AI Gateway.

---

## Vision

StatGate becomes Africa's first AI-powered Evidence Intelligence Platform.

AI assists users in generating insights, automating workflows, improving quality, accelerating research, and supporting evidence-based decisions. The platform uses Large Language Models, Retrieval-Augmented Generation (RAG), embeddings, and autonomous agents — all under human oversight with full audit trails.

---

## Services Involved

| Service | Technology | Port |
|---|---|---|
| Enterprise Core (AI layer) | Go + Gin | :8096 |
| Analytics UI (agentic engine) | Python / Flask | :5000 |

---

## What Exists ✅ (AI Infrastructure Already Built)

### AI Catalog & Schema (`enterprise/core/ai.go`)

```
GET    /api/ai/catalog             ← list AI capabilities available on the platform
GET    /api/ai/catalog/:app        ← AI capabilities for a specific application
GET    /api/ai/schema              ← AI data schema for LLM context preparation
```

### AI Investigations (`enterprise/core/ai_investigations.go`)

```
POST   /api/ai/investigations            ← start an AI investigation on an entity
GET    /api/ai/investigations            ← list AI investigations
GET    /api/ai/investigations/:id        ← investigation detail and results
POST   /api/ai/investigations/:id/run    ← run investigation step
GET    /api/ai/investigations/:id/report ← investigation report
```

### Phase XII Governed AI (`enterprise/core/phase12_ai*.go`)

```
POST   /api/ai/governed/analyze          ← AI analysis request (human-approval required)
GET    /api/ai/governed/requests         ← pending AI analysis requests
PUT    /api/ai/governed/requests/:id/approve  ← approve AI analysis
PUT    /api/ai/governed/requests/:id/reject   ← reject AI analysis
GET    /api/ai/governed/audit            ← AI audit trail (all AI actions logged)
GET    /api/ai/governed/lifecycle        ← AI model lifecycle events
```

**Governed AI features:**
- Every AI analysis request requires human approval before execution
- Full audit trail of all AI requests, approvals, and outputs
- Model lifecycle management (register, deprecate, retire)
- AI governance monitoring (bias detection hooks, content moderation hooks)

### Agentic Engine (Flask — `frontend/agentic_engine.py`)

- Rule-based automated report generation
- Agent feedback loops — AI evaluates its own report quality and iterates
- Triggered by scheduled jobs or user request
- Outputs persisted to `data/agent_feedback/`

### Schema Healer (`frontend/schema_healer.py`)

- Detects schema anomalies (missing columns, type mismatches, naming inconsistencies)
- Suggests corrective actions
- Operates on datasets registered in the analytics workspace

### Analytics AI Integration (`enterprise/core/analytics_ai.go`)

```
GET    /api/analytics/ai/insights        ← AI-generated insights for a dataset
POST   /api/analytics/ai/analyze         ← trigger AI analysis on dataset
```

---

## What is Missing ❌ — Core AI Infrastructure

### LLM Gateway (Priority 1)

The most critical missing component. The platform has no connection to any Large Language Model.

**Must build:**
- LLM Gateway service — routes requests to OpenAI GPT, Anthropic Claude, or local Ollama
- Configurable via environment variables (`LLM_PROVIDER`, `LLM_API_KEY`, `LLM_MODEL`)
- Rate limiting, retry, cost tracking per tenant
- Prompt template registry — versioned prompt templates for each use case
- Streaming response support for chat interfaces

```
POST   /api/ai/llm/complete          ← completion request (non-streaming)
POST   /api/ai/llm/stream            ← streaming completion (SSE)
GET    /api/ai/llm/models            ← available models
GET    /api/ai/llm/usage             ← usage and cost tracking
```

### RAG (Retrieval-Augmented Generation) Pipeline (Priority 1)

The platform has no vector search or semantic retrieval.

**Must build:**
- Embedding service — converts text to vectors (via OpenAI `text-embedding-3-small` or local model)
- Vector store — PostgreSQL with `pgvector` extension, or dedicated vector DB (Qdrant/Chroma)
- Chunking pipeline — split documents, knowledge articles, and datasets into chunks
- Retrieval service — semantic similarity search given a query
- RAG orchestrator — retrieve relevant context → augment prompt → call LLM → return response

```
POST   /api/ai/embed                 ← embed text into vector
POST   /api/ai/search/semantic       ← semantic similarity search
POST   /api/ai/rag/query             ← RAG query (retrieve + generate)
POST   /api/ai/rag/index/:source     ← index a data source into vector store
```

### StatGPT Interface (Priority 2)

The flagship AI chat interface. Not yet built.

**Must build:**
- `StatGPT/` — React/Vite frontend (or integrate into App Launcher)
- Multi-turn conversation with history
- Context selection (which datasets / documents to include in RAG context)
- Conversation persistence and search
- Module-aware context switching (switch to "Research mode", "Statistics mode", etc.)

### AI Assistants (Per Module — Priority 2)

Each module needs an AI assistant endpoint wired to the LLM Gateway:

| Module | Assistant endpoint | Key capabilities |
|---|---|---|
| RMS | `POST /api/ai/research/assist` | Proposal writing, literature summaries, gap analysis |
| PMS | `POST /api/ai/project/assist` | Risk prediction, schedule advice, meeting summaries |
| StatCollect | `POST /api/ai/survey/assist` | Questionnaire design recommendations, data quality review |
| Analytics | `POST /api/ai/analytics/assist` | Insight generation, NLQ, dashboard narrative |
| StatGovernance | `POST /api/ai/governance/assist` | Policy interpretation, compliance monitoring |
| StatChat | `POST /api/ai/chat/assist` | Meeting summaries, action item extraction |

### Agent Framework (Priority 3)

**Must build:**
- Agent definition model — name, trigger, tools, LLM config, human-approval requirement
- Tool registry — what functions agents can call (database queries, API calls, report generation)
- Multi-step agent execution with step-level audit trail
- Agent Builder UI — configure and test agents without code
- Multi-agent collaboration — agents can call other agents as tools

```
GET    /api/ai/agents                ← list registered agents
POST   /api/ai/agents                ← create agent definition
POST   /api/ai/agents/:id/run        ← run agent
GET    /api/ai/agents/:id/runs       ← agent execution history
GET    /api/ai/agents/:id/runs/:runId ← run detail with step-level trace
```

### Model Registry (Priority 3)

**Must build:**
- Registry of LLM models, fine-tuned models, and embedding models
- Model versioning, performance benchmarks, deprecation
- Model performance dashboard
- Model routing rules (e.g., use cheaper model for summaries, expensive for analysis)

### Intelligent Automation (Priority 3)

**Must build:**
- Automatic report generation (trigger on schedule or event)
- Automatic metadata creation (when a dataset is uploaded)
- Automatic risk detection (in PMS/RMS on change events)
- Automatic compliance monitoring (in StatGovernance on policy changes)
- Automatic survey generation from research protocols
- Automatic meeting minutes from StatChat call recordings

---

## AI Governance (Already Built — Phase XII)

The following AI governance infrastructure exists in `enterprise/core/phase12_ai*.go`:

- Human approval required before any AI analysis executes ✅
- Full audit trail of AI requests, approvals, and outputs ✅
- Model lifecycle management ✅
- AI governance monitoring hooks ✅

**Still missing in governance:**
- Bias detection pipeline
- Content moderation classifier
- Explainability output (LIME / SHAP or equivalent)
- AI ethics review workflow for new model onboarding

---

## Implementation Order

1. **LLM Gateway** — foundation for everything; must be built first
2. **Embedding Service + Vector Store** — enables RAG; build immediately after gateway
3. **RAG Orchestrator** — connects search + LLM; enables all AI assistants
4. **StatGPT Interface** — flagship visible feature
5. **Module-specific AI assistants** — wire each module to LLM Gateway
6. **Agent Framework** — advanced automation
7. **Model Registry** — operations and monitoring

---

## Environment Variables Required

```env
LLM_PROVIDER=openai          # openai | anthropic | ollama
LLM_API_KEY=sk-...
LLM_MODEL=gpt-4o-mini
LLM_EMBEDDING_MODEL=text-embedding-3-small
VECTOR_DB_URL=postgres://...  # pgvector in main PostgreSQL
AI_HUMAN_APPROVAL_REQUIRED=true
AI_MAX_TOKENS=4096
AI_TEMPERATURE=0.2
```

---

## Acceptance Criteria

- [x] AI catalog and schema APIs operational
- [x] AI investigations framework operational
- [x] Governed AI (human approval, audit trail, lifecycle) operational
- [x] Agentic report engine (rule-based) operational
- [x] Schema healer operational
- [ ] LLM Gateway operational (routes to OpenAI / Anthropic / Ollama)
- [ ] Embedding service operational
- [ ] Vector store (pgvector) operational
- [ ] RAG query pipeline operational
- [ ] StatGPT interface operational
- [ ] Research AI assistant operational
- [ ] Project AI assistant operational
- [ ] Survey design AI assistant operational
- [ ] Analytics NLQ (natural language queries) operational
- [ ] Agent framework with Agent Builder UI operational
- [ ] Model Registry operational
- [ ] AI integrated into all previous modules

---

## Ports & Services

| Component | Port |
|---|---|
| Enterprise Core — AI layer (Go) | :8096 |
| Analytics UI — agentic engine (Flask) | :5000 |
| StatGPT UI (React/Vite — to build) | :3013 (proposed) |

---

## Estimated Duration

14 weeks

## Milestone

Enterprise AI Platform complete. Every module has an AI assistant powered by RAG. StatGPT is the platform-wide intelligence interface.