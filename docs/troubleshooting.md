# Troubleshooting

## `ss: command not found`

Install iproute2:

```bash
apt-get update
apt-get install -y iproute2
Alternative checks:
lsof -iTCP -sTCP:LISTEN -P -n
or:
ps aux | grep -E 'vllm|main.py|poster-backend' | grep -v grep
Backend Session Appears Stale
Symptom:
Poster status is succeeded
AI Session still says generating_candidates
Cause:
A low-level Poster route was used instead of the AI Session route.
Fix:
curl \
  http://127.0.0.1:8080/api/ai/sessions/<SESSION_ID>
The Session GET reconciles state with the Poster.
Frontend prevention:
Use:
/api/ai/sessions/{sessionId}/candidates/{candidateId}/select
not:
/api/posters/{posterId}/select
Finalize Returns Conflict
Check Session:
curl \
  http://127.0.0.1:8080/api/ai/sessions/<SESSION_ID>
Finalize is allowed only after composition has succeeded.
VLM Is Sleeping
Check:
curl \
  http://127.0.0.1:8001/is_sleeping \
  -H "Authorization: Bearer $VLM_API_KEY"
Wake manually:
curl -X POST \
  http://127.0.0.1:8001/wake_up \
  -H "Authorization: Bearer $VLM_API_KEY"
Sleep manually:
curl -X POST \
  'http://127.0.0.1:8001/sleep?level=1' \
  -H "Authorization: Bearer $VLM_API_KEY"
Sleep Endpoints Return 404
Ensure vLLM is started with:
VLLM_SERVER_DEV_MODE=1
--enable-sleep-mode
These endpoints must remain private.
GPU Out of Memory
Check VRAM:
rocm-smi --showmeminfo vram
Check VLM state:
curl \
  http://127.0.0.1:8001/is_sleeping \
  -H "Authorization: Bearer $VLM_API_KEY"
Possible mitigations:
Sleep vLLM before ComfyUI generation
Reduce VLLM_GPU_MEMORY_UTILIZATION
Reduce VLLM_MAX_MODEL_LEN
Reduce multimodal input resolution
Reduce VLLM_ROCM_SLEEP_MEM_CHUNK_SIZE
Stop duplicate model processes
Duplicate Services
pgrep -af 'vllm serve'
pgrep -af 'main.py.*8188'
pgrep -af poster-backend
Stop duplicates before restarting.
ComfyUI Model Not Found
Verify:
find ComfyUI/models -type f -name '*.safetensors'
Required paths:
ComfyUI/models/diffusion_models/
ComfyUI/models/text_encoders/
ComfyUI/models/vae/
ComfyUI/models/loras/
Restart ComfyUI after adding model files.
Workflow Node Not Found
Check:
PROMPT_NODE_ID
SEED_NODE_ID
NEGATIVE_PROMPT_NODE_ID
Re-exported workflows can change node IDs.
Inspect:
python3 -m json.tool \
  /workspace/poster-engine/workflows/z_image_poster_v1.json \
  | less
Poster Candidate File Missing
Check job output:
sqlite3 data/poster.db "
SELECT id, job_id, status
FROM poster_candidates
ORDER BY created_at DESC
LIMIT 10;
"
Check output storage:
find storage/jobs -type f | tail
Review Snapshot Missing
Expected:
review-round-1-final_poster.png
review-round-1-thumbnail.png
Inspect:
find storage/posters -type f \
  -name 'review-round-*' \
  -printf '%s\t%p\n'
Cloudflare Tunnel Has No URL
grep -oE \
  'https://[a-zA-Z0-9-]+\.trycloudflare\.com' \
  logs/cloudflared.log
Cloudflare ICMP Warning
Warnings about ICMP proxy permissions do not normally block HTTP proxying.
Confirm:
Registered tunnel connection
Environment is healthy
Then test the public URL with curl.
Cloudflare 502
Check backend locally:
curl -i \
  http://127.0.0.1:8080/api/ai/sessions/not-found
Check tunnel target:
pgrep -af cloudflared
tail -100 logs/cloudflared.log
Go Module Download Timeout
Try:
export GOPROXY=https://proxy.golang.org,direct
go mod download
In restricted regions, configure an accessible Go proxy explicitly.
SQLite Locked
Check duplicate backend processes:
pgrep -af poster-backend
Only one backend process should write the local SQLite database.
Logs
tail -f logs/backend.log
tail -f logs/comfyui.log
tail -f logs/vllm.log
tail -f logs/cloudflared.log
