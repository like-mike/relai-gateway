# RelAI – The Go-Native LLM Gateway

RelAI: - Go-native LLM gateway built for infra teams.

---

## 🚀 Why RelAI?

Most AI proxies are:
- Slow (Python-based)
- Messy (bad prompt control)
- Opaque (no metrics or logs)
- Expensive (vendor lock-in, no caching)

**RelAI** solves that by being:
- 🧠 **LLM-native** – Routes OpenAI, Anthropic, etc.
- ⚡ **Fast as hell** – Built in Go, optimized for streaming
- 🔐 **Enterprise-ready** – Azure AD OAuth, RBAC, Redis
- 📊 **Fully observable** – Prometheus, Grafana, OpenTelemetry
- 🧼 **Prompt-aware** – Middleware for logging, redaction, rewrites

---

## 🧱 Roadmap

### ✅ MVP Goals (Week 1)
- [x] Go + Fiber backend scaffold
- [x] `/v1/chat/completions` endpoint proxy
- [x] Support OpenAI-style streaming (`stream: true`)
- [x] Redis token cache + request logging
- [x] Basic metrics via Prometheus
- [x] HTMX-powered UI for viewing API keys (stub)
- [x] Azure OAuth login (stubbed)
- [x] Admin/User role separation (basic RBAC)

### 🔜 Short-Term Goals
- [ ] Prompt middleware system (PII redaction, rewriting, logging)
- [ ] Multi-provider routing (OpenAI, Anthropic, Mistral)
- [ ] YAML config (models, providers, tokens, limits)
- [ ] Rate limiting + per-org quotas
- [ ] Token usage + cost dashboard
- [ ] Org-aware RBAC and scoping

### 🧠 Stretch Goals
- [ ] Fallback logic (e.g. OpenAI failover to Anthropic)
- [ ] UI for logs, token usage, cost projection
- [ ] Pluggable backends (Ollama, Bedrock, Azure, etc.)
- [ ] Hosted OSS edition with managed UI

---

## ⚙️ Stack

| Layer | Tech |
|-------|------|
| Backend | Go + Fiber |
| UI | HTMX + Tailwind |
| Auth | Azure OAuth + AD Groups |
| Cache / RBAC | Redis |
| Observability | Prometheus + Grafana + OpenTelemetry |
| Config | YAML (TBD) |

---

## 🧠 Philosophy

RelAI is being built:
- In public (eventually open source)
- For real-world internal use
- With production needs in mind (auth, caching, metrics, prompt safety)

---

## 📣 Status

RelAI is in **early development**.  

Stay tuned.  
