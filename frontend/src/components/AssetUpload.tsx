import {useRef, useState} from 'react';
import {CheckCircle2, ImagePlus, LoaderCircle, RefreshCw, Trash2} from 'lucide-react';
import type {UploadedAsset} from '../types';

const MAX_FILE_SIZE = 10 * 1024 * 1024;
const MAX_IMAGE_EDGE = 1600;
const VALID_EXTENSIONS = /\.(png|jpe?g|webp|gif|svg)$/i;

function readAsDataUrl(file: File) {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result));
    reader.onerror = () => reject(new Error('图片读取失败，请重新选择文件'));
    reader.readAsDataURL(file);
  });
}

async function prepareImage(file: File) {
  const source = await readAsDataUrl(file);
  if (file.type === 'image/svg+xml' || file.type === 'image/gif' || file.size < 700 * 1024) {
    return {dataUrl: source, type: file.type || 'image/png'};
  }

  const image = new Image();
  image.src = source;
  await image.decode();
  const scale = Math.min(1, MAX_IMAGE_EDGE / Math.max(image.naturalWidth, image.naturalHeight));
  const canvas = document.createElement('canvas');
  canvas.width = Math.max(1, Math.round(image.naturalWidth * scale));
  canvas.height = Math.max(1, Math.round(image.naturalHeight * scale));
  const context = canvas.getContext('2d');
  if (!context) throw new Error('浏览器无法处理这张图片');
  context.drawImage(image, 0, 0, canvas.width, canvas.height);
  return {dataUrl: canvas.toDataURL('image/webp', 0.88), type: 'image/webp'};
}

interface Props {
  label: string;
  kind: UploadedAsset['category'];
  value?: UploadedAsset;
  onChange: (asset?: UploadedAsset) => void;
  required?: boolean;
}

export default function AssetUpload({label, kind, value, onChange, required = false}: Props) {
  const [busy, setBusy] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const replaceRef = useRef<HTMLInputElement>(null);

  const pick = async (file?: File, input?: HTMLInputElement | null) => {
    if (!file) return;
    const isImage = file.type.startsWith('image/') || VALID_EXTENSIONS.test(file.name);
    if (!isImage) {
      onChange({id: crypto.randomUUID(), name: file.name, type: file.type, dataUrl: '', category: kind, status: 'error', error: '仅支持 PNG、JPG、WEBP、GIF 或 SVG 图片'});
      if (input) input.value = '';
      return;
    }
    if (file.size > MAX_FILE_SIZE) {
      onChange({id: crypto.randomUUID(), name: file.name, type: file.type, dataUrl: '', category: kind, status: 'error', error: '图片不能超过 10MB'});
      if (input) input.value = '';
      return;
    }

    setBusy(true);
    try {
      const prepared = await prepareImage(file);
      onChange({id: crypto.randomUUID(), name: file.name, type: prepared.type, dataUrl: prepared.dataUrl, category: kind, status: 'success'});
    } catch (error) {
      onChange({id: crypto.randomUUID(), name: file.name, type: file.type, dataUrl: '', category: kind, status: 'error', error: error instanceof Error ? error.message : '图片处理失败'});
    } finally {
      setBusy(false);
      if (input) input.value = '';
    }
  };

  return <div className={`upload ${value?.status === 'error' ? 'upload-error' : ''}`}>
    <div className="upload-head">
      <span>{label}{required && <b> *</b>}</span>
      {busy && <em><LoaderCircle className="spin" size={14}/> 正在处理</em>}
      {!busy && value?.status === 'success' && <em><CheckCircle2 size={14}/> 已上传</em>}
    </div>
    {value?.dataUrl ? <div className="upload-preview">
      <img src={value.dataUrl} alt={label}/>
      <div><strong>{value.name}</strong><small>{value.type}</small></div>
    </div> : <label className="drop">
      <ImagePlus size={22}/><span>{busy ? '正在处理图片…' : '点击选择图片'}</span><small>PNG / JPG / WEBP / SVG · 最大 10MB</small>
      <input ref={inputRef} type="file" accept="image/png,image/jpeg,image/webp,image/gif,image/svg+xml" disabled={busy} onChange={event => void pick(event.target.files?.[0], event.currentTarget)}/>
    </label>}
    {value?.error && <p className="field-error">{value.error}</p>}
    {value?.dataUrl && <div className="upload-actions">
      <label className={busy ? 'disabled' : ''}><RefreshCw size={14}/>替换
        <input ref={replaceRef} type="file" accept="image/png,image/jpeg,image/webp,image/gif,image/svg+xml" disabled={busy} onChange={event => void pick(event.target.files?.[0], event.currentTarget)}/>
      </label>
      <button type="button" disabled={busy} onClick={() => onChange()}><Trash2 size={14}/>删除</button>
    </div>}
  </div>;
}
