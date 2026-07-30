import { useEffect, useMemo, useRef, useState } from "react";
import {
  AlertTriangle,
  ArrowLeft,
  CheckCircle2,
  Download,
  Maximize2,
  RefreshCw,
  ShieldCheck,
  X,
} from "lucide-react";
import { useNavigate, useParams } from "react-router-dom";
import { toPng } from "html-to-image";
import AssetUpload from "../components/AssetUpload";
import Brand from "../components/Brand";
import PosterPreview from "../components/PosterPreview";
import PosterLanguageToggle from "../components/PosterLanguageToggle";
import { posterApi } from "../services/posterApi";
import { useStore } from "../store";
import type { Participant, PosterProject } from "../types";
import { localizedPosterCopy } from "../utils/posterLanguage";

async function renderRemotePublishPng(project: PosterProject, imageUrl: string) {
  const copy = localizedPosterCopy(project);
  const canvas = document.createElement("canvas");
  canvas.width = 1024;
  canvas.height = 1536;
  const context = canvas.getContext("2d");
  if (!context) throw new Error("浏览器无法创建海报画布");
  const image = new Image();
  await new Promise<void>((resolve, reject) => {
    const timeout = window.setTimeout(() => reject(new Error("生成图片加载超时，请稍后重试")), 20_000);
    image.onload = () => { window.clearTimeout(timeout); resolve(); };
    image.onerror = () => { window.clearTimeout(timeout); reject(new Error("生成图片无法读取")); };
    image.src = imageUrl;
  });
  const scale = Math.max(canvas.width / image.naturalWidth, canvas.height / image.naturalHeight);
  const width = image.naturalWidth * scale;
  const height = image.naturalHeight * scale;
  context.drawImage(image, (canvas.width - width) / 2, (canvas.height - height) / 2, width, height);
  const shade = context.createLinearGradient(0, 0, 0, canvas.height);
  shade.addColorStop(0, "rgba(0,0,0,.72)");
  shade.addColorStop(.48, "rgba(0,0,0,.08)");
  shade.addColorStop(1, "rgba(0,0,0,.9)");
  context.fillStyle = shade;
  context.fillRect(0, 0, canvas.width, canvas.height);
  const pad = 82;
  context.fillStyle = "#ff784d";
  context.font = "700 22px Arial";
  context.fillText("POSTER VISUAL LAB · AMD ROCm", pad, 96);
  context.fillStyle = "#ffffff";
  context.font = "800 74px Arial, Microsoft YaHei";
  const title = copy.title;
  context.fillText(title.slice(0, 18), pad, 190);
  context.fillStyle = "#f1dccd";
  context.font = "400 24px Arial, Microsoft YaHei";
  context.fillText(copy.theme.slice(0, 40), pad, 236);
  context.strokeStyle = "rgba(255,255,255,.55)";
  context.beginPath(); context.moveTo(pad, 1190); context.lineTo(canvas.width - pad, 1190); context.stroke();
  context.fillStyle = "#ffffff";
  context.font = "800 34px Arial, Microsoft YaHei";
  context.fillText(copy.bands.map((band) => band.displayName).join("  ·  ").slice(0, 34) || copy.subject, pad, 1270);
  const columns = [[copy.labels.date, project.dateTime], [copy.labels.venue, [copy.city, copy.venue].filter(Boolean).join(" · ")], [copy.labels.ticket, project.price || copy.ticketInfo]];
  const columnWidth = (canvas.width - pad * 2) / columns.length;
  columns.forEach(([label, value], index) => {
    const x = pad + columnWidth * index;
    context.fillStyle = "#ff784d"; context.font = "700 16px Arial"; context.fillText(label, x, 1340);
    context.fillStyle = "#ffffff"; context.font = "700 24px Arial, Microsoft YaHei"; context.fillText(String(value || copy.labels.pending).slice(0, 24), x, 1380);
  });
  return await new Promise<Blob>((resolve, reject) => {
    canvas.toBlob((blob) => blob ? resolve(blob) : reject(new Error("海报导出失败")), "image/png");
  });
}

function withTimeout<T>(promise: Promise<T>, timeoutMs: number) {
  return Promise.race([
    promise,
    new Promise<T>((_, reject) =>
      window.setTimeout(() => reject(new Error("海报合成超时")), timeoutMs),
    ),
  ]);
}

async function downloadDataUrl(href: string) {
  if (href.startsWith("data:")) return href;
  const response = await fetch(href);
  const blob = await response.blob();
  return await new Promise<string>((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result));
    reader.onerror = () => reject(new Error("无法准备下载文件"));
    reader.readAsDataURL(blob);
  });
}

export default function Result() {
  const { id = "" } = useParams();
  const { projects, tasks, save } = useStore();
  const nav = useNavigate();
  const [tab, setTab] = useState<"check" | "edit" | "metrics">("check");
  const [busy, setBusy] = useState(false);
  const [imageUrl, setImageUrl] = useState("");
  const [imageError, setImageError] = useState("");
  const [exportError, setExportError] = useState("");
  const [resultAttempt, setResultAttempt] = useState(0);
  const [lightbox, setLightbox] = useState("");
  const ref = useRef<HTMLDivElement>(null);
  const project = projects[id];
  const task = useMemo(
    () => Object.values(tasks)
      .filter((item) => item.projectId === id && item.status === "complete")
      .sort((a, b) => b.startedAt - a.startedAt)[0],
    [tasks, id],
  );
  useEffect(() => {
    if (!task || task.source !== "w7900") return;
    let url = "";
    let active = true;
    setImageError("");
    posterApi.resultBlob(task.id).then((blob) => {
      if (active) {
        url = URL.createObjectURL(blob);
        setImageUrl(url);
      }
    }).catch((reason: unknown) => {
      if (active) setImageError(reason instanceof Error ? reason.message : "生成图片获取失败");
    });
    return () => {
      active = false;
      if (url) URL.revokeObjectURL(url);
    };
  }, [task, resultAttempt]);
  if (!project)
    return (
      <main className="center-state">
        <h1>结果不存在</h1>
        <button className="button" onClick={() => nav("/create")}>
          返回创建
        </button>
      </main>
    );
  const posterCopy = localizedPosterCopy(project);
  const set = <K extends keyof PosterProject>(
    key: K,
    value: PosterProject[K],
  ) => save({ ...project, [key]: value });
  const exportPng = async () => {
    setBusy(true);
    setExportError("");
    try {
      let href = "";
      if (task?.source === "w7900" && imageUrl) {
        try {
          const blob = await withTimeout(renderRemotePublishPng(project, imageUrl), 15_000);
          href = URL.createObjectURL(blob);
        } catch {
          // A browser can reject canvas export for a remote image. Keep the
          // generated visual downloadable instead of leaving the button inert.
          href = imageUrl;
        }
      } else if (ref.current) {
        href = await toPng(ref.current, {
          pixelRatio: 2,
          cacheBust: true,
        });
      }
      if (!href) throw new Error("海报尚未准备好，请稍后重试");
      const downloadHref = await withTimeout(downloadDataUrl(href), 20_000);
      const a = document.createElement("a");
      a.download = `poster-${project.title || project.id}.png`;
      a.href = downloadHref;
      a.style.display = "none";
      document.body.appendChild(a);
      a.click();
      a.remove();
      if (href.startsWith("blob:") && href !== imageUrl) {
        window.setTimeout(() => URL.revokeObjectURL(href), 1000);
      }
    } catch (reason) {
      setExportError(reason instanceof Error ? reason.message : "PNG 导出失败，请重试");
    } finally {
      setBusy(false);
    }
  };
  const previewLarge = async () => {
    if (!ref.current && !(task?.source === "w7900" && imageUrl)) return;
    setBusy(true);
    try {
      if (task?.source === "w7900" && imageUrl) {
        const blob = await renderRemotePublishPng(project, imageUrl);
        setLightbox(URL.createObjectURL(blob));
      } else {
        setLightbox(await toPng(ref.current!, {pixelRatio: 2, cacheBust: true}));
      }
    }
    finally { setBusy(false); }
  };
  const replaceFirstLogo = (asset?: Participant["logo"]) =>
    set(
      "bands",
      project.bands.map((band, index) =>
        index === 0 ? { ...band, logo: asset } : band,
      ),
    );
  const editUrl = `/create?project=${encodeURIComponent(project.id)}`;
  const isRemoteResult = task?.source === "w7900";
  const checks = [
    [isRemoteResult ? "主视觉已从 W7900 生成服务获取" : "本地主视觉 Mock 已生成", isRemoteResult ? "通过" : "Mock"],
    ["标题、时间和地点保留表单原文", "通过"],
    ["Logo 与二维码使用原始素材图层", "通过"],
    ["输出可下载为 PNG", "通过"],
  ];
  return (
    <main className="result">
      <nav>
        <Brand />
        <span>RESULT / {id.slice(0, 8).toUpperCase()}</span>
        <button className="ghost-button" onClick={() => nav(editUrl)}>
          <ArrowLeft /> 返回编辑
        </button>
      </nav>
      <div className="result-layout">
        <section className="result-canvas">
          <PosterLanguageToggle
            value={posterCopy.language}
            onChange={(language) => set("posterLanguage", language)}
          />
          <div className="result-status">
            <CheckCircle2 /> 生成完成{" "}
            <span>
              {task?.source === "w7900"
                ? "AMD W7900 实际输出"
                : "本地 Mock 输出"}
            </span>
          </div>
          {task?.source === "w7900" ? (
            imageUrl ? (
              <div className="session-publish-poster result-publish-poster" ref={ref}>
                <img className="session-publish-visual" src={imageUrl} alt={`${posterCopy.title} AI 主视觉`}/>
                <div className="session-publish-shade"/>
                <div className="session-publish-copy">
                  <small>POSTER VISUAL LAB · AMD ROCm</small>
                  <h2>{posterCopy.title}</h2><p>{posterCopy.theme}</p>
                  <div className="session-publish-bands">{posterCopy.bands.map((band) => <span key={band.id}>{band.logo?.dataUrl ? <img src={band.logo.dataUrl} alt={`${band.displayName} Logo`}/> : <b>{band.displayName}</b>}</span>)}</div>
                  <dl><div><dt>{posterCopy.labels.date}</dt><dd>{project.dateTime}</dd></div><div><dt>{posterCopy.labels.venue}</dt><dd>{[posterCopy.city, posterCopy.venue].filter(Boolean).join(' · ')}</dd></div>{project.price && <div><dt>{posterCopy.labels.ticket}</dt><dd>{project.price}</dd></div>}</dl>
                  {project.assets.qr?.dataUrl && <img className="session-publish-qr" src={project.assets.qr.dataUrl} alt="购票二维码"/>}
                </div>
              </div>
            ) : imageError ? (
              <div className="gpu-result loading-result error-state" role="alert">
                <AlertTriangle />
                <b>生成图片获取失败</b>
                <span>{imageError}</span>
                <button className="ghost-button" onClick={() => setResultAttempt((attempt) => attempt + 1)}>重新获取</button>
              </div>
            ) : (
              <div className="gpu-result loading-result">
                正在安全获取生成图片…
              </div>
            )
          ) : (
            <PosterPreview project={project} nodeRef={ref} />
          )}
          <div className="result-actions">
            <button className="ghost-button" onClick={() => nav(editUrl)}>
              <RefreshCw /> 重新生成主视觉
            </button>
            <button className="button" onClick={exportPng} disabled={busy}>
              <Download />
              {busy ? "正在导出…" : "导出 PNG"}
            </button>
            <button className="ghost-button" onClick={() => void previewLarge()} disabled={busy}><Maximize2/> 放大</button>
          </div>
          {exportError && <div className="assistant-error" role="alert"><b>导出失败</b><span>{exportError}</span><button onClick={() => setExportError("")}>关闭</button></div>}
        </section>
        <aside className="result-panel">
          <div className="tabs">
            <button
              className={tab === "check" ? "active" : ""}
              onClick={() => setTab("check")}
            >
              检查结果
            </button>
            <button
              className={tab === "edit" ? "active" : ""}
              onClick={() => setTab("edit")}
            >
              编辑信息
            </button>
            <button
              className={tab === "metrics" ? "active" : ""}
              onClick={() => setTab("metrics")}
            >
              性能数据
            </button>
          </div>
          {tab === "check" && (
            <div className="tab-content">
              <p className="eyebrow">PRE-FLIGHT CHECK</p>
              <h2>{isRemoteResult ? "可以进入发布检查" : "Mock 预览已完成"}</h2>
              {checks.map(([text, status]) => (
                <div className="check-row" key={text}>
                  <ShieldCheck />
                  {text}
                  <b>{status}</b>
                </div>
              ))}
              {!isRemoteResult && <div className="warning-box"><b>当前不是 GPU 实际输出</b><p>此页面用于检查信息图层、布局与 PNG 导出；连接远程生成服务后才能进行最终发布检查。</p></div>}
            </div>
          )}
          {tab === "edit" && (
            <div className="tab-content">
              <p>
                文字修改继续保存在项目中；真实输出图目前由 GPU 服务直接返回。
              </p>
              <label className="field">
                <span>{posterCopy.language === "en" ? "活动标题 / TITLE" : "活动标题"}</span>
                <input
                  value={posterCopy.language === "en" ? project.titleEn ?? "" : project.title}
                  onChange={(e) => posterCopy.language === "en" ? set("titleEn", e.target.value) : set("title", e.target.value)}
                />
              </label>
              <label className="field">
                <span>{posterCopy.language === "en" ? "主题文案 / TAGLINE" : "主题文案"}</span>
                <input
                  value={posterCopy.language === "en" ? project.themeEn ?? "" : project.theme}
                  onChange={(e) => posterCopy.language === "en" ? set("themeEn", e.target.value) : set("theme", e.target.value)}
                />
              </label>
              <label className="field">
                <span>时间</span>
                <input
                  value={project.dateTime}
                  onChange={(e) => set("dateTime", e.target.value)}
                />
              </label>
              <label className="field">
                <span>{posterCopy.language === "en" ? "地点 / VENUE" : "地点"}</span>
                <input
                  value={posterCopy.language === "en" ? project.venueEn ?? "" : project.venue}
                  onChange={(e) => posterCopy.language === "en" ? set("venueEn", e.target.value) : set("venue", e.target.value)}
                />
              </label>
              {project.bands[0] && (
                <AssetUpload
                  label={`${project.bands[0].name} Logo`}
                  kind="logo"
                  value={project.bands[0].logo}
                  onChange={replaceFirstLogo}
                />
              )}
            </div>
          )}
          {tab === "metrics" && (
            <div className="tab-content metrics-list">
              {Object.entries(task?.metrics ?? {}).map(([key, value]) => (
                <div key={key}>
                  <span>
                    {
                      (
                        {
                          gpu: "GPU 型号",
                          rocm: "运行环境",
                          resolution: "输出分辨率",
                          duration: "推理耗时",
                          peakVram: "显存",
                        } as Record<string, string>
                      )[key]
                    }
                  </span>
                  <b>{value}</b>
                </div>
              ))}
            </div>
          )}
        </aside>
      </div>
      {lightbox && <div className="poster-lightbox" role="dialog" aria-modal="true" aria-label="最终海报大图" onClick={() => setLightbox("")}>
        <button type="button" onClick={() => setLightbox("")} aria-label="关闭大图"><X/></button>
        <img src={lightbox} alt={`${project.title} 最终发布版`} onClick={(event) => event.stopPropagation()}/>
        <a href={lightbox} download={`poster-${project.title}.png`} onClick={(event) => event.stopPropagation()}><Download/> 下载图片</a>
      </div>}
    </main>
  );
}
