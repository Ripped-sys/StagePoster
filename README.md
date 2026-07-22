# StagePoster Backend

StagePoster is an AI-native poster production backend for music events, festivals, bands, performances, and promotional campaigns.

It combines:

- A conversational requirement-gathering agent
- AI-generated visual design plans
- ComfyUI candidate image generation
- Deterministic Go-based typography and layout composition
- Multimodal VLM visual review
- Automatic recompose or regenerate loops
- Review snapshots and best-so-far restoration
- Downloadable final posters and thumbnails

## Core User Flow

```text
Create AI Session
        ↓
Chat continuously with the requirement agent
        ↓
missingFields gradually becomes empty
        ↓
Receive multiple design plans
        ↓
Confirm one plan
        ↓
Generate three visual candidates
        ↓
Select one candidate
        ↓
Compose the final poster
        ↓
VLM visual review
        ↓
RECOMPOSE / REGENERATE / ACCEPT
        ↓
Restore the best review round
        ↓
Download final poster
Architecture
Frontend
   |
   | HTTPS / JSON
   v
Cloudflare Quick Tunnel
   |
   v
Go Backend :8080
   |
   +---------------------------+
   |                           |
   v                           v
ComfyUI :8188              vLLM :8001
Z-Image-Turbo              Qwen3.5-9B
Image generation           Planning and visual review
Only the Go backend should be exposed publicly.
ComfyUI and vLLM should remain bound to localhost during normal deployment.
Verified Environment
The current golden E2E environment was verified with:
Component	Version or configuration
Operating system	Ubuntu 24.04.4 LTS
Kernel	Linux 6.8
GPU architecture	AMD gfx1100
ROCm	7.2.1
HIP	7.2.53211
Go	1.25.0
ComfyUI Python	3.10.20
ComfyUI Torch	2.13.0+rocm7.2
ComfyUI torchvision	0.28.0+rocm7.2
ComfyUI torchaudio	2.11.0+rocm7.2
vLLM Python	3.12.3
VLM	Qwen3.5-9B
Image model	Z-Image-Turbo BF16
ComfyUI commit	c9602625e445e9ee37d3ac6faf5ea9ec1e0de87e
Database	SQLite

The exact vLLM package version and image-provided PyTorch wheel build should be exported from the working instance with:
./scripts/export-runtime-locks.sh
Services
Service	Default address	Public
Go backend	127.0.0.1:8080	Through Cloudflare Tunnel
ComfyUI	127.0.0.1:8188	No
vLLM	127.0.0.1:8001	No

Quick Start on the Existing AMD Instance
cd /workspace/poster-engine/backend

cp .env.example .env
nano .env
chmod +x scripts/*.sh
./scripts/start-all.sh
./scripts/status.sh
./scripts/smoke-test.sh

Start a temporary development tunnel:

```bash
./scripts/start-dev-tunnel.sh
The generated URL will look like:
https://random-words.trycloudflare.com
Stop all services:
./scripts/stop-all.sh
Fresh Environment Bootstrap
The scripts assume that AMD ROCm drivers are already installed and working.
cd /workspace/poster-engine/backend

chmod +x scripts/*.sh

./scripts/bootstrap.sh
cp .env.example .env
nano .env
./scripts/download-models.sh
./scripts/start-all.sh
./scripts/smoke-test.sh

## Important Paths

```text
/workspace/poster-engine/
├── .venv-vllm/
├── ComfyUI/
├── models/
│   └── Qwen3.5-9B/
├── venv/
├── workflows/
│   └── z_image_poster_v1.json
└── backend/
    ├── data/
    ├── docs/
    ├── logs/
    ├── run/
    ├── scripts/
    ├── storage/
    └── poster-backend
Required Workflow
The current backend expects:
/workspace/poster-engine/workflows/z_image_poster_v1.json
Current node mapping:
Prompt node: 57:27
Seed node:   57:3
Any workflow update must also update the corresponding environment variables.
Golden E2E Result
The following complete flow has been verified against the real AMD GPU:
AI Session creation
Design plan generation
Plan confirmation
Three ComfyUI candidates
Candidate selection
Final Go composition
VLM Review Round 1
Automatic RECOMPOSE
Review snapshot
VLM Review Round 2
Best-so-far restoration
Finalize
Idempotent Finalize retry
VLM sleep and VRAM release
Cloudflare public access
The finalization retry was verified to:
Return HTTP 200
Keep the review count unchanged
Keep the final image SHA-256 unchanged
Avoid duplicate composition and duplicate review rounds
Frontend Integration
Frontend developers should begin with:
docs/frontend-integration.md
docs/api-reference.md
docs/conversation-flow.md
The public development base URL changes whenever the Quick Tunnel restarts.
Example:
NEXT_PUBLIC_API_BASE_URL=https://example.trycloudflare.com
Documentation
docs/
├── architecture.md
├── api-reference.md
├── cloudflare-tunnel.md
├── conversation-flow.md
├── deployment.md
├── frontend-integration.md
├── model-downloads.md
├── reproduction.md
└── troubleshooting.md
Security
Never expose these directly to the internet:
vLLM :8001
ComfyUI :8188
SQLite database
storage directory
VLM development sleep/wake endpoints
Never commit:
.env
HF_TOKEN
VLM_API_KEY
POSTER_API_TOKEN
Cloudflare tunnel token
data/poster.db*
storage/
logs/
run/
poster-backend
Development Tunnel
The current development URL is temporary:
https://cst-holmes-climate-charge.trycloudflare.com
It will stop working when:
The server instance shuts down
The cloudflared process stops
The Quick Tunnel is recreated
Do not hard-code this URL in production.
License
Add the project license before public release.
