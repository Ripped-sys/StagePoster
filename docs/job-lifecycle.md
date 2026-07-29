# StagePoster Job Lifecycle v1.0

## Goal

Define async task status, progress, persistence, cancellation, retry, and
recovery for the MVP: single Go backend process + SQLite + bounded in-memory
queue + private ComfyUI.

## 1. Why Async Is Required

A single poster generation may involve:

- Brief validation
- Style planning
- Workflow selection
- Asset preprocessing
- Upload inputs to ComfyUI
- Wait for GPU queue
- Model inference
- Retrieve outputs
- Logo and text composition
- Generate preview
- Persist results

The task creation endpoint must return `job_id` immediately. Long-lived HTTP
connections are not acceptable.

## 2. Top-Level States

| State | Meaning | Terminal |
|---|---|---|
| `queued` | Waiting to execute | No |
| `running` | Currently executing | No |
| `cancelling` | Cancellation in progress | No |
| `completed` | Finished successfully | Yes |
| `failed` | Finished with error | Yes |
| `cancelled` | Cancelled by user | Yes |

### Sub-States (ComfyUI polling)

| State | Meaning |
|---|---|
| `submitted` | Prompt sent to ComfyUI |
| `processing` | ComfyUI is generating |
| `uploading_output` | Copying output files to storage |

## 3. Progress Model

```json
{
  "total": 3,
  "completed": 1,
  "failed": 0,
  "percent": 33
}
```

Progress is updated in the database by the reconciler goroutine. The frontend
polls `GET /api/ai/sessions/{sessionId}` for current progress.

## 4. Cancellation

```
POST /api/jobs/{jobId}/cancel
```

Cancellation is best-effort:
1. Set status to `cancelling`
2. If ComfyUI job is still queued, remove from queue
3. If already running, let it finish but discard output
4. Update status to `cancelled`

## 5. Retry

```
POST /api/jobs/{jobId}/retry
```

Retry rules:
- Only `failed` jobs can be retried
- Maximum 2 retry attempts
- Each retry generates a new ComfyUI prompt (new seed)
- Retry count stored in `retry_count` column

## 6. Recovery on Backend Restart

On startup, the backend:

1. Queries all `queued` and `running` jobs from SQLite
2. For each job, checks ComfyUI history via `/history/{prompt_id}`
3. If ComfyUI shows `completed`, transitions to `completed` and copies output
4. If ComfyUI shows `failed`, transitions to `failed`
5. If ComfyUI has no record, transitions to `failed` (stale)

## 7. Memory Bounds

- In-memory queue: max 100 pending jobs
- Reconciler tick: every 2 seconds
- Reconciler timeout per tick: 90 seconds
- Max concurrent ComfyUI prompts: 1 (single GPU)

## 8. API

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/jobs?limit=20` | List jobs |
| `GET` | `/api/jobs/{jobId}` | Get job status |
| `POST` | `/api/jobs/{jobId}/cancel` | Cancel job |
| `POST` | `/api/jobs/{jobId}/retry` | Retry failed job |
| `GET` | `/api/jobs/{jobId}/result` | Download result |
| `GET` | `/api/jobs/{jobId}/thumbnail` | Download thumbnail |
