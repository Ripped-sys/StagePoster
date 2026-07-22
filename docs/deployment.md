# Deployment Guide

## Current Deployment Type

The current system runs on a single AMD GPU instance.

```text
Ubuntu 24.04
AMD gfx1100
ROCm 7.2.1
SQLite
Local filesystem storage
Process Layout
vLLM
PID file: run/vllm.pid
Log file: logs/vllm.log
Port: 8001

ComfyUI
PID file: run/comfyui.pid
Log file: logs/comfyui.log
Port: 8188
Go Backend
PID file: run/backend.pid
Log file: logs/backend.log
Port: 8080
Cloudflare Tunnel
PID file: run/cloudflared.pid
Log file: logs/cloudflared.log

## Build Backend

```bash
cd /workspace/poster-engine/backend

go mod download
go test ./...
go build -o poster-backend ./cmd/server

If the main package path differs, locate it with:

```bash
find cmd -name '*.go' -maxdepth 3 -print
Environment
cp .env.example .env
nano .env
Load it manually:
set -a
source .env
set +a
Start All Services
./scripts/start-all.sh
Check Status
./scripts/status.sh
Smoke Test
./scripts/smoke-test.sh
Stop
./scripts/stop-all.sh
Start Development Tunnel
./scripts/start-dev-tunnel.sh
Startup Order
The script follows:
1. vLLM
2. Put vLLM into sleep mode
3. ComfyUI
4. Go backend
5. Health checks
Sleeping vLLM before starting normal generation reduces GPU pressure.
Bind Addresses
Recommended:
vLLM:    127.0.0.1
ComfyUI: 127.0.0.1
Backend: 127.0.0.1 or :8080
When using Cloudflare Tunnel, the backend can remain bound to localhost.
Persistent Data
Back up:
backend/data/poster.db
backend/storage/
workflows/z_image_poster_v1.json
backend/.env
Do not include secrets in public backups.
SQLite
The current architecture is suitable for:
One backend process
One GPU worker
Hackathon demonstration
Development and small-team use
For multi-instance production deployment, replace SQLite and local storage with:
PostgreSQL
Object storage
Distributed job locks
Shared queue or workflow runtime
Log Inspection
tail -f logs/backend.log
tail -f logs/comfyui.log
tail -f logs/vllm.log
tail -f logs/cloudflared.log
Restart One Service
Backend:
kill "$(cat run/backend.pid)"
rm -f run/backend.pid
./scripts/start-all.sh
ComfyUI:
kill "$(cat run/comfyui.pid)"
rm -f run/comfyui.pid
./scripts/start-all.sh
vLLM:
kill "$(cat run/vllm.pid)"
rm -f run/vllm.pid
./scripts/start-all.sh
Production Recommendation
For a stable public endpoint:
Use a Cloudflare Named Tunnel
Use a fixed domain
Install cloudflared as a systemd service
Set a real POSTER_API_TOKEN
Restrict CORS
Put secrets in a secret manager
Add structured request IDs and logs
