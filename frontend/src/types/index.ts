export type SceneType='concert'|'festival'|'lecture'|'competition'|'commercial'|'custom';
export interface UploadedAsset{id:string;name:string;type:string;dataUrl:string;category:'person'|'venue'|'logo'|'qr'|'reference';status:'success'|'error';error?:string}
export type PosterLanguage='en'|'zh';
export interface Participant{id:string;name:string;nameEn?:string;genre:string;genreEn?:string;logo?:UploadedAsset;groupPhoto?:UploadedAsset;keyPhoto?:UploadedAsset}
export interface StylePreset{id:string;name:string;colors:string[];composition:string;tagline:string}
export interface OutputOptions{poster:boolean;teaser:boolean;vj:boolean}
export interface PosterProject{id:string;scene?:SceneType;posterLanguage?:PosterLanguage;title:string;titleEn?:string;theme:string;themeEn?:string;dateTime:string;city:string;cityEn?:string;venue:string;venueEn?:string;subject:string;subjectEn?:string;price:string;ticketInfo:string;ticketInfoEn?:string;bands:Participant[];speakerName:string;speakerBio:string;organizer:string;assets:Partial<Record<'venue'|'qr'|'reference'|'speaker'|'organizerLogo',UploadedAsset>>;styleId:string;outputs:OutputOptions;visualSeed:number;createdAt:string}
export type GenerationStatus='waiting'|'running'|'done';
export interface GenerationMetrics{gpu:string;rocm:string;resolution:string;duration:string;peakVram:string}
export interface GenerationCandidate{
  candidateId:string;
  variantKey:string;
  variantName:string;
  status:string;
  selected:boolean;
  attempt:number;
  imageUrl?:string;
  spec?:{motif?:string;composition?:string;camera?:string;materials?:string[];palette?:string[];lighting?:string};
}
export interface GenerationTask{
  id:string;
  projectId:string;
  step:number;
  progress:number;
  status:'running'|'complete'|'failed';
  startedAt:number;
  metrics:GenerationMetrics;
  outputUrl?:string;
  error?:string;
  source?:'w7900'|'local';
  remoteStatus?:string;
  candidates?:GenerationCandidate[];
  elapsedSeconds?:number;
  etaSeconds?:number;
}
