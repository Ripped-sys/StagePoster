# StagePoster — Claude Code 指引

> 项目根目录：`/workspace/poster-engine/`
> Go 模块：`github.com/Ripped-sys/StagePoster/backend`
> 目标硬件：AMD Radeon PRO W7900 48 GB + ROCm 7.2

---

## 1. 项目概览

StagePoster 是一个 **AI 原生音乐活动海报引擎**。核心流程：

```
结构化活动 Brief
        ↓
Qwen3.5-9B 艺术指导 Agent
        ↓
3 个结构化设计方案
        ↓
ComfyUI + Z-Image Turbo 生成 3 张候选图
        ↓
用户选择候选图
        ↓
Go 确定性文字 / Logo / 信息排版
        ↓
Qwen Vision 视觉审查与有限轮自动优化
        ↓
最终海报 + 缩略图 + 审查证据
```

前端只与 Go Backend 通信。ComfyUI 和 vLLM 是内部服务，不暴露给浏览器。

---

## 2. 技术栈

| 层 | 技术 | 端口 |
|---|---|---|
| Go Backend | Go 1.25.0 + `net/http` | `:8080` |
| 数据库 | SQLite (modernc.org/sqlite) | 本地文件 |
| 图像生成 | ComfyUI + Z-Image Turbo | `:8188` |
| LLM / VLM | vLLM + Qwen3.5-9B | `:8001` |
| GPU 驱动 | ROCm 7.2 + HIP | — |
| 对外隧道 | Cloudflare Quick Tunnel | HTTPS |

Go 依赖极少（仅 `golang.org/x/image` + `modernc.org/sqlite`），无外部 Web 框架。

---

## 3. 目录结构

```
/workspace/poster-engine/
├── ComfyUI/                         # ComfyUI fork（git submodule）
│   └── models/
│       ├── diffusion_models/
│       │   └── z_image_turbo_bf16.safetensors
│       ├── text_encoders/
│       │   └── qwen_3_4b.safetensors
│       ├── vae/
│       │   └── ae.safetensors
│       └── loras/
│           └── z_image_turbo_distill_patch_lora_bf16.safetensors
├── models/
│   └── Qwen3.5-9B/                  # vLLM 模型文件
├── workflows/
│   └── z_image_poster_v1.json       # ComfyUI 工作流模板
├── venv/                            # ComfyUI Python 3.10.20 环境
├── .venv-vllm/                      # vLLM Python 3.12 环境
├── .env                             # 项目级环境变量（服务端口、密钥等）
├── scripts/                         # 部署与服务管理脚本
│   ├── install-all.sh               # 一键部署
│   ├── start-all.sh                 # 启动所有服务
│   ├── stop-all.sh                  # 停止所有服务
│   ├── status.sh                    # 查看服务状态
│   ├── smoke-test.sh                # 冒烟测试
│   └── ...
└── backend/
    ├── .env.example                 # 后端环境变量模板
    ├── go.mod / go.sum
    ├── cmd/server/
    │   └── main.go                  # 入口：服务初始化 + 启动
    ├── data/
    │   └── poster.db                # SQLite 数据库
    ├── logs/                        # 日志目录
    ├── run/                         # PID 文件 + tunnel URL
    ├── storage/
    │   ├── jobs/                    # ComfyUI 任务输出
    │   ├── assets/                  # 上传素材存储
    │   └── posters/                 # 最终海报输出
    └── internal/                    # Go 内部包（不可外部导入）
        ├── api/                     # HTTP 路由 + 处理器
        │   ├── server.go            # Server 结构体 + 路由组装
        │   ├── posters.go           # 海报相关端点
        │   ├── assets.go            # 素材相关端点
        │   └── ai_sessions.go       # AI 会话端点
        ├── domain/                  # 领域模型 + 状态常量
        │   ├── poster.go            # PosterStatus, CandidateStatus, 请求/响应结构体
        │   ├── ai_session.go        # AISessionStatus, 会话结构体
        │   └── asset.go             # AssetStatus, 素材结构体
        ├── repository/              # SQLite 数据访问层
        │   ├── sqlite.go            # 连接 + 迁移
        │   ├── posters.go           # poster_requests / poster_candidates 表
        │   ├── ai_sessions.go       # ai_sessions / ai_messages 表
        │   └── assets.go            # assets 表
        ├── service/                 # 核心业务逻辑（无 HTTP）
        │   ├── poster.go            # PosterService：ComfyUI 提交 + 轮询
        │   ├── assets.go            # AssetService：素材处理
        │   ├── asset_processor.go   # 异步素材处理 goroutine
        │   └── health.go            # HealthCollector：各组件健康检查
        ├── poster/                  # 海报高级流程编排
        │   ├── service.go           # poster.Service：编排 planner + evaluator + composer
        │   ├── planner.go           # 设计规划：brief → 3 个 variant spec
        │   ├── evaluator.go         # 候选图视觉评估
        │   ├── composition.go       # 排版引擎接口
        │   ├── composer.go          # 排版实现（Go 原生 image/draw）
        │   ├── finalization.go      # 最终化流程
        │   └── reviews.go           # 视觉审查循环
        ├── assistant/               # AI 会话状态机
        │   ├── service.go           # AI Session 主流程
        │   ├── finalize.go          # finalize 逻辑
        │   └── keyed_lock.go        # 会话级锁
        ├── ai/                      # vLLM 客户端 + Runtime
        │   ├── client.go            # vLLM OpenAI 兼容 API 客户端
        │   ├── service.go           # LLM/VLM 调用封装
        │   └── runtime.go           # GPU 显存管理（sleep/wake）
        ├── comfy/                   # ComfyUI 客户端
        │   ├── client.go            # 提交 prompt + 轮询 history
        │   ├── workflow.go          # Workflow JSON 模板加载 + 填充
        │   └── client_memory_test.go # ComfyUI 内存压力测试
        ├── composer/                # 海报排版引擎
        │   └── composer.go          # 文字/Logo/条形码渲染
        ├── storage/                 # 文件系统抽象
        │   ├── filestore.go         # ComfyUI 任务输出存储
        │   ├── assetstore.go        # 素材文件存储（20MB 上限）
        │   └── posterstore.go       # 最终海报文件存储
        └── worker/                  # 后台 goroutine
            ├── reconciler.go        # ComfyUI 任务状态轮询
            └── poster_reconciler.go # 海报流程状态机推进
```

---

## 4. 核心流程

### 4.1 海报生成状态机

```
planning_candidates → generating_candidates → validating_candidates
                                                    ↓
                                              partial_ready (1/3 ready)
                                                    ↓
                                              awaiting_selection
                                                    ↓
                                              selected → composing
                                                    ↓
                                              succeeded / failed / canceled
```

- `worker/reconciler.go`：每 2 秒轮询 ComfyUI job 状态，将 `generating` → `ready`/`failed`
- `worker/poster_reconciler.go`：推进海报流程状态机，触发 composition、review、finalize
- **关键**：`PartialReady` 不算终态，reconciler 必须继续处理剩余候选图

### 4.2 AI 会话状态机

```
collecting_brief → planning → awaiting_approval → generating
                                                    ↓
                                              awaiting_candidate_selection
                                                    ↓
                                              selected → composing → reviewing
                                                    ↓
                                              succeeded / failed / canceled
```

- `assistant/service.go`：管理 AI 会话生命周期
- 用户消息触发 LLM 调用，LLM 决定下一步动作（规划 / 确认 / 选择 / finalize）

---

## 5. 开发工作流

### 编译

```bash
cd /workspace/poster-engine/backend
go build -o poster-backend ./cmd/server
```

### 运行测试

```bash
cd /workspace/poster-engine/backend
go test ./...
```

### 启动服务

```bash
cd /workspace/poster-engine
./scripts/start-all.sh          # 启动 ComfyUI + vLLM + Backend
./scripts/status.sh             # 检查各服务状态
```

### 冒烟测试

```bash
cd /workspace/poster-engine
./scripts/smoke-test.sh          # 组件级冒烟测试
# 完整 E2E 见 docs/one-click-deployment.md
```

### 一键部署

```bash
cd /workspace/poster-engine
sudo -E bash scripts/install-all.sh
```

---

## 6. 环境变量

`.env` 位于项目根目录。关键变量：

| 变量 | 默认值 | 说明 |
|---|---|---|
| `LISTEN_ADDR` | `:8080` | Backend 监听地址 |
| `COMFY_URL` | `http://127.0.0.1:8188` | ComfyUI 地址 |
| `VLM_URL` | `http://127.0.0.1:8001` | vLLM 地址 |
| `VLM_API_KEY` | `stageposter-vlm-local` | vLLM API Key |
| `VLM_MODEL` | `stageposter-vlm` | vLLM 模型名 |
| `DB_PATH` | `backend/data/poster.db` | SQLite 路径 |
| `STORAGE_ROOT` | `backend/storage/jobs` | 任务输出目录 |
| `WORKFLOW_PATH` | `workflows/z_image_poster_v1.json` | ComfyUI 工作流 |
| `WORKFLOW_KEY` | `poster-text` | 工作流标识 |
| `WORKFLOW_VERSION` | `1.0.0` | 工作流版本 |
| `POSTER_API_TOKEN` | `""` | API 认证 token |
| `CORS_ORIGIN` | `*` | CORS 来源 |
| `POSTER_FONT_REGULAR` | `""` | 正文字体路径（系统字体名） |
| `POSTER_FONT_BOLD` | `""` | 粗体字体路径 |
| `RECONCILE_INTERVAL` | `2s` | reconciler 轮询间隔 |
| `PROMPT_NODE_ID` | `57:27` | ComfyUI prompt 节点 |
| `NEGATIVE_PROMPT_NODE_ID` | `""` | ComfyUI negative prompt 节点 |
| `SEED_NODE_ID` | `57:3` | ComfyUI seed 节点 |

---

## 7. 已知约束与坑

### ROCm / GPU
- vLLM `--enable-sleep-mode` 在 ROCm 7.2 + vLLM 0.20.0 上有 `CUDA Error: invalid argument` 问题，**不要启用**
- ComfyUI 和 vLLM 共享单张 W7900。Qwen 唤醒前会自动卸载 ComfyUI 模型（`ReleaseComfyMemory`）
- GPU 架构 `gfx1100`，确保 `rocm-smi` 显示正确

### ComfyUI
- Workflow JSON 中的节点 ID 是固定的（通过 `PROMPT_NODE_ID` / `SEED_NODE_ID` 配置）
- 任务提交后通过 `/history/{prompt_id}` 轮询输出
- 输出图片存储在 ComfyUI 的输出目录，通过 `storage.FileStore` 复制到 `STORAGE_ROOT`

### SQLite
- 使用 `modernc.org/sqlite`（纯 Go 实现，无需 CGO）
- 迁移在 `repository/sqlite.go` 的 `Migrate` 中定义
- 所有查询使用 `context.Context` 超时控制

### 并发
- 两个 reconciler goroutine（`worker/reconciler.go` + `worker/poster_reconciler.go`）每 2 秒Tick一次
- **必须有 panic recovery** — 否则 goroutine 静默死亡
- 会话级操作使用 `sync.Mutex` 防止并发竞态

---

## 8. API 设计约定

- 所有 POST 请求体为 JSON
- 错误响应：`{"error": "message"}` + 对应 HTTP status
- 成功响应：`{"data": ...}` 或直接返回资源对象
- 分页：列表端点返回 `items []` + `total int`
- 认证：`Authorization: Bearer <POSTER_API_TOKEN>`（可选，生产环境建议启用）

---

## 9. 代码风格

- **包名**：`internal/*` 下的小写短名（`api`, `domain`, `service`, `poster`, `assistant` 等）
- **错误**：使用 `errors.New` 定义包级 sentinel error，不要用字符串比较
- **上下文**：所有耗时操作接收 `context.Context`，HTTP handler 从 `r.Context()` 传递
- **日志**：使用标准库 `log`，生产环境建议替换为结构化日志
- **ID 生成**：使用 `domain.NewID(prefix)` 生成 `prefix_xxxxxxxx` 格式 ID
- **JSON**：字段名使用 `json:"snake_case"`，API 响应与请求保持一致
- **文件路径**：绝对路径从环境变量读取，不要硬编码

---

## 10. 修改检查清单

修改代码后确认：
- [ ] `go build ./cmd/server` 编译通过
- [ ] `go test ./...` 全部通过
- [ ] 如有数据库变更，在 `repository/sqlite.go` 添加迁移语句
- [ ] 如有新环境变量，在 `.env.example` 和 `main.go` 的 `env()` 调用中添加
- [ ] 如有新的 ComfyUI 节点绑定，更新 `scripts/` 中的节点 ID 注释
- [ ] 冒烟测试通过 `./scripts/smoke-test.sh`
