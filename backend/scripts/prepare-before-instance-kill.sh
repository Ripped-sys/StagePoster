#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="/workspace/poster-engine"
BACKEND="$ROOT/backend"
PERSIST="$ROOT/persist"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
SNAPSHOT="$PERSIST/snapshots/$STAMP"

mkdir -p \
  "$SNAPSHOT" \
  "$SNAPSHOT/e2e" \
  "$PERSIST/private" \
  "$PERSIST/models/comfyui/diffusion_models" \
  "$PERSIST/models/comfyui/text_encoders" \
  "$PERSIST/models/comfyui/vae" \
  "$PERSIST/models/comfyui/loras" \
  "$ROOT/toolchains" \
  "$ROOT/tools/bin"

chmod 700 "$PERSIST/private"

echo "=================================================="
echo " StagePoster pre-kill persistence"
echo " Snapshot: $SNAPSHOT"
echo "=================================================="

echo
echo "===== WORKSPACE DISK ====="
df -h "$ROOT"
du -sh "$ROOT" 2>/dev/null || true

echo
echo "===== SQLITE CHECKPOINT AND BACKUP ====="

DB_PATH="$BACKEND/data/poster.db"

if [[ -f "$DB_PATH" ]]; then
  sqlite3 "$DB_PATH" "PRAGMA wal_checkpoint(FULL);" || true

  sqlite3 "$DB_PATH" \
    ".backup '$SNAPSHOT/poster.db'"

  cp -a "$DB_PATH" "$SNAPSHOT/poster.db.raw"

  [[ -f "${DB_PATH}-wal" ]] &&
    cp -a "${DB_PATH}-wal" "$SNAPSHOT/" || true

  [[ -f "${DB_PATH}-shm" ]] &&
    cp -a "${DB_PATH}-shm" "$SNAPSHOT/" || true

  echo "Database backup saved."
else
  echo "WARNING: database not found: $DB_PATH"
fi

echo
echo "===== SAVE ENVIRONMENT ====="

if [[ -f "$BACKEND/.env" ]]; then
  install -m 600 \
    "$BACKEND/.env" \
    "$PERSIST/private/backend.env"
fi

cp -a \
  "$BACKEND/.env.example" \
  "$SNAPSHOT/" \
  2>/dev/null || true

echo
echo "===== SAVE WORKFLOW ====="

mkdir -p "$SNAPSHOT/workflows"

cp -a \
  "$ROOT/workflows/." \
  "$SNAPSHOT/workflows/" \
  2>/dev/null || true

echo
echo "===== SAVE GOLDEN E2E ARTIFACTS FROM /tmp ====="

shopt -s nullglob

for directory in /tmp/stageposter-*; do
  echo "Copying $directory"
  cp -a "$directory" "$SNAPSHOT/e2e/"
done

shopt -u nullglob

echo
echo "===== EXPORT RUNTIME LOCKS ====="

if [[ -x "$BACKEND/scripts/export-runtime-locks.sh" ]]; then
  "$BACKEND/scripts/export-runtime-locks.sh" || true
fi

cp -a \
  "$BACKEND/locks" \
  "$SNAPSHOT/" \
  2>/dev/null || true

echo
echo "===== RECORD RUNNING COMMANDS ====="

{
  echo "DATE=$(date -u --iso-8601=seconds)"
  echo

  echo "VLLM"
  pgrep -af 'vllm serve' || true
  echo

  echo "COMFYUI"
  pgrep -af 'main.py.*8188' || true
  echo

  echo "BACKEND"
  pgrep -af poster-backend || true
  echo

  echo "CLOUDFLARE"
  pgrep -af cloudflared || true
} > "$SNAPSHOT/processes.txt"

for PID in $(
  pgrep -f \
    'vllm serve|main.py.*8188|poster-backend|cloudflared' \
    || true
); do
  {
    echo "===== PID $PID ====="
    echo "WORKDIR:"
    readlink -f "/proc/$PID/cwd" || true
    echo
    echo "COMMAND:"
    tr '\0' ' ' < "/proc/$PID/cmdline" || true
    echo
    echo
  } >> "$SNAPSHOT/process-details.txt"
done

echo
echo "===== RECORD SYSTEM VERSIONS ====="

{
  cat /etc/os-release
  echo

  uname -a
  echo

  go version || true
  echo

  rocm-smi --showproductname || true
  echo

  rocminfo |
    grep -m1 -E 'Name:.*gfx' || true
  echo

  hipcc --version || true
  echo

  "$ROOT/venv/bin/python" --version || true
  "$ROOT/.venv-vllm/bin/python" --version || true
} > "$SNAPSHOT/system-versions.txt"

echo
echo "===== SAVE GIT STATE ====="

{
  git -C "$BACKEND" remote -v || true
  git -C "$BACKEND" branch --show-current || true
  git -C "$BACKEND" rev-parse HEAD || true
  git -C "$BACKEND" status --short || true
} > "$SNAPSHOT/backend-git-state.txt"

{
  git -C "$ROOT/ComfyUI" remote -v || true
  git -C "$ROOT/ComfyUI" rev-parse HEAD || true
  git -C "$ROOT/ComfyUI" status --short || true
} > "$SNAPSHOT/comfyui-git-state.txt"

git config --global --list \
  > "$SNAPSHOT/git-global-config.txt" \
  2>/dev/null || true

echo
echo "===== AUDIT WORKSPACE SYMLINKS ====="

find "$ROOT" \
  -type l \
  -printf '%p -> %l\n' \
  > "$SNAPSHOT/symlinks.txt" \
  2>/dev/null || true

while IFS= read -r link; do
  resolved="$(readlink -f "$link" 2>/dev/null || true)"

  case "$resolved" in
    "$ROOT"/*)
      ;;
    "")
      echo "BROKEN: $link"
      ;;
    *)
      echo "EXTERNAL: $link -> $resolved"
      ;;
  esac
done < <(
  find "$ROOT" -type l 2>/dev/null
) | tee "$SNAPSHOT/external-symlinks.txt"

echo
echo "===== BACK UP REQUIRED COMFYUI MODELS ====="

SEARCH_ROOTS=(
  "$ROOT/ComfyUI/models"
)

if [[ -e /models ]]; then
  SEARCH_ROOTS+=("/models")
fi

MODEL_MISSING=0

copy_model() {
  local filename="$1"
  local category="$2"
  local source=""

  source="$(
    find -L "${SEARCH_ROOTS[@]}" \
      -type f \
      -name "$filename" \
      -print \
      2>/dev/null \
      | head -1
  )"

  if [[ -z "$source" ]]; then
    echo "MISSING MODEL: $filename"
    MODEL_MISSING=1
    return
  fi

  echo "Saving model:"
  echo "  $source"

  rsync -ah \
    --info=progress2 \
    "$source" \
    "$PERSIST/models/comfyui/$category/$filename"
}

copy_model \
  "z_image_turbo_bf16.safetensors" \
  "diffusion_models"

copy_model \
  "qwen_3_4b.safetensors" \
  "text_encoders"

copy_model \
  "ae.safetensors" \
  "vae"

copy_model \
  "z_image_turbo_distill_patch_lora_bf16.safetensors" \
  "loras"

echo
echo "===== VERIFY QWEN MODEL ====="

QWEN_DIR="$ROOT/models/Qwen3.5-9B"

if [[ -d "$QWEN_DIR" ]]; then
  find "$QWEN_DIR" \
    -maxdepth 1 \
    -type f \
    -printf '%s\t%f\n' \
    | sort -n \
    > "$SNAPSHOT/qwen-model-files.txt"
else
  echo "WARNING: Qwen model directory missing."
fi

echo
echo "===== SAVE GO TOOLCHAIN ====="

if [[ -d /usr/local/go ]]; then
  mkdir -p "$ROOT/toolchains/go1.25.0"

  rsync -a \
    --delete \
    /usr/local/go/ \
    "$ROOT/toolchains/go1.25.0/"

  echo "Go toolchain saved."
else
  echo "WARNING: /usr/local/go not found."
fi

echo
echo "===== SAVE CLOUDFLARED ====="

CLOUDFLARED_BIN="$(
  command -v cloudflared 2>/dev/null || true
)"

if [[ -n "$CLOUDFLARED_BIN" ]]; then
  install -m 755 \
    "$CLOUDFLARED_BIN" \
    "$ROOT/tools/bin/cloudflared"

  "$ROOT/tools/bin/cloudflared" --version \
    > "$SNAPSHOT/cloudflared-version.txt" \
    2>&1 || true
fi

echo
echo "===== SAVE UV BINARY ====="

UV_BIN="$(command -v uv 2>/dev/null || true)"

if [[ -n "$UV_BIN" ]]; then
  install -m 755 \
    "$UV_BIN" \
    "$ROOT/tools/bin/uv"
fi

save_python_runtime() {
  local name="$1"
  local venv_python="$2"

  if [[ ! -e "$venv_python" ]]; then
    echo "$name Python missing: $venv_python"
    return
  fi

  local resolved
  resolved="$(readlink -f "$venv_python")"

  echo "$name Python:"
  echo "  executable: $venv_python"
  echo "  resolved:   $resolved"

  printf '%s\n' "$resolved" \
    > "$SNAPSHOT/${name}-python-resolved.txt"

  case "$resolved" in
    "$ROOT"/*)
      echo "  already persistent"
      return
      ;;

    /usr/bin/*|/usr/local/bin/*)
      echo "  system interpreter, will be reinstalled after restart"
      return
      ;;
  esac

  local runtime_root
  runtime_root="$(dirname "$(dirname "$resolved")")"

  local destination="$ROOT/toolchains/${name}-python-runtime"

  mkdir -p "$destination"

  rsync -a \
    --delete \
    "$runtime_root/" \
    "$destination/"

  {
    echo "ORIGINAL_ROOT=$runtime_root"
    echo "PERSISTENT_ROOT=$destination"
  } > "$PERSIST/${name}-python-runtime.env"

  echo "  Python runtime copied to $destination"
}

save_python_runtime \
  "comfyui" \
  "$ROOT/venv/bin/python"

save_python_runtime \
  "vllm" \
  "$ROOT/.venv-vllm/bin/python"

echo
echo "===== CHECKSUMS ====="

{
  find "$PERSIST/models/comfyui" \
    -type f \
    -print0

  find "$QWEN_DIR" \
    -maxdepth 1 \
    -type f \
    \( \
      -name '*.safetensors' -o \
      -name '*.json' \
    \) \
    -print0 \
    2>/dev/null || true
} | xargs -0 -r sha256sum \
  > "$SNAPSHOT/model-checksums.sha256"

echo
echo "===== STORAGE SUMMARY ====="

du -sh \
  "$BACKEND/data" \
  "$BACKEND/storage" \
  "$ROOT/models" \
  "$PERSIST/models" \
  "$ROOT/toolchains" \
  2>/dev/null || true

sync

echo
echo "=================================================="
echo " PERSISTENCE PREPARATION COMPLETE"
echo "=================================================="
echo
echo "Snapshot:"
echo "$SNAPSHOT"
echo
echo "Persistent model backup:"
echo "$PERSIST/models/comfyui"
echo

if [[ "$MODEL_MISSING" -ne 0 ]]; then
  echo "WARNING: one or more required ComfyUI models were not found."
  echo "Inspect:"
  echo "$SNAPSHOT/external-symlinks.txt"
  exit 2
fi

echo "All required model files were copied successfully."
