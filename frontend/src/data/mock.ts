import type {PosterProject,StylePreset} from '../types';
export const styles:StylePreset[]=[
 {id:'rock',name:'Underground Rock',colors:['#0D0E13','#C8603D','#d4a05c'],composition:'双主体对峙 · 粗粝大标题 · 舞台逆光',tagline:'暗黑 / 暖橙 / 颗粒 / 现场感'},
 {id:'cyber',name:'Cyber Neon',colors:['#07131C','#00D9FF','#D946EF'],composition:'中心透视 · 数字故障 · 高反差轮廓光',tagline:'紫色 / 青色 / 数字噪点 / 未来现场'},
 {id:'editorial',name:'Editorial Minimal',colors:['#f0ebe0','#0D0E13','#C8603D'],composition:'非对称网格 · 大留白 · 清晰信息层级',tagline:'克制 / 现代 / 编辑式排版'}
];
export const emptyProject=():PosterProject=>({id:crypto.randomUUID(),title:'',theme:'',dateTime:'',city:'',venue:'',subject:'',price:'',ticketInfo:'',bands:[],speakerName:'',speakerBio:'',organizer:'',assets:{},styleId:'rock',outputs:{poster:true,teaser:false,vj:false},visualSeed:0,createdAt:new Date().toISOString()});
export const demoProject=():PosterProject=>({...emptyProject(),id:'demo-changan',scene:'concert',title:'长安双雄',theme:'两支金属乐队在古城展开对决',dateTime:'2026-08-08 20:00',city:'西安',venue:'大雁塔附近',subject:'两支金属乐队',price:'¥128',ticketInfo:'现场扫码购票 · 限量入场',bands:[{id:'a',name:'内网穿透 NATP',genre:'工业金属'},{id:'b',name:'示例金属乐队',genre:'战争金属'}],styleId:'rock'});
