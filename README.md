# Poster Engine

AI Multimodal Poster Generation System.

## define
用户提交活动信息和 Logo，系统自动规划三种设计方向，生成、检查、失败重试。用户只挑方案，不写提示词。挑中后不再重新抽图，直接排版生成最终海报。

## Architecture

User
 |
Poster API
 |
LLM Planner
 |
ComfyUI Runtime
 |
Z-Image Turbo
 |
Generated Poster


## Components

- LLM planning
- Image generation
- Workflow orchestration
- Review loop
- Asset management

## Current Status

- AMD W7900 ROCm inference ready
- Z-Image Turbo deployed
- ComfyUI workflow ready


## MVP-0: Poster Generation Pipeline

Goal:
输入:
- artist
- event info
- logo
- style

输出:
- generated poster

Pipeline:
API
 ↓
Prompt Builder
 ↓
ComfyUI
 ↓
Image Generation
 ↓
Result Storage


# backend run command (dev)


```bash
if [ -f /workspace/poster-engine/backend.pid ]; then
  kill "$(cat /workspace/poster-engine/backend.pid)" 2>/dev/null || true
fi

pkill -f '/workspace/poster-engine/backend/poster-backend' 2>/dev/null || true

cd /workspace/poster-engine/backend

nohup env \
  LISTEN_ADDR=:8080 \
  COMFY_URL=http://127.0.0.1:8188 \
  WORKFLOW_PATH=/workspace/poster-engine/workflows/z_image_poster_v1.json \
  PROMPT_NODE_ID='57:27' \
  SEED_NODE_ID='57:3' \
  DB_PATH=/workspace/poster-engine/backend/data/poster.db \
  STORAGE_ROOT=/workspace/poster-engine/backend/storage/jobs \
  WORKFLOW_KEY=poster-text \
  WORKFLOW_VERSION=1.0.0 \
  RECONCILE_INTERVAL=2s \
  POSTER_API_TOKEN='poster-dev-2026' \
  CORS_ORIGIN='*' \
  ./poster-backend \
  > /workspace/poster-engine/logs/backend.log 2>&1 &

echo $! > /workspace/poster-engine/backend.pid

sleep 2
cat /workspace/poster-engine/logs/backend.log

```

```bash
用户原始需求
       ↓
Poster Goal Spec
       ↓
Creative Brief Agent
       ↓
Prompt Planner
       ↓
生成多组 Key Visual
       ↓
多模板 Composer 合成
       ↓
程序化质量检查
       ↓
AI Vision Critic 视觉评审
       ↓
Decision Router
   ┌──────┼──────────┐
   ↓      ↓          ↓
通过   重新排版    重新生成主视觉
   │      │          │
   └──────┴──── Loop ┘
       ↓
保留最佳结果
       ↓
用户最终选择
```
