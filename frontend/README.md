# Poster 前端

Poster 是面向乐队、演出与活动的 AI Visual Studio。用户通过场景表单或真实 AI Session 对话录入信息与素材，生成三张候选主视觉，并由程序化图层保留标题、时间、地点、Logo 与二维码。

## 技术栈

- React 18、Vite 6、TypeScript
- React Router、Framer Motion
- 普通 CSS 与 CSS Variables
- lucide-react、html-to-image
- localStorage 项目和任务持久化
- Playwright 端到端验收

## 启动与检查

在 `frontend` 目录执行：

```bash
pnpm install
pnpm dev
```

默认开发地址为 `http://127.0.0.1:5173/`。仓库根目录的 `start-poster.cmd` 会使用演示固定地址 `http://127.0.0.1:4173/`。

```bash
pnpm lint
pnpm build
pnpm test:e2e
pnpm preview
```

常规端到端测试会自动启动本地服务，覆盖桌面与手机布局、四步创建、图片上传、校验、Mock 回退、结果编辑和 PNG 下载。视觉证据输出到 `artifacts/ui-audit/`。

真实 AI Session 回归需要一个已经跑通的 Session：

```powershell
$env:LIVE_SESSION_ID='session_xxx'
pnpm exec playwright test tests/live-session.spec.ts
```

## 页面路由

- `/`：沉浸式 Landing Page、场景入口、工作流和案例
- `/create`：四步创建工作台与对话式项目助手
- `/create?demo=1`：长安双雄演示项目
- `/create?project=:id`：继续编辑已保存项目
- `/generate/:id`：远端任务轮询或本地 Mock 生成进度
- `/result/:id`：检查、编辑、性能数据和 PNG 导出

## 已实现

- 六种场景及音乐演出、讲座动态字段
- 分类素材上传、预览、校验、替换和删除
- 乐队卡片增删及独立 Logo、成员照片素材
- Underground Rock、Cyber Neon、Editorial Minimal 风格预设
- 真实 AI 双向消息、缺失字段、三套设计方案和 `availableActions` 状态驱动
- 方案确认、三张候选图轮询、AI Session 候选选择、VLM Finalize 和安全恢复
- 人物、Logo、参考图上传到后端 `/api/assets` 并绑定 AI Session
- AI brief 由用户确认后同步到表单，不自动覆盖最终信息
- 选中 AI 主视觉后，用前端确定性信息层生成并导出发布版 PNG
- 输出选项、必填校验、错误定位及素材状态保留
- 生成阶段、刷新恢复、结果信息即时编辑与真实 PNG 下载
- 桌面三栏与移动单栏响应式布局

## 后端联调

通过环境变量配置 StagePoster AI Session 服务。Quick Tunnel 会变化，不应写入生产代码：

```text
VITE_API_BASE_URL=https://your-stageposter-host.example.com
```

正常业务流程全部走 `/api/ai/sessions`：创建 Session、持续发送消息、确认方案、轮询三张候选图、选择候选、Finalize、下载最终图。按钮只根据后端 `availableActions` 显示，图片相对路径统一拼接 `VITE_API_BASE_URL`。素材先上传 `/api/assets`，再绑定到 Session。

## 仍为 Mock 或规划中

- 四步工作台原有的“AI 帮我推荐”按钮仍是本地风格规则；右侧 AI Creative Agent 已是真实后端对话
- `/generate/:id` 旧流程在无 Token 时仍保留本地 Mock，真实完整流程位于 `/create` 的 AI Creative Agent
- 人物与参考图可以上传和绑定，但当前后端候选生成尚未证明具备严格身份保持/人物抠图能力
- 后端中文字体缺字、长信息拥挤和默认票务文案问题由前端“精确信息发布版”规避；后端原始结果仍可展开审查
- 5–8 秒宣传片为 P1 规划能力，必须由用户主动选择
- VJ 动态视觉为 P2，当前禁用
