# StagePoster — Claude Code Guide

> Project root: `/workspace/poster-engine/`
> Go module: `github.com/Ripped-sys/StagePoster/backend`
> Target hardware: AMD Radeon PRO W7900 48 GB + ROCm 7.2

---

## 1. Project Overview

StagePoster is an **AI-native music event poster engine**. Core pipeline:

```
Structured Event Brief
        ↓
Qwen3.5-9B Art Direction Agent
        ↓
3 Structured Design Plans
        ↓
ComfyUI + Z-Image Turbo → 3 Candidate Images
        ↓
User selects a candidate
        ↓
Go Deterministic Text / Logo / Info Layout
        ↓
Qwen Vision Review + Limited-Round Auto Optimization
        ↓
Final Poster + Thumbnail + Review Evidence
```

The frontend communicates only with the Go Backend. ComfyUI and vLLM are
internal services, never exposed to the browser.

---

## 2. Tech Stack

| Layer | Technology | Port |
|---|---|---|
| Go Backend | Go 1.25.0 + `net/http` | `:8080` |
| Database | SQLite (modernc.org/sqlite) | local file |
| Image Generation | ComfyUI + Z-Image Turbo | `:8188` |
| LLM / VLM | vLLM + Qwen3.5-9B | `:8001` |
| GPU Driver | ROCm 7.2 + HIP | — |
| Public Tunnel | Cloudflare Quick Tunnel | HTTPS |

Go dependencies are minimal — only `golang.org/x/image` and `modernc.org/sqlite`.
No external web framework.

---

## 3. Directory Structure

```
/workspace/poster-engine/
├── ComfyUI/                         # ComfyUI (git submodule)
│   └── models/
│       ├── diffusion_models/
│       │   └── z_image_turbo_bf16.safetensors
│       ├── text_encoders/
│       │   └── qwen_3_4b.safetensors
│       ├── vae/
│       │   └── ae.safetensors
│       └── loras/
│           └── z_image_turbo_distill_patch_lora_bf16.safetensors
├── models/
│   └── Qwen3.5-9B/                  # vLLM model files
├── workflows/
│   └── z_image_poster_v1.json       # ComfyUI workflow template
├── venv/                            # ComfyUI Python 3.10.20 env
├── .venv-vllm/                      # vLLM Python 3.12 env
├── .env                             # Project env vars
├── scripts/                         # Deploy and service scripts
│   ├── install-all.sh               # One-click deploy
│   ├── start-all.sh                 # Start all services
│   ├── stop-all.sh                  # Stop all services
│   ├── status.sh                    # Service status
│   ├── smoke-test.sh                # Smoke test
│   └── ...
└── backend/
    ├── cmd/server/
    │   └── main.go                  # Entry: service init + startup
    ├── data/
    │   └── poster.db                # SQLite database
    ├── logs/                        # Log directory
    ├── run/                         # PID files + tunnel URL
    ├── storage/
    │   ├── jobs/                    # ComfyUI task outputs
    │   ├── assets/                  # Uploaded asset storage
    │   └── posters/                 # Final poster outputs
    └── internal/                    # Go internal packages
        ├── api/                     # HTTP routes + handlers
        ├── domain/                  # Domain models + constants
        ├── repository/              # SQLite data access
        ├── service/                 # Core business logic
        ├── poster/                  # Poster flow orchestration
        ├── assistant/               # AI session state machine
        ├── ai/                      # vLLM client + Runtime
        ├── comfy/                   # ComfyUI client
        ├── composer/                # Poster layout engine
        ├── storage/                 # Filesystem abstraction
        └── worker/                  # Background goroutines
```

---

## 4. Core Flows

### 4.1 Poster Generation State Machine

```
planning_candidates → generating_candidates → validating_candidates
                                                       ↓
                                                 partial_ready (1/3 ready)
                                                       ↓
                                                 awaiting_selection
                                                       ↓
                                                 selected → composing
                                                       ↓
                                                 succeeded / failed / canceled
```

- `worker/reconciler.go`: polls ComfyUI job status every 2s, transitions
  `generating` → `ready`/`failed`
- `worker/poster_reconciler.go`: advances poster flow state machine, triggers
  composition, review, finalize
- **Key**: `PartialReady` is NOT a terminal state. The reconciler must continue
  processing remaining candidates.

### 4.2 AI Session State Machine

```
collecting_brief → planning → awaiting_approval → generating
                                                        ↓
                                                  awaiting_candidate_selection
                                                        ↓
                                                  selected → composing → reviewing
                                                        ↓
                                                  succeeded / failed / canceled
```

- `assistant/service.go`: manages AI session lifecycle
- User messages trigger LLM calls, which decide the next action (plan / confirm
  / select / finalize)

---

## 5. Development Workflow

### Build

```bash
cd /workspace/poster-engine/backend
go build -o poster-backend ./cmd/server
```

### Run Tests

```bash
cd /workspace/poster-engine/backend
go test ./...
```

### Start Services

```bash
cd /workspace/poster-engine
./scripts/start-all.sh          # Start ComfyUI + vLLM + Backend
./scripts/status.sh             # Check service status
```

### Smoke Test

```bash
cd /workspace/poster-engine
./scripts/smoke-test.sh          # Component smoke test
# Full E2E: see docs/one-click-deployment.md
```

### One-Click Deploy

```bash
cd /workspace/poster-engine
sudo -E bash scripts/install-all.sh
```

---

## 6. Environment Variables

Key variables from `.env` (project root):

| Variable | Default | Description |
|---|---|---|
| `LISTEN_ADDR` | `:8080` | Backend listen address |
| `COMFY_URL` | `http://127.0.0.1:8188` | ComfyUI URL |
| `VLM_URL` | `http://127.0.0.1:8001` | vLLM URL |
| `VLM_API_KEY` | `stageposter-vlm-local` | vLLM API key |
| `VLM_MODEL` | `stageposter-vlm` | vLLM model name |
| `DB_PATH` | `backend/data/poster.db` | SQLite path |
| `STORAGE_ROOT` | `backend/storage/jobs` | Task output dir |
| `WORKFLOW_PATH` | `workflows/z_image_poster_v1.json` | ComfyUI workflow |
| `WORKFLOW_KEY` | `poster-text` | Workflow identifier |
| `WORKFLOW_VERSION` | `1.0.0` | Workflow version |
| `POSTER_API_TOKEN` | `""` | API auth token |
| `CORS_ORIGIN` | `*` | CORS origin |
| `POSTER_FONT_REGULAR` | `""` | Regular font path |
| `POSTER_FONT_BOLD` | `""` | Bold font path |
| `RECONCILE_INTERVAL` | `2s` | Reconciler poll interval |
| `PROMPT_NODE_ID` | `57:27` | ComfyUI prompt node |
| `NEGATIVE_PROMPT_NODE_ID` | `""` | ComfyUI negative prompt node |
| `SEED_NODE_ID` | `57:3` | ComfyUI seed node |

---

## 7. Known Constraints and Pitfalls

### ROCm / GPU
- vLLM `--enable-sleep-mode` causes `CUDA Error: invalid argument` on ROCm 7.2
  + vLLM 0.20.0. **Do not enable.**
- ComfyUI and vLLM share a single W7900. Qwen wake-up automatically unloads
  ComfyUI models (`ReleaseComfyMemory`).
- GPU architecture is `gfx1100`. Verify with `rocm-smi`.

### ComfyUI
- Workflow node IDs are fixed, configured via `PROMPT_NODE_ID` / `SEED_NODE_ID`.
- After submitting a prompt, the caller receives a `jobID`. The reconciler
  polls `/history/{prompt_id}` until `completed` or `failed`.
- Output images are in ComfyUI's output dir, copied to `STORAGE_ROOT` via
  `storage.FileStore`.

### SQLite
- Uses `modernc.org/sqlite` (pure Go, no CGO).
- Migrations are in `repository/sqlite.go` `Migrate()`.
- All queries use `context.Context` with timeout.

### Concurrency
- Two reconciler goroutines tick every 2s.
- **Must have panic recovery** — otherwise goroutines die silently.
- Session-level operations use `sync.Mutex` to prevent races.

---

## 8. API Design Conventions

- All POST request bodies are JSON.
- Error responses: `{"error": "message"}` + appropriate HTTP status.
- Success responses: `{"data": ...}` or the resource object directly.
- Pagination: list endpoints return `items []` + `total int`.
- Auth: `Authorization: Bearer <POSTER_API_TOKEN>` (optional, recommended for
  production).

---

## 9. Code Style

- **Package names**: lowercase short names under `internal/` (`api`, `domain`,
  `service`, `poster`, `assistant`, etc.)
- **Errors**: define package-level sentinel errors with `errors.New`. Never
  compare error strings.
- **Context**: all long operations accept `context.Context`. HTTP handlers pass
  `r.Context()`.
- **Logging**: use stdlib `log`. Structured logging recommended for production.
- **ID generation**: use `domain.NewID(prefix)` for `prefix_xxxxxxxx` format.
- **JSON**: field names use `json:"snake_case"`. API request/response shapes are
  consistent.
- **File paths**: read absolute paths from env vars. Never hardcode.

---

## 10. Change Checklist

After modifying code:
- [ ] `go build ./cmd/server` compiles cleanly
- [ ] `go test ./...` passes
- [ ] Database changes → add migration in `repository/sqlite.go`
- [ ] New env vars → add to `.env.example` and `main.go` `env()` calls
- [ ] New ComfyUI node bindings → update node ID comments in `scripts/`
- [ ] Smoke test passes: `./scripts/smoke-test.sh`
