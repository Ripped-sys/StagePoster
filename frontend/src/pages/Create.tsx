import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import {
  ArrowLeft,
  ArrowRight,
  Bot,
  Check,
  ChevronRight,
  Plus,
  Sparkles,
  Trash2,
} from "lucide-react";
import Brand from "../components/Brand";
import AssetUpload from "../components/AssetUpload";
import ProjectAssistant from "../components/ProjectAssistant";
import { demoProject, emptyProject, styles } from "../data/mock";
import { useStore } from "../store";
import type { Participant, PosterProject, SceneType } from "../types";
const scenes: [SceneType, string, string][] = [
  ["concert", "音乐演出", "乐队、Livehouse 与巡演"],
  ["festival", "音乐节", "多艺人、多舞台阵容"],
  ["lecture", "讲座", "主讲人、主题与机构"],
  ["competition", "比赛", "赛事、奖项与报名"],
  ["commercial", "商业活动", "品牌、产品与转化"],
  ["custom", "自定义", "保留核心四要素"],
];
const Field = ({
  label,
  value,
  onChange,
  required = false,
  type = "text",
  error,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  required?: boolean;
  type?: string;
  error?: string;
}) => (
  <label className="field">
    <span>
      {label}
      {required && <b> *</b>}
    </span>
    {type === "textarea" ? (
      <textarea value={value} onChange={(e) => onChange(e.target.value)} />
    ) : (
      <input
        data-error={!!error}
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
      />
    )}{" "}
    {error && <small className="field-error">{error}</small>}
  </label>
);
export default function Create() {
  const [params] = useSearchParams();
  const { save, projects } = useStore();
  const [project, setProject] = useState<PosterProject>(() => {
    const projectId = params.get("project");
    if (projectId && projects[projectId]) return projects[projectId];
    return params.get("demo") ? demoProject() : emptyProject();
  });
  const [step, setStep] = useState(params.get("demo") || params.get("project") ? 1 : 0);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [notice, setNotice] = useState("");
  const nav = useNavigate();
  const mainRef = useRef<HTMLDivElement>(null);
  useEffect(() => save(project), [project, save]);
  const set = <K extends keyof PosterProject>(
    key: K,
    value: PosterProject[K],
  ) => setProject((p) => ({ ...p, [key]: value }));
  const completion = useMemo(
    () =>
      Math.round(
        ([
          project.scene,
          project.title,
          project.theme,
          project.dateTime,
          project.venue,
          project.subject,
          project.styleId,
        ].filter(Boolean).length /
          7) *
          100,
      ),
    [project],
  );
  const validate = () => {
    const next: Record<string, string> = {};
    if (!project.scene) next.scene = "请选择一个海报场景";
    if (!project.title.trim()) next.title = "请填写活动名称";
    if (!project.theme.trim()) next.theme = "请填写活动主题";
    if (!project.dateTime.trim()) next.dateTime = "请填写日期和时间";
    if (!project.venue.trim()) next.venue = "请填写场地名称";
    if (!project.subject.trim()) next.subject = "请填写活动主体";
    if (project.scene === "concert" && !project.bands.length)
      next.bands = "至少添加一支乐队";
    setErrors(next);
    if (Object.keys(next).length) {
      setStep(next.scene ? 0 : 1);
      setTimeout(
        () =>
          mainRef.current
            ?.querySelector('[data-error="true"],.error-anchor')
            ?.scrollIntoView({ behavior: "smooth", block: "center" }),
        30,
      );
      return false;
    }
    return true;
  };
  const generate = () => {
    if (!validate()) return;
    save(project);
    nav(`/generate/${project.id}`);
  };
  const addBand = () =>
    set("bands", [
      ...project.bands,
      { id: crypto.randomUUID(), name: "", genre: "" },
    ]);
  const updateBand = (id: string, patch: Partial<Participant>) =>
    set(
      "bands",
      project.bands.map((b) => (b.id === id ? { ...b, ...patch } : b)),
    );
  const removeBand = (id: string) =>
    set(
      "bands",
      project.bands.filter((b) => b.id !== id),
    );
  const recommend = () => {
    const text = (project.theme + " " + project.title).toLowerCase();
    const id =
      text.includes("科技") || text.includes("未来")
        ? "cyber"
        : project.scene === "lecture"
          ? "editorial"
          : "rock";
    set("styleId", id);
    setNotice(
      `Mock 建议：${styles.find((s) => s.id === id)?.name}，请确认后再生成。`,
    );
  };
  return (
    <main className="workspace">
      <header className="workspace-top">
        <Brand />
        <span>NEW PROJECT / {project.id.slice(0, 8).toUpperCase()}</span>
        <button className="ghost-button" onClick={() => nav("/")}>
          <ArrowLeft size={15} /> 返回首页
        </button>
      </header>
      <div className="workspace-grid">
        <aside className="steps">
          <p className="eyebrow">CREATE FLOW</p>
          {["选择场景", "信息与素材", "视觉风格", "输出确认"].map((x, i) => (
            <button
              key={x}
              className={i === step ? "active" : i < step ? "done" : ""}
              onClick={() => setStep(i)}
            >
              <b>{i < step ? <Check size={15} /> : i + 1}</b>
              <span>
                {x}
                <small>
                  {i === step ? "当前步骤" : i < step ? "已完成" : "等待中"}
                </small>
              </span>
              <ChevronRight />
            </button>
          ))}
          <div className="completion">
            <span>
              项目完成度 <b>{completion}%</b>
            </span>
            <i>
              <em style={{ width: `${completion}%` }} />
            </i>
          </div>
          <button
            className="ai-entry"
            onClick={() =>
              document
                .querySelector<HTMLTextAreaElement>("#project-assistant textarea")
                ?.focus()
            }
          >
            <Bot /> AI 项目助理<small>对话提取 · 确认后写入</small>
          </button>
        </aside>
        <section className="form-panel" ref={mainRef}>
          {notice && (
            <div className="notice">
              <Sparkles />
              {notice}
              <button onClick={() => setNotice("")}>×</button>
            </div>
          )}
          {step === 0 && (
            <div className="form-section">
              <p className="eyebrow">STEP 01 / SCENARIO</p>
              <h1>这次要做什么海报？</h1>
              <p>场景决定接下来出现的字段；核心信息始终由你确认。</p>
              {errors.scene && (
                <p className="field-error error-anchor">{errors.scene}</p>
              )}
              <div className="scene-grid">
                {scenes.map(([id, name, desc], i) => (
                  <button
                    key={id}
                    onClick={() => set("scene", id)}
                    className={project.scene === id ? "selected" : ""}
                  >
                    <small>0{i + 1}</small>
                    <b>{name}</b>
                    <span>{desc}</span>
                    {project.scene === id && <Check />}
                  </button>
                ))}
              </div>
            </div>
          )}
          {step === 1 && (
            <div className="form-section">
              <p className="eyebrow">STEP 02 / INFORMATION & ASSETS</p>
              <h1>准确的信息，独立的素材</h1>
              <p>人物、场地、Logo 与二维码承担不同职责，不会混在同一上传框。</p>
              <h2>核心信息</h2>
              <div className="fields two">
                <Field
                  label="活动名称"
                  value={project.title}
                  onChange={(v) => set("title", v)}
                  required
                  error={errors.title}
                />
                <Field
                  label="活动主体"
                  value={project.subject}
                  onChange={(v) => set("subject", v)}
                  required
                  error={errors.subject}
                />
                <Field
                  label="活动主题"
                  value={project.theme}
                  onChange={(v) => set("theme", v)}
                  required
                  error={errors.theme}
                />
                <Field
                  label="日期和时间"
                  value={project.dateTime}
                  onChange={(v) => set("dateTime", v)}
                  required
                  error={errors.dateTime}
                />
                <Field
                  label="城市"
                  value={project.city}
                  onChange={(v) => set("city", v)}
                />
                <Field
                  label="场地名称"
                  value={project.venue}
                  onChange={(v) => set("venue", v)}
                  required
                  error={errors.venue}
                />
              </div>
              <div className="uploads two">
                <AssetUpload
                  label="场地照片"
                  kind="venue"
                  value={project.assets.venue}
                  onChange={(v) =>
                    set("assets", { ...project.assets, venue: v })
                  }
                />
                <AssetUpload
                  label="风格参考图"
                  kind="reference"
                  value={project.assets.reference}
                  onChange={(v) =>
                    set("assets", { ...project.assets, reference: v })
                  }
                />
              </div>
              {project.scene === "concert" && (
                <>
                  <div className="section-title">
                    <div>
                      <h2>乐队列表</h2>
                      <p>每支乐队的名称、Logo 与人物素材保持绑定。</p>
                    </div>
                    <button className="ghost-button" onClick={addBand}>
                      <Plus /> 添加乐队
                    </button>
                  </div>
                  {errors.bands && (
                    <p className="field-error error-anchor">{errors.bands}</p>
                  )}
                  <div className="bands">
                    {project.bands.map((b, i) => (
                      <article key={b.id}>
                        <header>
                          <b>乐队 0{i + 1}</b>
                          <button onClick={() => removeBand(b.id)}>
                            <Trash2 />
                          </button>
                        </header>
                        <div className="fields two">
                          <Field
                            label="乐队名称"
                            value={b.name}
                            onChange={(v) => updateBand(b.id, { name: v })}
                          />
                          <Field
                            label="音乐风格"
                            value={b.genre}
                            onChange={(v) => updateBand(b.id, { genre: v })}
                          />
                        </div>
                        <div className="uploads three">
                          <AssetUpload
                            label="原始 Logo"
                            kind="logo"
                            value={b.logo}
                            onChange={(v) => updateBand(b.id, { logo: v })}
                          />
                          <AssetUpload
                            label="成员合照"
                            kind="person"
                            value={b.groupPhoto}
                            onChange={(v) =>
                              updateBand(b.id, { groupPhoto: v })
                            }
                          />
                          <AssetUpload
                            label="关键成员照片"
                            kind="person"
                            value={b.keyPhoto}
                            onChange={(v) => updateBand(b.id, { keyPhoto: v })}
                          />
                        </div>
                      </article>
                    ))}
                  </div>
                  <h2>票务信息</h2>
                  <div className="fields two">
                    <Field
                      label="票价"
                      value={project.price}
                      onChange={(v) => set("price", v)}
                    />
                    <Field
                      label="购票说明"
                      value={project.ticketInfo}
                      onChange={(v) => set("ticketInfo", v)}
                    />
                  </div>
                  <AssetUpload
                    label="购票二维码"
                    kind="qr"
                    value={project.assets.qr}
                    onChange={(v) =>
                      set("assets", { ...project.assets, qr: v })
                    }
                  />
                </>
              )}
              {project.scene === "lecture" && (
                <>
                  <h2>讲座信息</h2>
                  <div className="fields two">
                    <Field
                      label="主讲人姓名"
                      value={project.speakerName}
                      onChange={(v) => set("speakerName", v)}
                    />
                    <Field
                      label="主办单位"
                      value={project.organizer}
                      onChange={(v) => set("organizer", v)}
                    />
                    <Field
                      label="主讲人简介"
                      type="textarea"
                      value={project.speakerBio}
                      onChange={(v) => set("speakerBio", v)}
                    />
                  </div>
                  {errors.speaker && (
                    <p className="field-error error-anchor">{errors.speaker}</p>
                  )}
                  <div className="uploads two">
                    <AssetUpload
                      label="主讲人照片"
                      kind="person"
                      required
                      value={project.assets.speaker}
                      onChange={(v) =>
                        set("assets", { ...project.assets, speaker: v })
                      }
                    />
                    <AssetUpload
                      label="主办方 Logo"
                      kind="logo"
                      value={project.assets.organizerLogo}
                      onChange={(v) =>
                        set("assets", { ...project.assets, organizerLogo: v })
                      }
                    />
                  </div>
                </>
              )}
            </div>
          )}
          {step === 2 && (
            <div className="form-section">
              <p className="eyebrow">STEP 03 / VISUAL LANGUAGE</p>
              <h1>选择主视觉方向</h1>
              <p>风格只改变主视觉；真实文字与 Logo 仍由独立图层合成。</p>
              <button className="ai-recommend" onClick={recommend}>
                <Sparkles /> AI 帮我推荐 <small>Mock 规则</small>
              </button>
              <div className="style-list">
                {styles.map((s) => (
                  <button
                    key={s.id}
                    className={project.styleId === s.id ? "selected" : ""}
                    onClick={() => set("styleId", s.id)}
                  >
                    <div className={`style-thumb style-${s.id}`}>
                      <i />
                      <i />
                      <span>POSTER</span>
                    </div>
                    <div>
                      <h3>{s.name}</h3>
                      <p>{s.tagline}</p>
                      <small>{s.composition}</small>
                      <div className="swatches">
                        {s.colors.map((c) => (
                          <i key={c} style={{ background: c }} />
                        ))}
                      </div>
                    </div>
                    {project.styleId === s.id && <Check />}
                  </button>
                ))}
              </div>
            </div>
          )}
          {step === 3 && (
            <div className="form-section">
              <p className="eyebrow">STEP 04 / OUTPUT</p>
              <h1>选择要交付的内容</h1>
              <div className="outputs">
                <label className="selected">
                  <input type="checkbox" checked readOnly />
                  <span>
                    <b>静态竖版海报</b>
                    <small>P0 · 1024 × 1536 · PNG</small>
                  </span>
                  <em>默认</em>
                </label>
                <label className={project.outputs.teaser ? "selected" : ""}>
                  <input
                    type="checkbox"
                    checked={project.outputs.teaser}
                    onChange={(e) =>
                      set("outputs", {
                        ...project.outputs,
                        teaser: e.target.checked,
                      })
                    }
                  />
                  <span>
                    <b>5–8 秒宣传片</b>
                    <small>P1 · 规划中 · 会增加生成时间</small>
                  </span>
                  <em>需主动选择</em>
                </label>
                <label className="disabled">
                  <input type="checkbox" disabled />
                  <span>
                    <b>VJ 动态视觉</b>
                    <small>P2 · 后续版本</small>
                  </span>
                  <em>暂不可用</em>
                </label>
              </div>
              <div className="preflight">
                <Check />
                <div>
                  <b>确定性信息图层</b>
                  <p>
                    标题、时间、地点、Logo、二维码将使用你的原始数据合成，不由
                    AI 重画。
                  </p>
                </div>
              </div>
            </div>
          )}
          <div className="form-actions">
            <button
              className="ghost-button"
              disabled={step === 0}
              onClick={() => setStep((s) => s - 1)}
            >
              <ArrowLeft /> 上一步
            </button>
            {step < 3 ? (
              <button className="button" onClick={() => setStep((s) => s + 1)}>
                继续 <ArrowRight />
              </button>
            ) : (
              <button className="button" onClick={generate}>
                提交 W7900 真实生成 <Sparkles />
              </button>
            )}
          </div>
        </section>
        <ProjectAssistant
          project={project}
          onApply={(draft) => {
            setProject((current) => ({ ...current, ...draft }));
            setNotice("AI 候选信息已按你的确认写入表单；请继续检查并补充素材。");
            setStep(1);
          }}
        />
      </div>
    </main>
  );
}
