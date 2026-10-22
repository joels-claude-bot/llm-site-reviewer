import type { CSSProperties } from 'react';

const modes = [
  {
    name: 'Batch',
    colour: '#2563eb',
    wash: '#eff6ff',
    border: '#bfdbfe',
    body: 'Send the whole selected corpus in one request. Use it when the files fit comfortably in the model context.',
    tradeoff: 'Simple to run, but relevant details can get lost as the corpus grows.',
  },
  {
    name: 'OpenCode',
    colour: '#ea580c',
    wash: '#fff7ed',
    border: '#fed7aa',
    body: 'Let an agent search and read the repository before returning structured findings. Use it when the relevant evidence is not obvious up front.',
    tradeoff: 'Can follow useful leads, with variable runtime and token use.',
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
        <article key={mode.name} style={{ ...panel, background: mode.wash, borderColor: mode.border, color: '#172033' }}>
          <h3 style={{ margin: 0, color: mode.colour, fontSize: 20 }}>{mode.name}</h3>
          <p style={{ margin: '12px 0', lineHeight: 1.55, fontSize: 14 }}>{mode.body}</p>
          <div style={{ borderTop: `1px solid ${mode.border}`, paddingTop: 12, fontSize: 13 }}>
            <strong style={{ color: mode.colour }}>Trade-off:</strong> {mode.tradeoff}
          </div>
        </article>
      ))}
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
