#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="/workspace/poster-engine"
BACKEND="$ROOT/backend"
PERSIST="$ROOT/persist"

if [[ "$(id -u)" -ne 0 ]]; then
  echo "Run this script as root." >&2
  exit 1
fi

echo "=================================================="
echo " StagePoster instance recovery"
echo "=================================================="

echo
echo "===== INSTALL BASIC SYSTEM TOOLS ====="

apt-get update

apt-get install -y \
  build-essential \
  ca-certificates \
  curl \
  git \
  iproute2 \
  jq \
  lsof \
  procps \
  rsync \
  sqlite3 \
  unzip

echo
echo "===== RESTORE GO ====="

if [[ -d "$ROOT/toolchains/go1.25.0" ]]; then
  rm -rf /usr/local/go

  ln -s \
    "$ROOT/toolchains/go1.25.0" \
    /usr/local/go

  ln -sf \
    /usr/local/go/bin/go \
    /usr/local/bin/go

  ln -sf \
    /usr/local/go/bin/gofmt \
    /usr/local/bin/gofmt
else
  echo "Persistent Go toolchain not found."
  "$BACKEND/scripts/install-go.sh"
fi

go version

echo
echo "===== RESTORE CLOUDFLARED ====="

if [[ -x "$ROOT/tools/bin/cloudflared" ]]; then
  install -m 755 \
    "$ROOT/tools/bin/cloudflared" \
    /usr/local/bin/cloudflared
else
  "$BACKEND/scripts/install-cloudflared.sh"
fi

cloudflared --version

echo
echo "===== RESTORE UV ====="

if [[ -x "$ROOT/tools/bin/uv" ]]; then
  install -m 755 \
    "$ROOT/tools/bin/uv" \
    /usr/local/bin/uv
fi

restore_python_runtime() {
  local name="$1"
  local metadata="$PERSIST/${name}-python-runtime.env"

  [[ -f "$metadata" ]] || return 0

  unset ORIGINAL_ROOT
  unset PERSISTENT_ROOT

  source "$metadata"

  if [[ -z "${ORIGINAL_ROOT:-}" ]] ||
    [[ -z "${PERSISTENT_ROOT:-}" ]]; then
    return
  fi

  echo "Restoring $name Python runtime:"
  echo "  $ORIGINAL_ROOT -> $PERSISTENT_ROOT"

  mkdir -p "$(dirname "$ORIGINAL_ROOT")"

  rm -rf "$ORIGINAL_ROOT"

  ln -s \
    "$PERSISTENT_ROOT" \
    "$ORIGINAL_ROOT"
}

echo
echo "===== RESTORE PYTHON RUNTIMES ====="

restore_python_runtime "comfyui"
restore_python_runtime "vllm"

echo
echo "===== VERIFY PYTHON ENVIRONMENTS ====="

if ! "$ROOT/venv/bin/python" --version; then
  echo "ComfyUI Python environment is broken." >&2
  echo "Run scripts/install-comfyui.sh to recreate it." >&2
  exit 1
fi

if ! "$ROOT/.venv-vllm/bin/python" --version; then
  echo "vLLM Python environment is broken." >&2
  echo "Run scripts/install-vllm.sh to recreate it." >&2
  exit 1
fi

echo
echo "===== RESTORE COMFYUI MODELS ====="

COMFY_MODELS="$ROOT/ComfyUI/models"
MODEL_BACKUP="$PERSIST/models/comfyui"

if [[ -L "$COMFY_MODELS" ]]; then
  echo "Removing external ComfyUI models symlink:"
  ls -ld "$COMFY_MODELS"
  rm -f "$COMFY_MODELS"
fi

mkdir -p \
  "$COMFY_MODELS/diffusion_models" \
  "$COMFY_MODELS/text_encoders" \
  "$COMFY_MODELS/vae" \
  "$COMFY_MODELS/loras"

rsync -ah \
  "$MODEL_BACKUP/diffusion_models/" \
  "$COMFY_MODELS/diffusion_models/"

rsync -ah \
  "$MODEL_BACKUP/text_encoders/" \
  "$COMFY_MODELS/text_encoders/"

rsync -ah \
  "$MODEL_BACKUP/vae/" \
  "$COMFY_MODELS/vae/"

rsync -ah \
  "$MODEL_BACKUP/loras/" \
  "$COMFY_MODELS/loras/"

echo
echo "===== RESTORE ENV FILE ====="

if [[ ! -f "$BACKEND/.env" ]] &&
  [[ -f "$PERSIST/private/backend.env" ]]; then
  install -m 600 \
    "$PERSIST/private/backend.env" \
    "$BACKEND/.env"
fi

if [[ ! -f "$BACKEND/.env" ]]; then
  cp \
    "$BACKEND/.env.example" \
    "$BACKEND/.env"

  echo "Created a new .env file."
  echo "Review it before starting services."
fi

# A Quick Tunnel URL never survives an instance kill.
sed -i \
  's|^PUBLIC_API_URL=.*|PUBLIC_API_URL=|' \
  "$BACKEND/.env"

echo
echo "===== REMOVE STALE PID FILES ====="

rm -f \
  "$BACKEND/run/"*.pid \
  "$BACKEND/backend.pid" \
  2>/dev/null || true

mkdir -p \
  "$BACKEND/logs" \
  "$BACKEND/run" \
  "$BACKEND/data" \
  "$BACKEND/storage/jobs" \
  "$BACKEND/storage/assets" \
  "$BACKEND/storage/posters"

echo
echo "===== VERIFY ROCM ====="

rocm-smi --showproductname
rocminfo | grep -m1 -E 'Name:.*gfx'
hipcc --version || true

echo
echo "===== VERIFY MODELS ====="

find "$COMFY_MODELS" \
  -type f \
  -name '*.safetensors' \
  -printf '%s\t%p\n' \
  | sort -n

find "$ROOT/models/Qwen3.5-9B" \
  -maxdepth 1 \
  -type f \
  -printf '%s\t%p\n' \
  | sort -n

echo
echo "===== TEST BACKEND SOURCE ====="

cd "$BACKEND"

go test ./...
go build ./...

echo
echo "=================================================="
echo " RECOVERY COMPLETE"
echo "=================================================="
echo
echo "Start services:"
echo "  cd $BACKEND"
echo "  ./scripts/start-all.sh"
echo
echo "Then:"
echo "  ./scripts/status.sh"
echo "  ./scripts/smoke-test.sh"
echo "  ./scripts/start-dev-tunnel.sh"
