import {useEffect, useMemo, useRef, useState} from 'react';
import {Bot, Check, CornerDownLeft, Download, LoaderCircle, Maximize2, RefreshCw, Send, Sparkles, UserRound, X} from 'lucide-react';
import {toPng} from 'html-to-image';
import type {Participant, PosterProject, UploadedAsset} from '../types';
import {
  absoluteAIImageUrl,
  aiSessionApi,
  type AISession,
  type AISessionBrief,
  type BackendDependencies,
  type BackendHealth,
} from '../services/aiSessionApi';
import AssetUpload from './AssetUpload';
import PosterLanguageToggle from './PosterLanguageToggle';
import {localizedPosterCopy} from '../utils/posterLanguage';

export type ProjectDraft = Partial<Omit<PosterProject, 'bands'>> & {bands?: Participant[]};

const fieldLabels: Record<string, string> = {
  'event.title': '活动标题', 'event.artist': '艺人 / 乐队', 'event.date': '日期',
  'event.time': '时间', 'event.venue': '场地', 'visual.style': '视觉风格',
  'visual.theme': '视觉主题', 'visual.musicGenre': '音乐类型', 'visual.mood': '情绪关键词',
};

async function renderPublishPng(node: HTMLElement): Promise<string> {
  const width = Math.max(node.getBoundingClientRect().width, 1);
  const source = await toPng(node, {
    pixelRatio: Math.max(2, 1024 / width),
    cacheBust: true,
  });

  return new Promise((resolve, reject) => {
    const image = new Image();
    image.onload = () => {
      const canvas = document.createElement('canvas');
      canvas.width = 1024;
      canvas.height = 1536;
      const context = canvas.getContext('2d');
      if (!context) {
        reject(new Error('浏览器无法创建 PNG 画布'));
        return;
      }
      context.imageSmoothingEnabled = true;
      context.imageSmoothingQuality = 'high';
      context.drawImage(image, 0, 0, canvas.width, canvas.height);
      resolve(canvas.toDataURL('image/png'));
    };
    image.onerror = () => reject(new Error('发布版海报栅格化失败'));
    image.src = source;
  });
}

function briefToDraft(session: AISession, project: PosterProject): ProjectDraft {
  const {event, visual} = session.brief;
  const names = (event.artist ?? '').split(/\s*(?:&|×|x|、|,|，| and )\s*/i).filter(Boolean);
  return {
    scene: project.scene ?? 'concert',
    title: event.title || project.title,
    theme: visual.theme || project.theme,
    subject: event.artist || project.subject,
    dateTime: [event.date, event.time].filter(Boolean).join(' '),
    venue: event.venue || project.venue,
    price: event.presalePrice || event.doorPrice || '',
    ticketInfo: '',
    bands: names.length ? names.map((name) => project.bands.find((band) => band.name === name) ?? {
      id: crypto.randomUUID(), name, genre: visual.musicGenre ?? '',
    }) : project.bands,
  };
}

/**
 * The deployed agent can return a partial brief while it is still collecting
 * details. Keep the local form authoritative for values the user already
 * entered so “同步到表单” never drops them when the model parser misses a
 * Chinese label or an optional field.
 */
function mergeLocalBrief(session: AISession, project: PosterProject): AISession {
  const event = session.brief.event;
  const visual = session.brief.visual;
  const artist = event.artist || project.bands.map((band) => band.name).filter(Boolean).join(' & ') || project.subject;
  const mergedBrief = {
    ...session.brief,
    event: {
      ...event,
      title: event.title || project.title,
      artist,
      date: event.date || project.dateTime.split(/\s+/)[0] || '',
      time: event.time || project.dateTime.split(/\s+/)[1] || '',
      venue: event.venue || [project.city, project.venue].filter(Boolean).join(' · '),
    },
    visual: {
      ...visual,
      style: visual.style || project.styleId,
      theme: visual.theme || project.theme,
      musicGenre: visual.musicGenre || project.bands.map((band) => band.genre).filter(Boolean).join(' / '),
      mood: visual.mood?.length ? visual.mood : (project.theme ? [project.theme] : undefined),
    },
  };
  const values: Record<string, string> = {
    'event.title': mergedBrief.event.title,
    'event.artist': mergedBrief.event.artist ?? '',
    'event.date': mergedBrief.event.date,
    'event.time': mergedBrief.event.time,
    'event.venue': mergedBrief.event.venue,
    'visual.style': mergedBrief.visual.style,
    'visual.theme': mergedBrief.visual.theme,
    'visual.musicGenre': mergedBrief.visual.musicGenre ?? '',
    'visual.mood': mergedBrief.visual.mood?.join(' / ') ?? '',
  };
  const missingFields = (session.missingFields ?? []).filter((field) => !values[field]);
  return {...session, brief: mergedBrief, missingFields: missingFields.length ? missingFields : null};
}

function projectToSessionBrief(project: PosterProject): AISessionBrief {
  const [date = '', time = ''] = project.dateTime.trim().split(/\s+/, 2);
  return {
    event: {
      title: project.title,
      artist: project.bands.map((band) => band.name).filter(Boolean).join(' & ') || project.subject,
      date,
      time,
      venue: [project.city, project.venue].filter(Boolean).join(' · '),
      presalePrice: project.price,
    },
    branding: {
      artistLogoAssetId: undefined,
    },
    visual: {
      style: project.styleId,
      theme: project.theme,
      musicGenre: project.bands.map((band) => band.genre).filter(Boolean).join(' / '),
      mood: project.theme ? [project.theme] : undefined,
    },
  };
}

type AssetBinding = {asset: UploadedAsset; kind: 'person' | 'logo' | 'reference'; purpose: string};

function projectAssetBindings(project: PosterProject): AssetBinding[] {
  const bindings: AssetBinding[] = [];
  project.bands.forEach((band) => {
    if (band.groupPhoto) bindings.push({asset: band.groupPhoto, kind: 'person', purpose: 'performer'});
    if (band.keyPhoto) bindings.push({asset: band.keyPhoto, kind: 'person', purpose: 'performer'});
    if (band.logo) bindings.push({asset: band.logo, kind: 'logo', purpose: 'artist_logo'});
  });
  if (project.assets.speaker) bindings.push({asset: project.assets.speaker, kind: 'person', purpose: 'performer'});
  if (project.assets.organizerLogo) bindings.push({asset: project.assets.organizerLogo, kind: 'logo', purpose: 'event_logo'});
  if (project.assets.reference) bindings.push({asset: project.assets.reference, kind: 'reference', purpose: 'reference'});
  if (project.assets.venue) bindings.push({asset: project.assets.venue, kind: 'reference', purpose: 'reference'});
  return bindings;
}

async function dataUrlBlob(asset: UploadedAsset): Promise<Blob> {
  const response = await fetch(asset.dataUrl);
  const source = await response.blob();
  // The deployed asset endpoint accepts PNG/JPEG. Rasterize SVG/WebP/GIF
  // client-side while keeping the original local asset for the final poster.
  if (source.type === 'image/png' || source.type === 'image/jpeg') return source;
  const image = new Image();
  image.src = asset.dataUrl;
  await image.decode();
  const canvas = document.createElement('canvas');
  canvas.width = Math.max(1, image.naturalWidth || 1024);
  canvas.height = Math.max(1, image.naturalHeight || 1024);
  const context = canvas.getContext('2d');
  if (!context) throw new Error('无法准备素材上传');
  context.drawImage(image, 0, 0, canvas.width, canvas.height);
  return new Promise<Blob>((resolve, reject) => {
    canvas.toBlob((blob) => blob ? resolve(blob) : reject(new Error('素材转码失败')), 'image/png');
  });
}

export default function ProjectAssistant({project, onApply}: {
  project: PosterProject;
  onApply: (draft: ProjectDraft) => void;
}) {
  const storageKey = `poster-ai-session:${project.id}`;
  const [session, setSession] = useState<AISession | null>(null);
  const [input, setInput] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [imageErrors, setImageErrors] = useState<Record<string, boolean>>({});
  const [lightbox, setLightbox] = useState<{src: string; alt: string} | null>(null);
  const [boundAssetCount, setBoundAssetCount] = useState(0);
  const [uploadProgress, setUploadProgress] = useState('');
  const [pendingMessage, setPendingMessage] = useState('');
  const [backendHealth, setBackendHealth] = useState<BackendHealth | null>(null);
  const [backendDependencies, setBackendDependencies] = useState<BackendDependencies | null>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const messagesRef = useRef<HTMLDivElement>(null);
  const publishRef = useRef<HTMLDivElement>(null);
  const projectRef = useRef(project);
  projectRef.current = project;
  const actions = session?.availableActions ?? [];
  const canMessage = !session || actions.includes('send_message');
  const canAttach = !!session && !['succeeded', 'completed_with_warnings', 'failed', 'canceled', 'cancelled'].includes(session.status);
  const candidates = session?.poster?.candidates ?? [];
  const shouldPoll = !!session && (
    actions.includes('refresh') ||
    session.status === 'generating_candidates' ||
    session.status === 'looping'
  );

  useEffect(() => {
    aiSessionApi.health().then(setBackendHealth).catch(() => setBackendHealth(null));
    aiSessionApi.dependencies().then(setBackendDependencies).catch(() => setBackendDependencies(null));
  }, []);

  useEffect(() => {
    const id = localStorage.getItem(storageKey);
    if (!id) return;
    aiSessionApi.get(id)
      .then((next) => setSession(mergeLocalBrief(next, projectRef.current)))
      .catch(() => localStorage.removeItem(storageKey));
  }, [storageKey]);

  useEffect(() => {
    if (!shouldPoll || !session) return;
    const timer = window.setInterval(() => {
      aiSessionApi.get(session.sessionId).then(setSession).catch((reason: unknown) => {
        setError(reason instanceof Error ? reason.message : '刷新 AI Session 失败');
      });
    }, 2500);
    return () => window.clearInterval(timer);
  }, [session, shouldPoll]);

  useEffect(() => {
    messagesRef.current?.scrollTo({top: messagesRef.current.scrollHeight, behavior: 'smooth'});
  }, [session?.messages.length, pendingMessage, busy]);

  const readiness = useMemo(() => {
    const missing = session ? (session.missingFields?.length ?? 0) : 9;
    return Math.max(0, Math.round((9 - missing) / 9 * 100));
  }, [session]);

  const run = async (operation: () => Promise<AISession>) => {
    setBusy(true); setError('');
    try {
      const next = await operation();
      setSession(next);
      localStorage.setItem(storageKey, next.sessionId);
      return next;
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'AI 服务暂时不可用');
      return null;
    } finally { setBusy(false); }
  };

  const send = async (preset?: string) => {
    const content = (preset ?? input).trim();
    if (!content || busy) return;
    setInput(''); setPendingMessage(content); setBusy(true); setError('');
    try {
      let current = session;
      if (!current) {
        current = await aiSessionApi.create(projectToSessionBrief(project));
        localStorage.setItem(storageKey, current.sessionId);
        setSession(current);
      }
      const next = await aiSessionApi.sendMessage(current.sessionId, content);
      setSession(mergeLocalBrief(next, project));
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '发送消息失败');
      setInput(content);
    } finally { setBusy(false); setPendingMessage(''); }
  };

  const applyBrief = () => {
    if (!session) return;
    onApply(briefToDraft(session, project));
    inputRef.current?.focus();
  };

  const reset = () => {
    localStorage.removeItem(storageKey);
    setSession(null); setError(''); setInput(''); setImageErrors({}); setBoundAssetCount(0);
  };

  const finalUrl = absoluteAIImageUrl(session?.poster?.resultUrl);
  const posterCopy = localizedPosterCopy(project);
  const selectedCandidate = candidates.find((candidate) => candidate.selected)
    ?? candidates.find((candidate) => candidate.candidateId === session?.poster?.selectedCandidateId);
  const selectedVisualUrl = absoluteAIImageUrl(selectedCandidate?.imageUrl);
  const assetBindings = projectAssetBindings(project);
  const usageEvidence = session?.assetUsages?.length
    ? session.assetUsages
    : (session?.assets ?? []).map((asset) => ({
      assetId: asset.assetId,
      purpose: asset.purpose,
      stage: asset.usedInStage?.join(' / ') || (asset.actuallyUsed ? '生成流程' : undefined),
      used: asset.actuallyUsed,
      status: asset.processStatus,
      message: asset.usageNote || asset.processing?.error,
    }));
  const generationHasOutput = candidates.some((candidate) => candidate.status === 'ready')
    || Boolean(session?.poster?.resultUrl);
  const boundEvidenceCount = session?.assets?.length ?? boundAssetCount;
  const hasUnusedBoundAssets = generationHasOutput
    && boundEvidenceCount > 0
    && !usageEvidence.some((item) => item.used);
  const publishFactsConfirmed = Boolean(
    project.title.trim()
    && project.dateTime.trim()
    && project.venue.trim()
    && (project.subject.trim() || project.bands.some((band) => band.name.trim())),
  );

  const setConversationAsset = (kind: 'person' | 'logo' | 'reference', asset?: UploadedAsset) => {
    if (kind === 'reference') {
      onApply({assets: {...project.assets, reference: asset}});
      return;
    }
    const currentBands = project.bands.length ? project.bands : [{
      id: crypto.randomUUID(),
      name: session?.brief.event.artist || '表演者',
      genre: session?.brief.visual.musicGenre || '',
    }];
    onApply({bands: currentBands.map((band, index) => index === 0
      ? {...band, ...(kind === 'person' ? {groupPhoto: asset} : {logo: asset})}
      : band)});
  };

  const previewPublishedPoster = async () => {
    if (!publishRef.current) return;
    setBusy(true); setError('');
    try {
      const src = await renderPublishPng(publishRef.current);
      setLightbox({src, alt: '最终发布版海报'});
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '海报预览生成失败');
    } finally { setBusy(false); }
  };

  const downloadPublishedPoster = async () => {
    if (!publishRef.current) return;
    setBusy(true); setError('');
    try {
      const dataUrl = await renderPublishPng(publishRef.current);
      const anchor = document.createElement('a');
      anchor.download = `${project.title || session?.brief.event.title || 'poster'}-publish.png`;
      anchor.href = dataUrl;
      anchor.style.display = 'none';
      document.body.appendChild(anchor);
      anchor.click();
      anchor.remove();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '发布版导出失败');
    } finally { setBusy(false); }
  };

  const uploadAndBindAssets = async () => {
    if (!session || !assetBindings.length) return;
    setBusy(true); setError('');
    setUploadProgress(`准备上传 ${assetBindings.length} 项素材…`);
    try {
      const cacheKey = `poster-remote-assets:${project.id}`;
      const cache = JSON.parse(localStorage.getItem(cacheKey) ?? '{}') as Record<string, string>;
      const remoteBindings: {assetId: string; purpose: string}[] = [];
      const failures: string[] = [];
      for (const binding of assetBindings) {
        setUploadProgress(`正在处理 ${binding.asset.name}…`);
        try {
          let assetId = cache[binding.asset.id];
          if (!assetId) {
            const uploaded = await aiSessionApi.uploadAsset(
              await dataUrlBlob(binding.asset), binding.asset.name.replace(/\.[^.]+$/, '.png'), binding.kind,
            );
            assetId = uploaded.assetId;
            cache[binding.asset.id] = assetId;
          }
          remoteBindings.push({assetId, purpose: binding.purpose});
        } catch (reason) {
          failures.push(`${binding.asset.name}：${reason instanceof Error ? reason.message : '上传失败'}`);
        }
      }
      localStorage.setItem(cacheKey, JSON.stringify(cache));
      if (remoteBindings.length) {
        const next = await aiSessionApi.bindAssets(session.sessionId, remoteBindings);
        setSession(next);
        setBoundAssetCount(remoteBindings.length);
      }
      if (failures.length) setError(`部分素材未绑定：${failures.join('；')}`);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '素材上传失败');
    } finally { setBusy(false); setUploadProgress(''); }
  };

  return <aside className="assistant-panel assistant-live" id="project-assistant">
    <header className="assistant-head">
      <div><span><Bot size={16}/> AI CREATIVE AGENT</span><b>与 AI 一起完成海报</b></div>
      <em>LIVE API</em>
    </header>
    <div className="assistant-context">
      <Sparkles size={14}/><span>{session ? session.status : '等待开始'}</span>
      <b>{readiness}% READY</b>
    </div>
    {session && <div className="assistant-telemetry" aria-label="生成运行状态">
      <div><small>GPU</small><b>{session.metrics?.gpu ?? backendHealth?.gpu?.model ?? 'AMD GPU 节点'}</b></div>
      <div><small>ROCm</small><b>{session.metrics?.rocm ?? '后端未透出'}</b></div>
      <div><small>WORKFLOW</small><b>{backendHealth?.comfyui?.workflowVersion ?? '实时读取中'}</b></div>
      <div><small>阶段</small><b>{session.generationStages?.find((stage) => stage.status === 'running')?.label ?? session.status}</b></div>
      <div><small>进度</small><b>{session.poster ? `${session.poster.progress.completed}/${session.poster.progress.total}` : '等待任务'}</b></div>
    </div>}
    {backendDependencies?.capabilities && <details className="assistant-capabilities">
      <summary>
        <span>生成能力检查</span>
        <small>{backendDependencies.status === 'healthy' ? '后端在线' : backendDependencies.status}</small>
      </summary>
      <div>
        {([
          ['负向提示词', backendDependencies.capabilities.negativePrompt],
          ['人物 / Logo 抠图', backendDependencies.capabilities.backgroundRemoval],
          ['人物相似度', backendDependencies.capabilities.personSimilarityMetric],
          ['参考图条件化', backendDependencies.capabilities.referenceImageConditioning],
        ] as const).map(([label, capability]) => capability && <p key={label}>
          <b className={capability.available ? 'ok' : 'muted'}>{capability.available ? '可用' : '未接入'}</b>
          <span>{label}</span>
          {!capability.available && capability.reason && <small title={capability.reason}>{capability.reason}</small>}
        </p>)}
      </div>
    </details>}

    <div className="assistant-messages" ref={messagesRef} aria-live="polite">
      {!session?.messages.length && <div className="assistant-message assistant"><i><Bot/></i><p>告诉我演出、活动或品牌故事。我会逐步追问，并由后端 AI 生成可确认的设计方案。</p></div>}
      {session?.messages.map((message) => <div key={message.messageId} className={`assistant-message ${message.role}`}>
        <i>{message.role === 'user' ? <UserRound/> : <Bot/>}</i><p>{message.content}</p>
      </div>)}
      {pendingMessage && !session?.messages.some((message) => message.role === 'user' && message.content === pendingMessage) && <div className="assistant-message user pending"><i><UserRound/></i><p>{pendingMessage}<small>正在送达 AI…</small></p></div>}
      {busy && <div className="assistant-message assistant"><i><Bot/></i><p className="assistant-typing"><span/><span/><span/></p></div>}
    </div>

    {error && <div className="assistant-error"><b>连接或生成失败</b><span>{error}</span><button onClick={() => setError('')}>关闭</button></div>}

    {!!session?.missingFields?.length && <section className="assistant-missing">
      <header><b>还需要补充</b><span>{session.missingFields.length} 项</span></header>
      <div>{session.missingFields.map((field) => <span key={field}>{fieldLabels[field] ?? field}</span>)}</div>
    </section>}

    {!!usageEvidence.length && <section className="assistant-asset-status">
      <header><b>素材处理与使用证据</b><span>{usageEvidence.filter((item) => item.used).length} 项已使用</span></header>
      {usageEvidence.map((item) => <div key={`${item.assetId}-${item.purpose}`} title={item.message}><span>{item.purpose}</span><b className={item.used ? 'ok' : 'muted'}>{item.used ? `已用于 ${item.stage ?? '生成'}` : item.message ?? (generationHasOutput ? '已绑定 · 条件化未启用' : '已绑定 · 等待生成')}</b></div>)}
    </section>}
    {hasUnusedBoundAssets && <div className="assistant-error" role="alert">
      <b>真实素材尚未参与本次生成</b>
      <span>后端已接收素材，但没有返回任何实际使用证据。当前候选图不能视为人物保持或风格参考已生效。</span>
    </div>}

    {session && <button className="assistant-sync" onClick={applyBrief}><Check/> 将 AI 已理解的信息同步到表单</button>}
    {canAttach && assetBindings.length > 0 && <button className="assistant-assets" disabled={busy} onClick={() => void uploadAndBindAssets()}>
      {boundAssetCount ? <Check/> : <Sparkles/>} {boundAssetCount ? `已绑定 ${boundAssetCount} 项真实素材` : `上传并绑定 ${assetBindings.length} 项真实素材`}
    </button>}
    {uploadProgress && <div className="assistant-upload-progress" role="status"><LoaderCircle/> {uploadProgress}<small>后端正在校验和预处理，可能需要几十秒</small></div>}

    {actions.includes('confirm_plan') && !!session?.plans?.length && <section className="assistant-plans">
      <header><b>选择一个设计方向</b><span>{session.plans.length} 个真实方案</span></header>
      {!publishFactsConfirmed && <p className="assistant-plan-gate" role="status">先点击“将 AI 已理解的信息同步到表单”，确认标题、时间和地点后再生成。</p>}
      {session.plans.map(({planId, plan}) => <article key={planId}>
        <div className="plan-palette">{plan.palette.map((color) => <i key={color} style={{background: color}}/>)}</div>
        <b>{plan.name}</b><p>{plan.concept}</p><small>{plan.composerTemplate} · {plan.composition.symmetry}</small>
        <button disabled={busy || !publishFactsConfirmed} onClick={() => run(() => aiSessionApi.confirmPlan(session.sessionId, planId))}>{publishFactsConfirmed ? '确认此方案' : '请先确认活动信息'}</button>
      </article>)}
    </section>}

    {session?.poster && <section className="assistant-generation">
      <header><b>候选视觉</b><span>{session.poster.progress.completed} / {session.poster.progress.total}</span></header>
      {shouldPoll && <div className="generation-live"><LoaderCircle/> AMD GPU 正在生成候选图，页面会自动刷新</div>}
      <div className="candidate-grid">{candidates.map((candidate) => {
        const imageUrl = absoluteAIImageUrl(candidate.imageUrl);
        return <article key={candidate.candidateId} className={candidate.selected ? 'selected' : ''}>
          {imageUrl && !imageErrors[candidate.candidateId]
            ? <button className="candidate-image-button" type="button" onClick={() => setLightbox({src: imageUrl, alt: candidate.variantName})} aria-label={`放大查看 ${candidate.variantName}`}><img src={imageUrl} alt={candidate.variantName} onError={() => setImageErrors((current) => ({...current, [candidate.candidateId]: true}))}/><Maximize2/></button>
            : <div className="candidate-placeholder"><LoaderCircle/><span>{candidate.status}</span></div>}
          <b>{candidate.variantName}</b><small>AI 主视觉 · Attempt {candidate.attempt} · {candidate.status}</small>
          {actions.includes('select_candidate') && candidate.status === 'ready' && <button disabled={busy} onClick={() => run(() => aiSessionApi.selectCandidate(session.sessionId, candidate.candidateId))}>选择这张</button>}
          {candidate.status === 'failed' && session.poster && <button disabled={busy} onClick={() => run(() => aiSessionApi.retryCandidate(session.sessionId, session.poster!.posterId, candidate.candidateId))}>仅重试这张</button>}
        </article>;
      })}</div>
      {!!session.generationStages?.length && <div className="assistant-stage-trace">{session.generationStages.map((stage) => <div key={stage.id ?? stage.key ?? stage.label} className={stage.status}><span>{stage.label}</span><b>{stage.progress ?? (stage.status === 'completed' ? 100 : 0)}%</b>{stage.etaSeconds != null && <small>约 {stage.etaSeconds}s</small>}</div>)}</div>}
    </section>}

    {actions.includes('finalize') && session && <button className="button assistant-finalize" disabled={busy} onClick={() => run(() => aiSessionApi.finalize(session.sessionId))}><RefreshCw/> 启动 AI 视觉审查与优化</button>}
    {session && actions.includes('cancel') && <button className="ghost-button assistant-cancel" disabled={busy} onClick={() => run(() => aiSessionApi.cancel(session.sessionId))}>取消当前生成</button>}
    {session?.reviewSummary?.warning && actions.includes('finalize') && <button className="ghost-button assistant-retry" disabled={busy} onClick={() => run(() => aiSessionApi.retryFinalize(session.sessionId))}><RefreshCw/> 安全重试审查</button>}
    {finalUrl && <section className="assistant-final">
      <header><b>发布版海报</b><span>真实信息程序化叠加</span></header>
      <PosterLanguageToggle value={posterCopy.language} onChange={(language) => onApply({posterLanguage: language})}/>
      {selectedVisualUrl && <div className="session-publish-poster" ref={publishRef}>
        <img className="session-publish-visual" src={selectedVisualUrl} alt="选中的 AI 主视觉"/>
        <div className="session-publish-shade"/>
        <div className="session-publish-copy">
          <small>POSTER VISUAL LAB · AI KEY VISUAL</small>
          <h2>{posterCopy.title || session?.brief.event.title}</h2>
          <p>{posterCopy.theme || session?.brief.visual.theme}</p>
          <div className="session-publish-bands">{posterCopy.bands.map((band) => <span key={band.id}>{band.logo?.dataUrl ? <img src={band.logo.dataUrl} alt={`${band.displayName} Logo`}/> : <b>{band.displayName}</b>}</span>)}</div>
          <dl><div><dt>{posterCopy.labels.date}</dt><dd>{project.dateTime || `${session?.brief.event.date ?? ''} ${session?.brief.event.time ?? ''}`}</dd></div><div><dt>{posterCopy.labels.venue}</dt><dd>{[posterCopy.city, posterCopy.venue].filter(Boolean).join(' · ') || session?.brief.event.venue}</dd></div>{project.price && <div><dt>{posterCopy.labels.ticket}</dt><dd>{project.price}</dd></div>}</dl>
          {project.assets.qr?.dataUrl && <img className="session-publish-qr" src={project.assets.qr.dataUrl} alt="购票二维码"/>}
        </div>
      </div>}
      {selectedVisualUrl && <button className="button assistant-publish-download" disabled={busy} onClick={() => void downloadPublishedPoster()}><Download/> 导出精确信息发布版</button>}
      {selectedVisualUrl && <button className="ghost-button assistant-publish-preview" disabled={busy} onClick={() => void previewPublishedPoster()}><Maximize2/> 放大查看最终发布版</button>}
      {session?.reviewSummary?.warning && <p className="assistant-review-warning">评分 {session.reviewSummary.bestScore ?? '—'} · {session.reviewSummary.warning}</p>}
      {actions.includes('download_final') && <details><summary>查看后端原始合成结果</summary><img src={finalUrl} alt="后端合成的最终海报"/><a className="button" href={finalUrl} target="_blank" rel="noreferrer"><Download/> 下载后端结果</a></details>}
    </section>}

    {canMessage && <>
      <details className="assistant-upload-studio assistant-attachment-tray">
        <summary><span>＋ 添加附件（可选）</span><small>{assetBindings.length ? `${assetBindings.length} 项素材` : '人物 / Logo / 风格参考'}</small></summary>
        <p>所有附件均非必填。人物和 Logo 保留原始内容，参考图只用于提取视觉语言。</p>
        <div className="assistant-upload-grid">
          <AssetUpload label="人物 / 乐队照片（可选）" kind="person" value={project.bands[0]?.groupPhoto} onChange={(asset) => setConversationAsset('person', asset)}/>
          <AssetUpload label="乐队原始 Logo（可选）" kind="logo" value={project.bands[0]?.logo} onChange={(asset) => setConversationAsset('logo', asset)}/>
          <AssetUpload label="海报风格参考（可选）" kind="reference" value={project.assets.reference} onChange={(asset) => setConversationAsset('reference', asset)}/>
        </div>
        {!session && assetBindings.length > 0 && <small className="assistant-asset-hint">附件已保存在项目中。发送消息后即可绑定给 AI。</small>}
      </details>
      <div className="assistant-prompts">
        <button onClick={() => send('我要做一张地下金属演出海报。标题是长安双雄，艺人是内网穿透 NATP 与示例金属乐队，2026 年 8 月 8 日 20:00，在西安大雁塔附近演出。视觉是黑色、骨白、氧化红的手工复印拼贴，工业金属风，情绪原始、黑暗、仪式感、高能量。')}>地下金属演出</button>
        <button onClick={() => send('我要做一张独立音乐节海报，请逐项问我还缺少什么。')}>逐步问我</button>
      </div>
      <div className="assistant-compose">
        <textarea ref={inputRef} value={input} onChange={(event) => setInput(event.target.value)} onKeyDown={(event) => {if (event.key === 'Enter' && !event.shiftKey) {event.preventDefault(); void send();}}} placeholder="描述活动，或继续回答 AI 的问题……"/>
        <button onClick={() => void send()} disabled={!input.trim() || busy} aria-label="发送消息"><Send/></button>
        <small><CornerDownLeft/> Enter 发送 · Shift + Enter 换行</small>
      </div>
    </>}
    {session && <button className="assistant-reset" onClick={reset}>开始新的 AI 会话</button>}
    {lightbox && <div className="poster-lightbox" role="dialog" aria-modal="true" aria-label={lightbox.alt} onClick={() => setLightbox(null)}>
      <button type="button" onClick={() => setLightbox(null)} aria-label="关闭大图"><X/></button>
      <img src={lightbox.src} alt={lightbox.alt} onClick={(event) => event.stopPropagation()}/>
      <a href={lightbox.src} download={`${lightbox.alt}.png`} onClick={(event) => event.stopPropagation()}><Download/> 下载图片</a>
    </div>}
  </aside>;
}
