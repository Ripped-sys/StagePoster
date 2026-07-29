import { Sparkles } from 'lucide-react';
import { Link } from 'react-router-dom';
import HeroStudio from '../components/HeroStudio';
import SceneSelector from '../components/SceneSelector';
import WorkflowTimeline from '../components/WorkflowTimeline';
import PosterGallery from '../components/PosterGallery';

export default function Landing() {
  return (
    <main className="landing-studio">
      <HeroStudio />
      <SceneSelector />
      <WorkflowTimeline />
      <PosterGallery />
      <footer className="landing-studio-footer">
        <div className="landing-studio-footer-inner">
          <Link className="brand" to="/">
            <span className="brand-mark">
              <Sparkles size={18} />
            </span>
            POSTER <small>VISUAL LAB</small>
          </Link>
          <p>
            多模态 AI 海报创作 · 让真实素材撞上主视觉。
            <br />
            <span>POWERED BY AMD RADEON · ROCm 6.x</span>
          </p>
          <small>© 2026 Poster · Visual Lab · Hackathon MVP</small>
        </div>
      </footer>
    </main>
  );
}
