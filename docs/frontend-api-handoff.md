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
| 鉴权 | 当前 `POSTER_API_TOKEN` 为空 = **无需鉴权**。`GET /health` 的 `tokenRequired` 字段会告诉你要不要带。开启后三种方式任选：`Authorization: Bearer <token>` / `X-Poster-Token: <token>` / `?token=<token>` |
| CORS | `Access-Control-Allow-Origin: *`，允许头 `Content-Type, Authorization, X-Poster-Token`，方法 `GET, POST, OPTIONS`，预检返回 `204` |
| 时间 | ISO 8601 UTC，如 `2026-07-30T17:55:12.34Z` |
| ID 前缀 | `poster_` / `candidate_` / `session_` / `asset_` / `job_` / `message_` / `review_` |

**耗时预期**（务必做 loading 态，别用短超时）

| 操作 | 实测 |
|---|---|
| 只读接口 | 1–3 s（含隧道 RTT ~1 s） |
| LLM 出方案（`/api/ai/design`、`/messages`） | **20–35 s** |
| 3 张候选图生成 | **80–170 s**（异步，需轮询） |
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
  }
}
```

同样地，`vlm.sleeping` 只在为 `true` 时出现。`vlm.url` 是内部地址，前端不要用它去发请求。

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
  "progress": { },
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

### `GET /api/posters/{posterId}/thumbnail` → `200 image/png`
512×768。**本轮新增的路由** —— `thumbnailUrl` 之前一直在响应里但没有实现，会 404。

### `GET /api/posters/{posterId}/timeline` → `200`

```json
{ "poster": { ...PosterResponse... }, "events": [ ... ] }
```

### `GET /api/posters/{posterId}/reviews` → `200`

```json
{ "posterId": "poster_...", "reviews": [] }
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
  → succeeded / completed_with_warnings / failed / cancelled
```

⚠️ 会话终态拼作 **`cancelled`**（英式双 l），而海报和任务用 `canceled`（单 l）。不一致，暂未统一以免破坏 API。

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
`status` 变 `cancelled`。

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
`{ "items": [ ...Asset... ], "count": 12 }`

⚠️ 信封字段是 **`count`**，不是 `total`（CLAUDE.md 里写的 `total` 是错的，实测为 `count`）。

### `GET /api/assets/{assetId}` → `200` Asset ／ `404`

### `GET /api/assets/{assetId}/content` → `200` 原始图片字节

---

## 5. 任务（legacy，调试用）

`GET /api/jobs` → `{"items":[...],"count":n}`，item 字段：`jobId`、`promptId`、`status`、`prompt`、`negativePrompt`、`seed`、`workflowKey`、`workflowVersion`、`image`、`resultUrl`、`createdAt`、`startedAt`、`completedAt`、`updatedAt`
`GET /api/jobs/{jobId}` → `200`
`GET /api/jobs/{jobId}/result` → `200 image/png`
`POST /api/generate` — 旧的单图直生成路径，新前端不要用。

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

### 未实现 —— 前端不要依赖

| 项 | 现状 |
|---|---|
| **参考图条件化** | `assets[].actuallyUsed` **永远是 `false`，这是诚实的**。工作流只绑定 prompt/negative/seed，参考图完全不影响生成。要做得改 ComfyUI 拓扑（IPAdapter / ControlNet），是最重的一项 |
| 抠图 / logo 透明化 / 参考图分析 | 无独立状态，`maskPath` 里的"蒙版"目前只是源图副本 |
| 人物相似度指标 | 无 |
| 子阶段进度 / ETA | `progress` 字段存在但内容很薄，没有细粒度阶段和预计剩余时间 |
| 任务级 metrics（耗时 / token / 显存） | 未透出 |
| `usedInStage` 之类的素材使用证据 | 无 |
| 负向提示词实际生效 | 计算了、存库了、**没发给 ComfyUI**。工作流采样器 `cfg=1`，负向输入接 `ConditioningZeroOut`，没有负向编码节点。要生效必须加节点并抬 CFG，会破坏 turbo 蒸馏、推理翻倍 |
| 分页参数 | 列表接口返回 `items` + `count`，但不接受 `page` / `limit`，也没有游标 |

### 已知残留问题

1. **下部条带偶尔被画成纯白色块**。`cfg=1` 让负向词失效，压不住。当前被 composer 的信息面板（默认从 77% 高度起）完整遮住，成品看不出来；但候选图预览里能看到。
2. **会话终态 `cancelled` 与海报 `canceled` 拼写不一致**。
3. **`finalize` 常返回 `completed_with_warnings`**（实测得分 75，阈值 82）。复审能识别问题，但 `RECOMPOSE`（重排版）救不了已经烤进图里的瑕疵 —— 那种情况需要 `REGENERATE`。

---

## 7. 前端接入清单

1. `VITE_API_BASE_URL` = 上面那个隧道地址，不要硬编码进源码
2. 先打 `GET /health`，用 `tokenRequired` 决定要不要带 token
3. 所有 JSON 请求体**严格按字段名**，多一个字段就 400
4. LLM 和生成类请求超时设到 **≥ 300 s**
5. 生成中的轮询用 `GET /api/posters/{id}`，3–5 s 一次；`partial_ready` 不是终态，要继续轮
6. 会话 UI 用 `availableActions` 驱动，不要自己推状态机
7. 候选图从 `session.poster.candidates` 取，不是顶层
8. 图片 URL 都是相对路径，需要拼基址
9. 别显示 `actuallyUsed`，或者明确标成"未启用"
