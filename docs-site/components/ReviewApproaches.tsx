import type { CSSProperties, ReactNode } from 'react';

const modes = [
  {
    icon: '📦',
    name: 'Batch',
    strapline: 'Everything, all at once',
    colour: '#2563eb',
    wash: '#eff6ff',
    border: '#bfdbfe',
    body: 'One request contains the whole corpus. The simplest path when the repository fits comfortably in a strong model’s context window.',
    tags: ['1 model call', 'maximum context'],
    tradeoff: 'Relevant facts get diluted as the corpus grows.',
  },
  {
    icon: '🧩',
    name: 'Dependency',
    strapline: 'Summarise, connect, review',
    colour: '#7c3aed',
    wash: '#f5f3ff',
    border: '#ddd6fe',
    body: 'Build compact summaries and resolve likely dependencies before reviewing each target. Useful for small-context or local models.',
    tags: ['small contexts', 'high control'],
    tradeoff: 'Lossy summaries and hand-built links can hide evidence.',
  },
  {
    icon: '🧭',
    name: 'Controlled',
    strapline: 'Model judgment, Go guardrails',
    colour: '#059669',
    wash: '#ecfdf5',
    border: '#a7f3d0',
    body: 'The model plans searches and selects evidence; Go enforces the budgets and performs concurrent search, reads and image work.',
    tags: ['3 model calls', 'bounded work'],
    tradeoff: 'Predictable, but a weak search plan can reduce recall.',
    recommended: true,
  },
  {
    icon: '🔭',
    name: 'Full OpenCode',
    strapline: 'Let the agent investigate',
    colour: '#ea580c',
    wash: '#fff7ed',
    border: '#fed7aa',
    body: 'A read-only agent repeatedly chooses read, grep and glob calls. Best when finding the right evidence matters more than fixed latency.',
    tags: ['adaptive search', 'highest autonomy'],
    tradeoff: 'Tool limits are advisory; time and token use vary.',
  },
];

const panel: CSSProperties = {
  borderRadius: 16,
  padding: 20,
  border: '1px solid var(--rp-c-divider-light, #e2e8f0)',
};

export function ReviewModeCards() {
  return (
    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(230px, 1fr))', gap: 16, margin: '24px 0 32px' }}>
      {modes.map((mode) => (
        <article key={mode.name} style={{ ...panel, position: 'relative', background: mode.wash, borderColor: mode.border, color: '#172033' }}>
          {mode.recommended && (
            <span style={{ position: 'absolute', top: 14, right: 14, borderRadius: 999, padding: '3px 8px', background: mode.colour, color: 'white', fontSize: 11, fontWeight: 800, letterSpacing: '.04em', textTransform: 'uppercase' }}>
              Default
            </span>
          )}
          <div style={{ fontSize: 28, marginBottom: 10 }}>{mode.icon}</div>
          <h3 style={{ margin: 0, color: mode.colour, fontSize: 20 }}>{mode.name}</h3>
          <div style={{ margin: '2px 0 12px', fontWeight: 700, fontSize: 13, color: '#475569' }}>{mode.strapline}</div>
          <p style={{ margin: '0 0 14px', lineHeight: 1.55, fontSize: 14 }}>{mode.body}</p>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginBottom: 14 }}>
            {mode.tags.map((tag) => <span key={tag} style={{ borderRadius: 999, padding: '3px 8px', background: '#ffffffb8', border: `1px solid ${mode.border}`, fontSize: 12, fontWeight: 650 }}>{tag}</span>)}
          </div>
          <div style={{ borderTop: `1px solid ${mode.border}`, paddingTop: 12, fontSize: 13 }}>
            <strong style={{ color: mode.colour }}>Watch for:</strong> {mode.tradeoff}
          </div>
        </article>
      ))}
    </div>
  );
}

function Step({ number, colour, title, children }: { number: string; colour: string; title: string; children: ReactNode }) {
  return (
    <div style={{ display: 'grid', gridTemplateColumns: '34px 1fr', gap: 10, alignItems: 'start' }}>
      <span style={{ width: 30, height: 30, borderRadius: 9, display: 'grid', placeItems: 'center', background: colour, color: 'white', fontWeight: 800 }}>{number}</span>
      <div><strong style={{ display: 'block', color: '#172033' }}>{title}</strong><span style={{ color: '#475569', fontSize: 13 }}>{children}</span></div>
    </div>
  );
}

export function ControlledPipeline() {
  return (
    <div style={{ ...panel, background: 'linear-gradient(135deg, #ecfdf5 0%, #eff6ff 100%)', borderColor: '#a7f3d0', margin: '20px 0 30px' }}>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(190px, 1fr))', gap: 18 }}>
        <Step number="1" colour="#059669" title="Plan">Model returns focused search patterns.</Step>
        <Step number="2" colour="#0284c7" title="Gather">Go searches concurrently and caps snippets.</Step>
        <Step number="3" colour="#7c3aed" title="Select">Model chooses the strongest evidence pages.</Step>
        <Step number="4" colour="#ea580c" title="Review">Go assembles evidence; model returns the schema.</Step>
      </div>
      <div style={{ marginTop: 18, padding: '10px 13px', borderRadius: 10, background: '#ffffffb8', color: '#334155', fontSize: 13 }}>
        <strong>Exactly three model calls.</strong> Search, reads and image transcription are bounded by code—not by a hopeful prompt.
      </div>
    </div>
  );
}

export function ImagePipeline() {
  return (
    <div style={{ ...panel, background: 'linear-gradient(110deg, #fff7ed, #fdf2f8)', borderColor: '#fed7aa', margin: '18px 0 28px' }}>
      <div style={{ display: 'flex', alignItems: 'center', flexWrap: 'wrap', gap: 9, color: '#172033', fontWeight: 750 }}>
        <span style={{ padding: '8px 12px', background: '#fff', borderRadius: 10 }}>Markdown image</span>
        <span style={{ color: '#ea580c' }}>→</span>
        <span style={{ padding: '8px 12px', background: '#fff', borderRadius: 10 }}>SHA-256 cache</span>
        <span style={{ color: '#ea580c' }}>→</span>
        <span style={{ padding: '8px 12px', background: '#fff', borderRadius: 10 }}>SVG text / raster vision</span>
        <span style={{ color: '#ea580c' }}>→</span>
        <span style={{ padding: '8px 12px', background: '#fff', borderRadius: 10 }}>Review context</span>
      </div>
      <p style={{ margin: '14px 0 0', color: '#475569', fontSize: 13 }}>
        Unchanged local images reuse their transcription. The cache is shared preprocessing, not extra agent tool work.
      </p>
    </div>
  );
}
