## 图片鉴权
如果启用了：
POSTER_API_TOKEN=poster-dev-2026
浏览器直接：
<img src="/api/posters/.../image">
不能自动带 X-Poster-Token。
所以当前需要三选一：
本地 Demo 不开启 Backend Token。
前端使用 fetch 带 Header，再转成 Blob URL。
后端生成短时签名图片 URL。
黑客松 MVP 推荐第 1 或第 2 种。正式环境再做签名 URL 或 Cookie 鉴权
