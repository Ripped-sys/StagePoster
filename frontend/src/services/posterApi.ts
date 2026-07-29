import type{GenerationTask,PosterProject}from'../types';

const BASE=((import.meta.env.VITE_API_BASE_URL as string|undefined)??(import.meta.env.VITE_API_URL as string|undefined)??'http://127.0.0.1:8080').replace(/\/$/,'');
const TOKEN=(import.meta.env.VITE_POSTER_TOKEN as string|undefined)??'';
const metrics={gpu:'AMD Radeon Pro W7900',rocm:'远程 GPU 节点',resolution:'服务端输出',duration:'W7900 推理中…',peakVram:'由 GPU 服务管理'};
interface Created{jobId:string;promptId:string;status:string;seed:number}
interface RemoteJob{jobId:string;status:'queued'|'running'|'succeeded'|'failed';result?:{filename:string;type:string};error?:string}
interface Health{status:string;comfy:string;tokenRequired:boolean}

async function json<T>(path:string,init?:RequestInit):Promise<T>{let last:unknown;for(let attempt=0;attempt<3;attempt+=1){try{const response=await fetch(`${BASE}${path}`,{...init,headers:{Accept:'application/json',...init?.headers}});if(response.ok)return response.json() as Promise<T>;if(response.status<500||attempt===2)throw new Error(`StagePoster API ${response.status}: ${await response.text()}`)}catch(error){last=error;if(attempt<2)await new Promise(resolve=>window.setTimeout(resolve,500*(attempt+1)));}}throw last instanceof Error?last:new Error('StagePoster API 暂时不可用')}

const SYSTEM_VISUAL_PROMPT=`You are the visual-art director for a Chinese independent live-music project. Create only the image layer that sits beneath programmatic typography, original band logos and ticket information. The house style is distilled from a real archive of Xi'an underground-show artwork rather than commercial concert advertising: handmade cut-and-paste collage, photocopy and risograph grain, uneven screen-print ink, torn and weathered surfaces, accidental registration errors, analog photographic blur, raw drawings, found objects, symbolic imagery and experimental editorial spacing. The work may feel gothic, absurd, poetic or digitally damaged, but must always feel locally made, specific and imperfect. Favor one memorable visual metaphor over a literal generic concert scene. Use a vertical 2:3 canvas, deliberate asymmetry or severe centered symmetry, bold value contrast, tactile material depth and controlled visual density. Reserve usable breathing room near the top and bottom for the application's exact information layers. Avoid polished commercial advertising, generic festival photography, stock imagery, esports styling and glossy AI-render aesthetics.`;

const STRICT_VISUAL_RULES=`Artwork only. Absolutely no words, Chinese characters, English letters, numbers, logos, band names, captions, signs, watermark, QR code, interface elements, fake typography or symbols resembling writing. Do not generate a recognizable human face unless explicitly requested. Avoid stadiums, football fields, generic leather-jacket portraits, glossy 3D rendering, esports aesthetics, stock photography, plastic skin, excessive neon and generic cyberpunk cities. The frontend will add all factual text, original logos and QR codes as deterministic layers.`;

const STYLE_PROMPTS:Record<string,string>={
 rock:'ARCHIVE STYLE — OCCULT METAL COLLAGE. Use charcoal black, dirty bone white, dried-blood red, oxidized brown and muted gray-green. Combine decayed classical imagery, thorn or bone-like ornament, corroded metal, smoke, ritual objects and distressed paper into a dense but controlled collage. Favor confrontational symmetry, a monumental central emblem or two opposing symbolic masses. Add photocopy noise, scratched ink, torn edges and imperfect screen-print registration. The mood is solemn, dangerous, ancient and theatrical, never a clean modern rock photograph.',
 cyber:'ARCHIVE STYLE — EXPERIMENTAL DIGITAL ZINE. Use deep blue-black with restrained electric cyan, acid green, violet and magenta. Combine analog video feedback, RGB channel separation, halftone dots, scan noise, blurred documentary fragments, translucent waveforms and awkward floating geometric shapes. Use asymmetric editorial spacing and deliberate low-resolution artifacts. The result should resemble a small independent club flyer assembled from damaged screens and photocopies, not glossy cyberpunk concept art.',
 editorial:'ARCHIVE STYLE — POETIC DIY EDITORIAL. Use warm off-white, charcoal, faded forest green, muted rust and one unexpected fluorescent accent. Build the image from a single poetic subject, found photography, primitive hand-drawn marks, cut paper, risograph texture and generous quiet space. Allow unusual cropping and modest asymmetry. The result should feel intimate, literary, inexpensive and handmade, with the calm tension of an independent-music poster rather than a corporate minimalist template.'
};

export function buildPosterPrompt(project:PosterProject){
 const genres=[...new Set(project.bands.map(b=>b.genre.trim()).filter(Boolean))].join(', ');
 const bandContext=project.bands.length>1?`${project.bands.length} contrasting musical identities in visual tension.`:project.bands.length===1?'One central musical identity.':'';
 const eventConcept=[project.theme,project.subject,bandContext,genres&&`Musical atmosphere: ${genres}.`,project.city&&`Cultural atmosphere inspired by ${project.city}.`].filter(Boolean).join(' ');
 return [SYSTEM_VISUAL_PROMPT,STYLE_PROMPTS[project.styleId]??STYLE_PROMPTS.rock,`Event concept to interpret visually: ${eventConcept}`,STRICT_VISUAL_RULES].join(' ');
}

export const posterApi={
 health:()=>json<Health>('/health'),
 async submit(project:PosterProject):Promise<GenerationTask>{const seed=Math.abs(project.visualSeed||Math.floor(Math.random()*2_147_483_647));const created=await json<Created>('/api/generate',{method:'POST',headers:{'Content-Type':'application/json',...(TOKEN?{'X-Poster-Token':TOKEN}:{})},body:JSON.stringify({prompt:buildPosterPrompt(project),seed})});return{id:created.jobId,projectId:project.id,step:0,progress:2,status:'running',startedAt:Date.now(),metrics,source:'w7900'}},
 async getJob(task:GenerationTask):Promise<GenerationTask>{const remote=await json<RemoteJob>(`/api/jobs/${task.id}`,{headers:TOKEN?{'X-Poster-Token':TOKEN}:{}});if(remote.status==='failed')return{...task,status:'failed',error:remote.error??'GPU generation failed'};if(remote.status==='succeeded')return{...task,status:'complete',step:5,progress:100,outputUrl:`${BASE}/api/jobs/${task.id}/result`,metrics:{...task.metrics,duration:`${((Date.now()-task.startedAt)/1000).toFixed(1)} s · W7900`}};const elapsed=Date.now()-task.startedAt;const progress=remote.status==='queued'?8:Math.min(88,20+Math.round(elapsed/1200));return{...task,step:remote.status==='queued'?0:2,progress,status:'running'}},
 async resultBlob(jobId:string){const response=await fetch(`${BASE}/api/jobs/${jobId}/result`,{headers:{'X-Poster-Token':TOKEN}});if(!response.ok)throw new Error(`Result ${response.status}`);return response.blob()},
 resultUrl:(jobId:string)=>`${BASE}/api/jobs/${jobId}/result`
};

export function localMockTask(projectId:string):GenerationTask{return{id:projectId,projectId,step:0,progress:4,status:'running',startedAt:Date.now(),metrics:{gpu:'浏览器本地 Mock',rocm:'未连接',resolution:'1024 × 1536',duration:'本地 Mock',peakVram:'—'},source:'local'}}
