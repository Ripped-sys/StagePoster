#!/usr/bin/env bash
set -Eeuo pipefail

BACKEND_ROOT="$(
  cd "$(dirname "${BASH_SOURCE[0]}")/.." &&
  pwd
)"

ENV_FILE="${ENV_FILE:-$BACKEND_ROOT/.env}"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Missing $ENV_FILE" >&2
  echo "Copy .env.example to .env and edit it first." >&2
  exit 1
fi

set -a
source "$ENV_FILE"
set +a

ROOT="${STAGEPOSTER_ROOT:-/workspace/poster-engine}"
COMFY_ROOT="${COMFY_ROOT:-$ROOT/ComfyUI}"
COMFY_VENV="${COMFY_VENV:-$ROOT/venv}"
VLLM_VENV="${VLLM_VENV:-$ROOT/.venv-vllm}"

mkdir -p \
  "$BACKEND_ROOT/logs" \
  "$BACKEND_ROOT/run" \
  "$BACKEND_ROOT/data" \
  "$BACKEND_ROOT/storage/jobs" \
  "$BACKEND_ROOT/storage/assets" \
  "$BACKEND_ROOT/storage/posters"

pid_alive() {
  local file="$1"

  [[ -f "$file" ]] || return 1

  local pid
  pid="$(cat "$file")"

  kill -0 "$pid" 2>/dev/null
}

wait_http() {
  local name="$1"
  local url="$2"
  local attempts="${3:-120}"
  local auth="${4:-}"

  for _ in $(seq 1 "$attempts"); do
    if [[ -n "$auth" ]]; then
      if curl -fsS \
        -H "Authorization: Bearer $auth" \
        "$url" >/dev/null 2>&1; then
        echo "$name READY"
        return 0
      fi
    else
      if curl -fsS "$url" >/dev/null 2>&1; then
        echo "$name READY"
        return 0
      fi
    fi

    sleep 2
  done

  echo "$name failed to become ready: $url" >&2
  return 1
}

start_vllm() {
  local pid_file="$BACKEND_ROOT/run/vllm.pid"

  if pid_alive "$pid_file"; then
    echo "vLLM already running: PID $(cat "$pid_file")"
    return
  fi

  if [[ ! -x "$VLLM_VENV/bin/vllm" ]]; then
    echo "Missing vLLM executable: $VLLM_VENV/bin/vllm" >&2
    exit 1
  fi

  if [[ ! -d "$VLM_MODEL_PATH" ]]; then
    echo "Missing VLM model: $VLM_MODEL_PATH" >&2
    exit 1
  fi

  (
    cd "$ROOT"

    nohup env \
      VLLM_SERVER_DEV_MODE="${VLLM_SERVER_DEV_MODE:-1}" \
      VLLM_ROCM_SLEEP_MEM_CHUNK_SIZE="${VLLM_ROCM_SLEEP_MEM_CHUNK_SIZE:-256}" \
      "$VLLM_VENV/bin/vllm" serve "$VLM_MODEL_PATH" \
        --host "${VLM_HOST:-127.0.0.1}" \
        --port "${VLM_PORT:-8001}" \
        --served-model-name "${VLM_MODEL:-stageposter-vlm}" \
        --api-key "$VLM_API_KEY" \
        --dtype "${VLLM_DTYPE:-float16}" \
        --max-model-len "${VLLM_MAX_MODEL_LEN:-4096}" \
        --max-num-seqs "${VLLM_MAX_NUM_SEQS:-1}" \
        --max-num-batched-tokens "${VLLM_MAX_BATCHED_TOKENS:-4096}" \
        --gpu-memory-utilization "${VLLM_GPU_MEMORY_UTILIZATION:-0.65}" \
        --limit-mm-per-prompt '{"image":{"count":1,"width":768,"height":1152},"video":0}' \
        --enforce-eager \
        --enable-sleep-mode \
        --default-chat-template-kwargs '{"enable_thinking":false}' \
        --generation-config vllm \
        > "$BACKEND_ROOT/logs/vllm.log" 2>&1 &

    echo $! > "$pid_file"
  )

  wait_http \
    "VLM" \
    "${VLM_URL:-http://127.0.0.1:8001}/v1/models" \
    180 \
    "$VLM_API_KEY"

  curl -fsS \
    -X POST \
    -H "Authorization: Bearer $VLM_API_KEY" \
    "${VLM_URL:-http://127.0.0.1:8001}/sleep?level=1" \
    >/dev/null || true

  echo "VLM sleep requested."
}

start_comfyui() {
  local pid_file="$BACKEND_ROOT/run/comfyui.pid"

  if pid_alive "$pid_file"; then
    echo "ComfyUI already running: PID $(cat "$pid_file")"
    return
  fi

  if [[ ! -x "$COMFY_VENV/bin/python" ]]; then
    echo "Missing ComfyUI Python: $COMFY_VENV/bin/python" >&2
    exit 1
  fi

  if [[ ! -f "$COMFY_ROOT/main.py" ]]; then
    echo "Missing ComfyUI: $COMFY_ROOT/main.py" >&2
    exit 1
  fi

  (
    cd "$COMFY_ROOT"

    nohup "$COMFY_VENV/bin/python" main.py \
      --listen "${COMFY_LISTEN:-127.0.0.1}" \
      --port "${COMFY_PORT:-8188}" \
      > "$BACKEND_ROOT/logs/comfyui.log" 2>&1 &

    echo $! > "$pid_file"
  )

  wait_http \
    "COMFYUI" \
    "${COMFY_URL:-http://127.0.0.1:8188}/system_stats" \
    180
}

start_backend() {
  local pid_file="$BACKEND_ROOT/run/backend.pid"
  local binary="$BACKEND_ROOT/poster-backend"

  if pid_alive "$pid_file"; then
    echo "Backend already running: PID $(cat "$pid_file")"
    return
  fi

  cd "$BACKEND_ROOT"

  go test ./...

  if ! go build -o "$binary" ./cmd/server; then
    echo "Build ./cmd/server failed." >&2
    echo "Check the actual backend main package path." >&2
    exit 1
  fi

  nohup "$binary" \
    > "$BACKEND_ROOT/logs/backend.log" 2>&1 &

  echo $! > "$pid_file"

  for _ in $(seq 1 60); do
    code="$(
      curl -sS \
        -o /dev/null \
        -w '%{http_code}' \
        "http://127.0.0.1:8080/api/ai/sessions/not-found" \
        || true
    )"

    if [[ "$code" == "404" ]]; then
      echo "BACKEND READY"
      return
    fi

    sleep 1
  done

  echo "Backend failed to become ready." >&2
  return 1
}

start_vllm
start_comfyui
start_backend

echo
echo "All StagePoster services are running."
echo "Run: $BACKEND_ROOT/scripts/status.sh"
