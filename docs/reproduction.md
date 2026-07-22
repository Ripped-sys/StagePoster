# Official Reproduction Guide

## Target Environment

Verified reference:

```text
Ubuntu 24.04.4 LTS
AMD GPU with gfx1100
ROCm 7.2.1
HIP 7.2.53211
Go 1.25.0
Important Assumption
These scripts do not install the AMD kernel driver or ROCm stack.
Start from:
The official AMD hackathon image
An AMD Developer Cloud image
Another image where rocminfo already works
Verify:
rocminfo | grep -m1 -E 'Name:.*gfx'
rocm-smi --showproductname
Clone Project
cd /workspace
git clone <STAGEPOSTER_REPOSITORY_URL> poster-engine
cd poster-engine/backend
Replace the placeholder repository URL before publication.
Bootstrap
chmod +x scripts/*.sh
./scripts/bootstrap.sh
This installs or prepares:
System packages
Go 1.25.0
uv
Python 3.10.20 ComfyUI environment
Python 3.12.3 vLLM environment
ComfyUI at the verified commit
Backend dependencies
Configure
cp .env.example .env
nano .env
At minimum, verify:
WORKFLOW_PATH
PROMPT_NODE_ID
SEED_NODE_ID
VLM_API_KEY
VLM_MODEL_PATH
Download Models
export HF_TOKEN=<optional-token>
./scripts/download-models.sh

Expected files:
models/Qwen3.5-9B/

ComfyUI/models/diffusion_models/z_image_turbo_bf16.safetensors
ComfyUI/models/text_encoders/qwen_3_4b.safetensors
ComfyUI/models/vae/ae.safetensors
ComfyUI/models/loras/z_image_turbo_distill_patch_lora_bf16.safetensors

## Install Workflow

Ensure this file exists:

```text
/workspace/poster-engine/workflows/z_image_poster_v1.json
Verify node mapping:
PROMPT_NODE_ID=57:27
SEED_NODE_ID=57:3
If the workflow was re-exported, node IDs may change.
Build Backend
cd /workspace/poster-engine/backend

go mod download
go test ./...
go build -o poster-backend ./cmd/server

## Start Services

```bash
./scripts/start-all.sh
Verify
./scripts/status.sh
./scripts/smoke-test.sh
Expected:
VLM READY
VLM SLEEPING
COMFYUI READY
BACKEND READY
Start Public Development Access
./scripts/install-cloudflared.sh
./scripts/start-dev-tunnel.sh
The script prints a temporary public URL.
Full Golden E2E
The repository already contains:
scripts/e2e-test.sh
Review the script before running because it performs real GPU inference.
Run:
./scripts/e2e-test.sh
The expected lifecycle is:
Create session
Send requirements
Generate plans
Confirm plan
Generate three candidates
Select candidate
Compose poster
Review
Recompose
Review again
Restore best round
Finalize
Retry finalize
Capture Exact Runtime Locks
Because AMD image wheels may differ from generic ROCm repositories:
./scripts/export-runtime-locks.sh
Commit the generated lock reports after reviewing them for secrets.
Reproduction Evidence
Recommended evidence to store under docs/evidence/:
system.txt
gpu.txt
versions.txt
go-test.txt
smoke-test.txt
golden-e2e.txt
final-poster.png
review-round-1-final_poster.png
review-round-2-final_poster.png
Known Environment Difference
The verified ComfyUI environment contains:
torch 2.13.0+rocm7.2
torchvision 0.28.0+rocm7.2
torchaudio 2.11.0+rocm7.2
These exact builds may be supplied by the AMD base image.
The generic installer uses a compatible AMD ROCm wheel set when the exact image wheel is unavailable.
For bit-for-bit reproduction, retain the exact AMD base image identifier and exported lock report.
