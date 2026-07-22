# Frontend Integration Guide

## Development Base URL

```env
NEXT_PUBLIC_API_BASE_URL=https://cst-holmes-climate-charge.trycloudflare.com
This is a temporary Cloudflare Quick Tunnel URL.
Do not commit it as a production constant.
API Client
Example TypeScript client:
const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://127.0.0.1:8080";

export async function apiRequest<T>(
  path: string,
  init: RequestInit = {},
): Promise<T> {
  const response = await fetch(${API_BASE_URL}${path}, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...init.headers,
    },
  });
  const data = await response.json().catch(() => null);
  if (!response.ok) {
    const message =
      data && typeof data.error === "string"
        ? data.error
        : Request failed with HTTP ${response.status};
throw new Error(message);
  }
  return data as T;
}

## State-Driven UI

Do not maintain a separate frontend workflow state machine when the backend already returns one.

Use:

```text
status
availableActions
missingFields
plans
poster
reviewSummary
Recommended Screen States
Chat
Show when:
No confirmed plan exists
and the session is not terminal
Render:
Message history
Current brief
Missing fields
Message input
Plan Selection
Show when plans exist and confirmation is available.
Each plan should display:
Name
Concept
Palette
Composition
Composer template
Candidate Generation
Show when:
status == generating_candidates
Poll the Session every two or three seconds.
Display:
poster.progress.completed
poster.progress.total
Candidate Selection
Show when:
availableActions includes select_candidate
Use each candidate's relative imageUrl.
Final Review
Show when:
availableActions includes finalize
Finalization can take several minutes.
Use a long request timeout or an asynchronous UI state.
Download
Show when:
availableActions includes download_final
Use:
poster.resultUrl
poster.thumbnailUrl
Relative Image URLs
Backend image URLs are relative.
Convert:
/api/posters/poster_x/result
to:
const fullImageUrl = `${API_BASE_URL}${relativeImageUrl}`;
Authenticated Images
When POSTER_API_TOKEN is enabled, normal <img src> tags cannot attach a custom header.
Use one of these approaches:
Keep read-only image routes publicly accessible.
Fetch the image as a Blob with an Authorization header.
Introduce short-lived signed image URLs.
Blob example:
export async function fetchProtectedImage(path: string, token: string) {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    throw new Error(`Image request failed: ${response.status}`);
  }

  return URL.createObjectURL(await response.blob());
}
Polling
Cloudflare Quick Tunnel does not support SSE.
Use polling during development.
Recommended behavior:
Interval: 2 to 3 seconds
Stop on terminal state
Stop when expected availableAction appears
Back off after network failures
Cancel polling when component unmounts
Example:
const TERMINAL = new Set([
  "succeeded",
  "completed_with_warnings",
  "failed",
  "cancelled",
  "needs_user_input",
]);

async function waitForCandidateSelection(sessionId: string) {
  for (;;) {
    const session = await apiRequest<any>(
      /api/ai/sessions/${sessionId},
    );
if (
  session.availableActions?.includes("select_candidate") ||
  TERMINAL.has(session.status)
) {
  return session;
}

await new Promise((resolve) => setTimeout(resolve, 2500));
  }
}

## Timeouts

Recommended client timeouts:

| Operation | Timeout |
|---|---:|
| Create session | 30 seconds |
| Send chat message | 5 minutes |
| Confirm plan | 2 minutes |
| Session poll | 20 seconds |
| Candidate selection | 2 minutes |
| Finalize | 12 minutes |
| Download image | 2 minutes |

## Error Handling

Map status codes as follows:

| HTTP | Meaning |
|---:|---|
| 400 | Invalid JSON or request |
| 404 | Session, poster, candidate, or plan not found |
| 409 | Invalid workflow state |
| 503 | AI service not configured or unavailable |
| 502 | Upstream ComfyUI or VLM failure |
| 500 | Internal storage or repository failure |

## Important Routing Rule

Normal UI flow must select a candidate through:

```text
/api/ai/sessions/{sessionId}/candidates/{candidateId}/select
Do not use:
/api/posters/{posterId}/select
from the normal UI.
Finalize Retry
Finalize is safe to retry.
A network timeout does not necessarily mean that finalization failed.
After a timeout:
GET the Session.
Inspect reviewSummary.
Retry Finalize only if availableActions still includes finalize.
Public Tunnel Caveat
The current Quick Tunnel URL changes when the tunnel restarts.
The frontend should load it from environment configuration instead of source code.
