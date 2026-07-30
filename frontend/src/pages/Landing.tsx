import { Sparkles } from 'lucide-react';
import { Link } from 'react-router-dom';
import HeroStudio from '../components/HeroStudio';
import SceneSelector from '../components/SceneSelector';
import WorkflowTimeline from '../components/WorkflowTimeline';
import PosterGallery from '../components/PosterGallery';
import PosterLanguageToggle from '../components/PosterLanguageToggle';
import {useState} from 'react';
import type {PosterLanguage} from '../types';

export default function Landing() {
  const [language, setLanguage] = useState<PosterLanguage>(() =>
    localStorage.getItem('poster-site-language') === 'zh' ? 'zh' : 'en'
  );
  const changeLanguage = (next: PosterLanguage) => {
    setLanguage(next);
    localStorage.setItem('poster-site-language', next);
  };
  return (
    <main className="landing-studio">
      <div className="landing-language-switch">
        <PosterLanguageToggle value={language} onChange={changeLanguage}/>
      </div>
      <HeroStudio language={language} />
      <SceneSelector />
      <WorkflowTimeline />
      <PosterGallery language={language} />
      <footer className="landing-studio-footer">
        <div className="landing-studio-footer-inner">
          <Link className="brand" to="/">
            <span className="brand-mark">
              <Sparkles size={18} />
            </span>
            POSTER <small>VISUAL LAB</small>
          </Link>
          <p>
            {language === 'en' ? 'Multimodal AI poster creation · Reality collides with visual.' : '多模态 AI 海报创作 · 让真实素材撞上主视觉。'}
            <br />
            <span>POWERED BY AMD RADEON · ROCm 6.x</span>
          </p>
          <small>© 2026 Poster · Visual Lab · Hackathon MVP</small>
        </div>
      </footer>
    </main>
  );
}
