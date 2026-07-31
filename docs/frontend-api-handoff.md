# StagePoster 后端 API — 前端对接文档

**基址（Cloudflare Quick Tunnel，HTTPS）**

```
https://<当前隧道子域>.trycloudflare.com
```

> 具体地址**不写进仓库**：Quick Tunnel 子域每次重启都会变，而且当前部署 `POSTER_API_TOKEN`
> 为空（无鉴权），把地址提交进代码等于公开一个任何人都能打的后端。
>
> 取当前地址：服务器上 `cat backend/run/public-api-url.txt`（该文件已被 `.gitignore` 排除），
> 或 `./scripts/status.sh` 输出里的 `Public URL`。由后端负责人直接转发给前端。

前端从 `VITE_API_BASE_URL` 读这个值，**不要硬编码进源码**。
只调这个地址；ComfyUI（:8188）和 vLLM（:8001）是内部服务，不对浏览器开放。

**通用约定**

| 项 | 说明 |
|---|---|
| 请求体 | JSON，`Content-Type: application/json; charset=utf-8` |
| **未知字段会被拒绝** | 所有 JSON 入口都开了 `DisallowUnknownFields`，多传一个字段就是 `400`。字段名照下面写 |
| JSON 响应 | 一律 `application/json; charset=utf-8` |
| 图片响应 | 裸 `image/png`（不带 charset），`Cache-Control: private, max-age=3600` |
| 错误体 | `{"error": "message"}` |
| **`500` 不带细节** | 所有 `500` 一律 `{"error":"internal server error"}`，真实错误只进服务端日志。以前 `500` 会回显 `err.Error()`，文件打开失败时把服务器绝对路径也送出去了 |
| 鉴权 | 当前 `POSTER_API_TOKEN` 为空 = **无需鉴权**。`GET /health` 的 `tokenRequired` 字段会告诉你要不要带。开启后三种方式任选：`Authorization: Bearer <token>` / `X-Poster-Token: <token>` / `?token=<token>` |
| CORS | `Access-Control-Allow-Origin: *`，允许头 `Content-Type, Authorization, X-Poster-Token`，方法 `GET, POST, OPTIONS`，预检返回 `204` |
| 时间 | ISO 8601 UTC，如 `2026-07-30T17:55:12.34Z` |
| ID 前缀 | `poster_` / `candidate_` / `session_` / `asset_` / `job_` / `message_` / `review_` |

**耗时预期**（务必做 loading 态，别用短超时）

| 操作 | 实测 |
|---|---|
| 只读接口 | 1–3 s（含隧道 RTT ~1 s） |
| LLM 出方案（`/api/ai/design`、`/messages`） | **20–35 s** |
| 单张图生成 | **~36 s**（`cfg=2`；`COMFY_CFG=1` 时约 21 s，但负向词失效） |
| 3 张候选图生成 | **100–190 s**（异步，需轮询；`cfg=2` 后比之前慢约 1.67×） |
| 合成最终海报 | 4–8 s |
| `finalize`（视觉复审 + 有限轮优化） | **30–60 s** |

---

## 1. 系统

### `GET /health`
无参数。

```json
{
  "status": "ok",
  "gpu": { "model": "cuda:0 AMD Radeon Graphics : native",
           "vramTotalGB": 47.98, "vramUsedGB": 32.35 },
  "comfyui": { "status": "ready", "workflowVersion": "1.0.0" },
  "vlm": { "status": "ready", "model": "stageposter-vlm" },
  "runtime": { "goVersion": "go1.25.0" },
  "tokenRequired": false
}
```

⚠️ `vlm.sleeping` 是 `omitempty`：**不睡的时候整个字段不出现**，只在为 `true` 时才有。前端判断请用 `d.vlm?.sleeping === true`，不要指望字段一定存在。当前部署已关闭休眠，正常情况下不会看到它。

### `GET /api/system/dependencies`

```json
{
  "status": "healthy",
  "tokenRequired": false,
  "dependencies": {
    "comfyui":  { "status": "ready" },
    "database": { "status": "ready" },
    "vlm": { "status": "ready", "model": "stageposter-vlm",
             "url": "http://127.0.0.1:8001" }
  },
  "capabilities": {
    "negativePrompt":             { "available": true,  "node": "57:34", "cfg": 2 },
    "referenceImageConditioning": { "available": false, "influences": ["brief_understanding"],
                                    "reason": "workflow has no image input node; ..." },
    "backgroundRemoval":          { "available": false, "reason": "no background removal model wired; ..." },
    "personSimilarityMetric":     { "available": false, "reason": "no face/image embedding model installed" }
  }
}
```

同样地，`vlm.sleeping` 只在为 `true` 时出现。`vlm.url` 是内部地址，前端不要用它去发请求。

`capabilities` 是**本轮新增**的能力矩阵。以前无从判断一个 `false` 是"这次没用上"
还是"整条链路根本不存在" —— 现在每项都带 `reason`。前端据此决定要不要显示对应入口，
不要硬编码。

---

## 2. 海报直接生成（不经 AI 对话）

### `POST /api/posters` → `202`

请求（`event.title` 必填；`style` 目前只支持 `metal-gothic-v1`）：

```json
{
  "event": {
    "title": "混沌冲撞之夜",
    "artist": "铁锈教堂",
    "date": "2026-08-15",
    "time": "20:00",
    "venue": "上海育音堂",
    "presalePrice": "120",
    "doorPrice": "150"
  },
  "visual": {
    "style": "metal-gothic-v1",
    "theme": "工业废墟中的仪式",
    "musicGenre": "post-metal",
    "mood": ["dark", "ritualistic"],
    "preferredColors": ["oxide red", "ink black"]
  },
  "branding": {
    "artistLogoAssetId": "asset_xxx",
    "eventLogoAssetId": "asset_xxx",
    "sponsorLogoAssetIds": ["asset_xxx"]
  }
}
```

`event` 里 **没有** `subtitle` 字段（副标题用 `artist`）。`branding` 可整段省略。

响应即 **PosterResponse**（下面所有海报接口共用这个形状）：

```json
{
  "posterId": "poster_cbe313fc-...",
  "status": "generating_candidates",
  "selectedCandidateId": "candidate_77acb43a-...",
  "resultUrl": "/api/posters/poster_.../result",
  "thumbnailUrl": "/api/posters/poster_.../thumbnail",
  "progress": {
    "completed": 3,
    "total": 3,
    "stage": "awaiting_selection",
    "percent": 65,
    "elapsedSeconds": 84,
    "etaSeconds": 253
  },
  "candidates": [
    {
      "candidateId": "candidate_77acb43a-...",
      "variantKey": "iron-church-forgotten-balanced",
      "variantName": "铁教堂：被遗忘 · Balanced",
      "status": "ready",
      "attempt": 1,
      "selected": true,
      "seed": 1785431970673648270,
      "imageUrl": "/api/posters/poster_.../candidates/candidate_.../image",
      "spec": {
        "variantKey": "iron-church-forgotten-balanced",
        "variantName": "铁教堂：被遗忘 · Balanced",
        "motif": "...",
        "composition": "...",
        "camera": "straight-on medium full shot at eye level, 50mm equivalent, ...",
        "materials": ["cinematic editorial texture", "..."],
        "palette": ["#4D0000", "#1F0505", "#7B2E2E"],
        "lighting": "balanced cinematic lighting with controlled contrast..."
      }
    }
  ],
  "createdAt": "2026-07-30T17:55:12.34Z",
  "updatedAt": "2026-07-30T17:58:02.11Z"
}
```

`spec.palette` 保留原始十六进制，可以直接拿去画色板。
`imageUrl` / `resultUrl` / `thumbnailUrl` 都是**相对路径**，前端拼基址。

**海报状态机**

```
planning_candidates → generating_candidates → validating_candidates
   → partial_ready（1/3 就绪，不是终态，要继续轮询）
   → awaiting_selection → selected → composing
   → succeeded / failed / canceled
```

轮询 `GET /api/posters/{id}`，间隔 3–5 s。见到 `awaiting_selection` 就可以让用户选图。

### `GET /api/posters/{posterId}` → `200` PosterResponse

`progress` 本轮从两个计数扩成了完整进度：

| 字段 | 含义 |
|---|---|
| `completed` / `total` | 候选图计数。**只在候选生成阶段变化**，composing 和复审期间冻住 |
| `stage` | 当前状态字符串，等同顶层 `status`，方便直接做文案映射 |
| `percent` | 整条流水线的推进比例 0–100，不只是候选那一段 |
| `elapsedSeconds` | 自任务创建起的实际耗时 |
| `etaSeconds` | 剩余时间估计。取自历史上已成功海报的**中位总耗时**；样本不足 3 个或任务已到终态时**整个字段消失**（不是 0）—— 用 `?.etaSeconds` 判断 |

实测一次完整流程的 `percent` 走位：`20`（generating）→ `60`（partial_ready）→
`65`（awaiting_selection）→ `100`（succeeded）。
不存在 → `404 {"error":"poster not found"}`
`GET /api/posters`（不带 id）→ `405`

### `GET /api/posters/{posterId}/candidates/{candidateId}/image` → `200 image/png`
1024×1536。候选还没就绪 → `409`。

### `POST /api/posters/{posterId}/select` → `200` PosterResponse

```json
{ "candidateId": "candidate_77acb43a-..." }
```

同步完成合成，返回时 `status` 通常已是 `succeeded`。

### `GET /api/posters/{posterId}/result` → `200 image/png`
最终成品 1024×1536。未到 `succeeded` → `409 {"error":"..."}`；无记录 → `404`。

**文件在磁盘上丢失时现在也返回 `404`**（`{"error":"poster result not found"}`），
之前是 `500`，而且响应体里带着服务器绝对路径。本轮把所有 `500` 分支统一成
`{"error":"internal server error"}`，真实错误只进服务端日志 —— 不要再指望从
`500` 响应体里读到诊断信息。

### `GET /api/posters/{posterId}/thumbnail` → `200 image/png`
512×768。**本轮新增的路由** —— `thumbnailUrl` 之前一直在响应里但没有实现，会 404。

### `GET /api/posters/{posterId}/timeline` → `200`

```json
{
  "posterId": "poster_...",
  "poster": { "...PosterResponse..." },
  "reviews": [ "...PosterReviewRecord..." ],
  "metrics": {
    "reviewRounds": 2,
    "promptTokens": 4814,
    "completionTokens": 1067,
    "totalTokens": 5881,
    "reviewLatencyMs": 42398,
    "wallClockSeconds": 304
  }
}
```

`metrics` 是**本轮新增**的任务级成本汇总。每轮复审的 token 和耗时一直在写库，
但之前没有任何地方加总 —— `AIMessageResponse.metrics` 的作用域只有单次 LLM 调用。
`wallClockSeconds` 包含 GPU 排队和图像生成，这部分不体现在 token 里。

### `GET /api/posters/{posterId}/reviews` → `200`

接受 `?limit=`（1–100，默认 20）和 `?offset=`（默认 0）。

```json
{
  "posterId": "poster_...",
  "reviews": [],
  "count": 1, "total": 2, "limit": 1, "offset": 0
}
```

### `POST /api/posters/{posterId}/review` → `201`
请求体可省略（空体也接受）。触发 Qwen 视觉复审一次。

```json
{ "review": {
    "reviewId": "review_b92f9396-...",
    "posterId": "poster_...",
    "outputId": "poster_output_...",
    "candidateId": "candidate_...",
    "totalScore": 75,
    "scores": { "requirementAlignment": 0, "composition": 0, "typography": 0,
                "readability": 0, "visualQuality": 0, "brandConsistency": 0 },
    "hardFailures": [ { "code": "GENERATED_GIBBERISH_TEXT", "description": "中文描述" } ],
    "issues": [ { "code": "TITLE_COLLISION", "severity": "high", "layer": "composition",
                  "description": "中文描述", "suggestion": "中文建议" } ],
    "decision": "RECOMPOSE"
} }
```

`decision` ∈ `ACCEPT` / `RECOMPOSE` / `REGENERATE` / `REWRITE_BRIEF`。`ACCEPT` 需要 `totalScore ≥ 82` 且无硬失败。

### `POST /api/posters/{posterId}/cancel` → `200`

```json
{ "posterId": "poster_...", "status": "succeeded" }
```

返回的是**真实状态**。终态（`succeeded`/`failed`/`canceled`）上取消是无操作 —— 之前这里写死返回 `"canceled"`，会骗前端，本轮已修。

### `POST /api/posters/{posterId}/candidates/{candidateId}/retry` → `202`
重跑单个候选。已就绪的候选上调用 → `409`。

---

## 3. AI 对话式生成（推荐主流程）

### `POST /api/ai/design` → `200`
一次性拿 3 个设计方案，**不建会话**。适合"换一批方案"按钮。耗时 20–35 s。

```json
{
  "event":  { "title": "混沌冲撞之夜", "artist": "铁锈教堂", "date": "2026-08-15",
              "time": "20:00", "venue": "上海育音堂" },
  "visual": { "style": "metal-gothic-v1", "theme": "工业废墟中的仪式",
              "musicGenre": "post-metal", "mood": ["dark"] },
  "message": "想要压抑一点"
}
```

**只有 `event` / `visual` / `message` 三个字段，没有 `branding`。**

```json
{ "result": {
    "reply": "收到，已为您构思三个……",
    "state": "awaiting_plan_selection",
    "missingFields": [],
    "plans": [
      {
        "id": "corroded-monolith",
        "name": "锈蚀巨碑",
        "concept": "中文视觉概念描述",
        "palette": ["#4D0000", "#1F0505", "#7B2E2E"],
        "composition": { "subject": "center", "symmetry": "strong",
                         "titleSafeZone": "top_20_percent",
                         "informationSafeZone": "bottom_22_percent" },
        "positivePrompt": "English prompt ...",
        "negativePrompt": "English negative ...",
        "composerTemplate": "editorial_top"
      }
    ]
} }
```

`composerTemplate` ∈ `editorial_top` / `cinematic_center` / `gothic_frame`。
恒定返回 3 个方案；`id` 已规范化成小写 kebab。

### `POST /api/ai/sessions` → `201` SessionResponse

```json
{ "brief": {
    "event":  { "title": "...", "artist": "...", "date": "...", "time": "...",
                "venue": "...", "presalePrice": "120", "doorPrice": "150" },
    "visual": { "style": "metal-gothic-v1", "theme": "...", "musicGenre": "...",
                "mood": ["dark","ritualistic"], "preferredColors": ["oxide red"] },
    "branding": {}
  },
  "assets": [ { "assetId": "asset_xxx", "purpose": "artist_logo" } ]
}
```

**注意是 `{"brief": {...}}` 包一层**，不是把 event/visual 摊平。`assets` 可省略。
`branding` 必须给（可以是空对象 `{}`）。

**SessionResponse**（所有会话接口共用）：

```json
{
  "sessionId": "session_8aa67f7b-...",
  "status": "awaiting_plan_selection",
  "availableActions": ["send_message", "confirm_plan", "cancel"],
  "brief": { "event": {...}, "branding": {...}, "visual": {...} },
  "missingFields": ["visual.mood"],
  "selectedPlanId": "corroded-monolith",
  "posterId": "poster_bbe271b4-...",
  "reviewSummary": {
    "finalized": true, "accepted": false, "rounds": 2,
    "bestRound": 2, "bestScore": 75, "latestDecision": "RECOMPOSE",
    "warning": "Maximum review rounds reached ..."
  },
  "messages": [ { "messageId": "message_...", "sessionId": "session_...",
                  "role": "user", "content": "...", "createdAt": "..." } ],
  "assets": [ { "assetId": "asset_...", "purpose": "artist_logo",
                "actuallyUsed": false } ],
  "plans": [ { "sessionId": "...", "planId": "corroded-monolith",
               "plan": { ...同 /api/ai/design 的 plan... } } ],
  "poster": { ...PosterResponse... },
  "createdAt": "...", "updatedAt": "..."
}
```

**关键：候选图在 `session.poster.candidates` 里**，不在顶层 `candidates`。

**用 `availableActions` 驱动 UI**，别自己推状态。取值：`send_message`、`attach_asset`、`confirm_plan`、`select_candidate`、`finalize`、`download_final`、`refresh`、`cancel`。

**会话状态机**

```
collecting_brief → planning → awaiting_plan_selection
  → generating_candidates → awaiting_candidate_selection
  → selected → composing → reviewing
  → succeeded / completed_with_warnings / failed / canceled
```

终态取消统一拼 **`canceled`**（单 l），与海报、任务一致。会话曾经返回英式双 l 的 `cancelled`，本轮已统一；读取侧仍会把库里的旧值折叠成 `canceled`，所以不会再有两种拼法出现在同一个字段里。

### `GET /api/ai/sessions/{sessionId}` → `200` SessionResponse
不存在 → `404 {"error":"AI session not found"}`
`GET /api/ai/sessions`（不带 id）→ `405`

### `POST /api/ai/sessions/{sessionId}/messages` → `200`
耗时 20–35 s。

```json
{ "content": "铁锈红为主，压抑" }
```

响应是 `{"session": {...SessionResponse...}}`（**多包一层 `session`**）。

若 `brief` 不完整，会停在 `collecting_brief` 并给出 `missingFields`（如 `["visual.mood"]`）—— 这是正常行为，把缺的问出来再发下一条。

### `POST /api/ai/sessions/{sessionId}/assets` → `200` SessionResponse

```json
{ "assets": [ { "assetId": "asset_xxx", "purpose": "artist_logo" } ] }
```

`purpose` 只接受 `performer` / `artist_logo` / `event_logo` / `sponsor_logo` / `reference`。
传别的 → `400 {"error":"invalid AI session asset purpose: logo"}`

### `POST /api/ai/sessions/{sessionId}/plans/{planId}/confirm` → `202` SessionResponse
确认方案，开始生成 3 张候选。返回后 `status=generating_candidates`、`posterId` 已就位，之后轮询会话或直接轮询 `GET /api/posters/{posterId}`。

### `POST /api/ai/sessions/{sessionId}/candidates/{candidateId}/select` → `200` SessionResponse
选定候选并合成。返回后通常 `status=succeeded`、`availableActions=["finalize","download_final"]`。

### `POST /api/ai/sessions/{sessionId}/finalize` → `200` SessionResponse
触发视觉复审 + 有限轮自动优化，耗时 30–60 s。
结果看 `reviewSummary`：`accepted=true` → `succeeded`；否则 `completed_with_warnings` 并保留最佳版本。

### `POST /api/ai/sessions/{sessionId}/cancel` → `200` SessionResponse
`status` 变 `canceled`。

未定义的子路径 → `404 {"error":"AI session route not found"}`

---

## 4. 素材

### `POST /api/assets` → `201`
`multipart/form-data`，字段 `file` + `kind`。上限 20 MiB。

`kind` 只接受 **`person` / `logo` / `reference`**，传别的 → `400 {"error":"kind must be person, logo, or reference"}`

注意：这里的 `kind` 和会话绑定用的 `purpose` 是**两套枚举**。

```json
{
  "assetId": "asset_6c4c1c4a-...",
  "kind": "logo",
  "originalName": "logo.png",
  "filename": "asset_6c4c1c4a-....png",
  "mimeType": "image/png",
  "sizeBytes": 2300921,
  "sha256": "756cdc41...",
  "contentUrl": "/api/assets/asset_6c4c1c4a-.../content",
  "width": 1024, "height": 1536,
  "processStatus": "ready",
  "processedAt": "2026-07-30T16:27:48.26Z",
  "processVersion": "v1",
  "createdAt": "2026-07-30T16:27:48.25Z"
}
```

### `GET /api/assets` → `200`

接受 `?limit=`（1–100，默认 20）和 `?offset=`（默认 0）。

```json
{
  "items": [ "...Asset..." ],
  "count": 2, "total": 20, "limit": 2, "offset": 0
}
```

`count` 是**当前页**行数，`total` 是表内总行数 —— 用 `offset + count < total` 判断
还有没有下一页。之前只有 `count`，客户端没法翻页；CLAUDE.md 里写的单个 `total`
也和代码不符，两边都已改成如实描述。

越界参数返回 **400**（`limit must be between 1 and 100` / `offset must not be
negative`），不再像以前那样被静默改成默认值 —— 传 `limit=1000` 却拿到 20 行且毫无
提示，是很难查的坑。

Asset 现在带 `cutout`：

```json
"cutout": { "status": "ready", "hasAlpha": true }
```

`status` 取值：`ready`（自带可用透明通道）、`opaque`（完全不透明，会以矩形压在海报上）、
`unsupported`（还没处理）、`failed`（解码失败）。

### `GET /api/assets/{assetId}` → `200` Asset ／ `404`

### `GET /api/assets/{assetId}/content` → `200` 原始图片字节

---

## 5. 任务（legacy，调试用）

`GET /api/jobs` → `{"items":[...],"count":n,"total":n,"limit":n,"offset":n}`，
接受 `?limit=` / `?offset=`。item 字段：`jobId`、`promptId`、`status`、`prompt`、
`negativePrompt`、`seed`、`workflowKey`、`workflowVersion`、`image`、`resultUrl`、
`createdAt`、`startedAt`、`completedAt`、`updatedAt`

`GET /api/jobs/{jobId}` → `200`

`GET /api/jobs/{jobId}/result` → `200 image/png`。文件丢失时 `404`
（`{"error":"job result not found"}`），之前是带绝对路径的 `500`。

`GET /api/jobs/{jobId}/thumbnail` — **不存在**。只有海报有缩略图路由；这个路径会落到
job id 解析上，返回 `404 {"error":"job not found"}`。

`POST /api/generate` — 旧的单图直生成路径，新前端不要用。它现在会把
`negativePrompt` 真正传给 ComfyUI（见能力矩阵）。

---

## 6. 已实现 vs 未实现

### 已实现并实测通过

- 结构化 brief → 3 个设计方案（Qwen）→ 3 张候选图（ComfyUI + Z-Image Turbo）→ 用户选图 → 确定性中文排版合成 → 视觉复审 + 有限轮优化 → 成品 + 缩略图
- 中文全链路：请求 → 存库 → prompt → 排版渲染 → 响应，UTF-8 无损；海报上是真实中日韩字形（Noto CJK）
- 三个候选在主体取景、镜头（50/85/28mm）、构图、材质、灯光、配色侧重上真正拉开
- 结构化 `spec`（motif/composition/camera/materials/palette/lighting）随候选返回，前端可直接展示
- 标题区亮度自适应压暗：按底图 90 分位亮度反解压暗强度，保证白标题对比度 ≥ 6:1
- 生成图不含任何文字（禁令点名拉丁字母 + 中日韩字形；色值不入 prompt）
- 复审给出结构化评分 / 硬失败 / 问题清单 / 决策
- 素材上传、去重（sha256）、尺寸探测、异步处理状态
- **负向提示词真正生效**：工作流新增负向 `CLIPTextEncode`（`57:34`）接进采样器 negative，`cfg=2`。同 seed 只改负向词，出图哈希不同（实测），说明负向分支确实进了采样
- **列表接口分页**：`limit` + `offset`，信封带 `count` / `total` / `limit` / `offset`
- **抠图透明度实测状态**：`cutout.hasAlpha` 是真的解码图片查到的结果，不再是伪造蒙版
- **素材使用证据**：`actuallyUsed` / `usedInStage` 由真实调用驱动（VLM 视觉输入、合成器实际绘制）
- **任务级 metrics**：`GET /api/posters/{id}/timeline` 返回累计轮数 / token / 复审耗时 / 墙上时间
- **子阶段进度与 ETA**：`progress.stage` / `percent` / `elapsedSeconds` / `etaSeconds`
- **能力矩阵**：`GET /api/system/dependencies` 的 `capabilities` 如实报告哪些能力没接通

### 未实现 —— 前端不要依赖

这一节现在**只剩下真正被外部依赖卡住的项**。判断依据不用猜，直接读
`GET /api/system/dependencies` 的 `capabilities`。

| 项 | 现状 | 卡在哪 |
|---|---|---|
| **参考图条件化（进扩散过程）** | `capabilities.referenceImageConditioning.available = false`。参考图**只**影响需求理解那次 VLM 调用（每次请求最多一张图），不进入扩散 | 工作流里没有任何图像输入节点；`ComfyUI/models/` 下 `clip_vision` / `controlnet` / `style_models` 全是空的，也没有 IPAdapter 目录和自定义节点包；本环境网络在自签名证书代理后面，下不了模型 |
| **背景去除 / 抠图** | `capabilities.backgroundRemoval.available = false`。`/api/assets/{id}/process` 的 `background_removal` 步骤现在如实返回 `skipped` | 没有 rembg / matting 模型。合成器只按素材自带 alpha 叠加（`xDraw.Over`），不会自己抠背景 |
| **人物相似度指标** | `capabilities.personSimilarityMetric.available = false` | 没有人脸 / 图像嵌入模型 |

`maskPath` 字段仍在响应里，但**现在恒为空**：它以前存的不是蒙版，而是源文件的逐字节
副本（扩展名硬编码成 `.png`，JPEG 上传会得到一个装着 JPEG 字节的 `.png`），没有任何
代码消费过它。留着那个值等于对外宣称抠图做过了，已在迁移里清空。要判断透明度请用
`cutout`。

### 已知残留问题

1. **`cfg=2` 让出图变慢约 1.67×**。实测单张 36 s，`cfg=1` 时是 21.6 s。这是负向词
   生效的代价（CFG 每步要多跑一次前向）。想换回纯 turbo 速度就设 `COMFY_CFG=1`，
   代价是负向词重新变成惰性 —— 后端启动时会打 warning 告知，不会静默失效。
2. **`cfg=2` 下负向词的抑制强度是温和的**。实测同 seed 加上
   `purple, cyan, neon, crowd, people` 后画面确实变了（哈希不同），但紫/青色调没被
   完全消掉。要更强的抑制得继续抬 `COMFY_CFG`，需要自己权衡画质。
3. **复审升级逻辑已改但未经线上大样本验证**。`layer` 字段现在优先于关键词匹配，
   且 `visualQuality` 明显低于排版分时会从 `RECOMPOSE` 升级成 `REGENERATE`
   （单测覆盖了这几条路径）。真实复审样本上的效果还需要观察。
4. **艺术家名偶尔和主视觉主体挤在一起**（本轮实测的成品里 "Vela" 压在王座底座上）。
   属于 `RECOMPOSE` 能处理的排版类问题。

## 7. 前端接入清单

1. `VITE_API_BASE_URL` = 上面那个隧道地址，不要硬编码进源码
2. 先打 `GET /health`，用 `tokenRequired` 决定要不要带 token
3. 所有 JSON 请求体**严格按字段名**，多一个字段就 400
4. LLM 和生成类请求超时设到 **≥ 300 s**
5. 生成中的轮询用 `GET /api/posters/{id}`，3–5 s 一次；`partial_ready` 不是终态，要继续轮
6. 会话 UI 用 `availableActions` 驱动，不要自己推状态机
7. 候选图从 `session.poster.candidates` 取，不是顶层
8. 图片 URL 都是相对路径，需要拼基址
9. `actuallyUsed` / `usedInStage` 现在是**真实证据**，可以直接显示。`false` 的含义是
   "这次确实没用上"，不再是"功能没做"
10. 要判断某个能力在不在，读 `GET /api/system/dependencies` 的 `capabilities`，
    不要靠猜或硬编码
11. 列表接口翻页用 `?limit=&offset=`；越界会返回 **400** 而不是静默截断。用
    `offset + count < total` 判断还有没有下一页
12. logo 请上传**透明 PNG**。上传后看 `cutout.hasAlpha`：`false` 表示这个 logo 会以
    不透明矩形压在海报上（后端不会替你抠背景）
