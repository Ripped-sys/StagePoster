#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="/workspace/poster-engine"
BACKEND="$ROOT/backend"

# 必须落在 NFS 持久化根目录下，不能落在 $ROOT 里面。
#
# 这里原先是 "$ROOT/persist" —— 也就是 /workspace/poster-engine/persist，
# 恰好是实例重置时会被清掉的地方。脚本会照常跑完并打印
# "PERSISTENCE PREPARATION COMPLETE"，然后连同备份一起消失。
PERSIST="${PERSIST_ROOT:-/workspace/persistence/stageposter}/prekill"

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

# DB_PATH 从 .env 读，不要硬编码。
#
# 这里原先写死 "$BACKEND/data/poster.db"。那个文件还在盘上，但它是早期遗留 ——
# 线上库由 .env 的 DB_PATH 指向持久化目录。写死的结果是备份了一份陈旧数据
# （实测 13 行 vs 线上 56 行）并且报告成功，这比备份失败更危险。
DB_PATH=""
if [[ -f "$ROOT/.env" ]]; then
  DB_PATH="$(grep -E '^DB_PATH=' "$ROOT/.env" | tail -1 | cut -d= -f2- || true)"
fi
DB_PATH="${DB_PATH:-$BACKEND/data/poster.db}"

echo "DB_PATH=$DB_PATH"

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

# 复制权重前先看清两件事，否则这一步是纯亏。
#
# 1. 如果 $PERSIST 和权重源在同一个文件系统上，复制 20G 不构成任何保护 ——
#    卷被清掉时两份一起没。真正的保护是"能重新下载"，靠的是 CHECKSUMS 段
#    产出的清单，不是这份副本。
# 2. 剩余空间不够时 rsync 会写到一半失败，留下一堆截断文件，
#    而截断的权重比没有权重更难排查。
#
# 所以：同卷 + 空间不足 → 跳过复制，只留清单。设 FORCE_MODEL_COPY=1 可强制。
SRC_DEV="$(stat -c%d "$ROOT/ComfyUI/models" 2>/dev/null || echo 0)"
DST_DEV="$(stat -c%d "$PERSIST" 2>/dev/null || echo 1)"

MODELS_BYTES="$(du -sb "$ROOT/ComfyUI/models" 2>/dev/null | cut -f1 || echo 0)"
FREE_BYTES="$(df -B1 --output=avail "$PERSIST" 2>/dev/null | tail -1 | tr -d ' ' || echo 0)"

SKIP_MODEL_COPY=0

if [[ "${FORCE_MODEL_COPY:-0}" != "1" ]]; then
  if [[ "$SRC_DEV" == "$DST_DEV" ]]; then
    echo "跳过权重复制：源与目标在同一文件系统 (dev=$SRC_DEV)，"
    echo "复制无法提供额外保护。清单仍会生成。"
    echo "如确需复制：FORCE_MODEL_COPY=1 $0"
    SKIP_MODEL_COPY=1
  elif (( FREE_BYTES < MODELS_BYTES * 110 / 100 )); then
    echo "跳过权重复制：空间不足。"
    echo "  需要约 $(( MODELS_BYTES / 1024 / 1024 / 1024 ))G，可用 $(( FREE_BYTES / 1024 / 1024 / 1024 ))G"
    SKIP_MODEL_COPY=1
  fi
fi

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

  # 记录尺寸供恢复时核对。hf-mirror 的 Xet 大文件会静默截断
  # （绕过镜像直连 cas-server.xethub.hf.co 然后 401），只能按字节数核对。
  echo "$(stat -c%s "$source") $filename" \
    >> "$SNAPSHOT/model-sizes.txt"

  if [[ "$SKIP_MODEL_COPY" == "1" ]]; then
    echo "  (只记清单，不复制)"
    return
  fi

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
