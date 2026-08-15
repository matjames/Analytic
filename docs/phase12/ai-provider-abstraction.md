# AI Provider Abstraction
## StatGate Phase XII — Sprint 0 Architecture Validation
**Document:** `docs/phase12/ai-provider-abstraction.md`
**Status:** APPROVED

---

## Ruling

No AI provider is hard-coded into Phase XII business logic. All AI capabilities are accessed through a defined interface. Provider implementations are swappable without changing the Intelligence Engine.

---

## AIProvider Interface (Conceptual — Go)

```go
// AIProvider is the governed interface through which all AI capabilities
// are accessed. No business logic may call a provider SDK directly.
type AIProvider interface {
    // Analyze produces structured analysis of input data.
    Analyze(ctx context.Context, req AIAnalysisRequest) (*AIAnalysisResult, error)

    // Summarize produces a human-readable summary of input content.
    Summarize(ctx context.Context, req AISummaryRequest) (*AISummaryResult, error)

    // Classify classifies input content into a set of predefined categories.
    Classify(ctx context.Context, req AIClassifyRequest) (*AIClassifyResult, error)

    // Recommend produces a governed recommendation from input signals.
    Recommend(ctx context.Context, req AIRecommendRequest) (*AIRecommendResult, error)

    // Forecast produces a probabilistic forecast from historical data.
    Forecast(ctx context.Context, req AIForecastRequest) (*AIForecastResult, error)

    // ProviderName returns the provider identifier (e.g., "gemini", "openai", "local").
    ProviderName() string

    // ProviderVersion returns the model/version identifier.
    ProviderVersion() string
}
```

All request types carry:
```go
type AIRequestBase struct {
    TenantID       string
    Actor          string
    CorrelationID  string
    DataClassification DataClassification
    Purpose        string
    InputEvidence  []string // canonical IDs of supporting evidence
}
```

All result types carry:
```go
type AIResultBase struct {
    ProviderName    string
    ProviderVersion string
    GeneratedAt     time.Time
    AIGenerated     bool   // always true
    Classification  string // FACT / INFERENCE / RECOMMENDATION / PREDICTION / UNKNOWN
    Confidence      string // HIGH / MEDIUM / LOW / UNKNOWN
    SupportingRefs  []string // canonical IDs referenced in output
}
```

---

## Permitted Providers

| Provider | Status | Notes |
|---|---|---|
| Google Gemini API | Candidate | Requires data classification gate |
| OpenAI API | Candidate | Requires data classification gate |
| Private hosted model | Candidate | Preferred for RESTRICTED/SENSITIVE data |
| Local inference | Candidate | For development and offline environments |

**Selection is deferred to Sprint 5.** The abstraction must be implemented first.

---

## Provider Registration

Providers are registered via configuration (environment variable or config file), not code:

```yaml
ai_provider:
  name: gemini
  model: gemini-2.0-flash
  endpoint: https://generativelanguage.googleapis.com
  max_classification: CONFIDENTIAL  # blocks RESTRICTED/SENSITIVE data
  timeout_seconds: 30
  audit_all_calls: true
```

---

## Audit Requirement

Every AI provider call must produce an audit record containing:
- Provider name + version
- Operation type
- Input data classification
- Tenant ID
- Actor
- Correlation ID
- Input token count (where available)
- Output classification (`FACT` / `INFERENCE` / `RECOMMENDATION` / `PREDICTION`)
- Latency
- Success / error status

This audit record is separate from the platform audit log but linked via `correlation_id`.

---

*Status: APPROVED*
*Sprint 0 Gate: AI ABSTRACTION — PASS*
