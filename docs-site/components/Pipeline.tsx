import React, { useState } from 'react';

// A flowchart of the real dataflow, drawn from internal/review/doc.go. Two lanes
// share one vocabulary: the test side (built) turns fixture pages into expected
// findings; the runtime side (mostly a stub) turns a built site into reported
// findings. Each box carries three things a reader needs: what KIND of Go thing
// it is (package / type / func / directory / input), a one-line ROLE in the flow,
// and its real DOC comment lifted verbatim from the source.
//
// TODO(codemap): every `doc` below is hand-copied from the source today. Each
// symbol already carries an `//arch:` tag, and cmd/codemap already reads those
// tags. The next step is for codemap to also extract the doc comment sitting
// above each tagged symbol and emit this data, so the flowchart can never drift
// from the code it describes. Until then, if you edit a doc comment in Go, copy
// the change here too.

const GH = 'https://github.com/joels-claude-bot/llm-site-reviewer/blob/main';

// The visual role a box plays in the flow (drives colour).
type Kind = 'data' | 'code' | 'value' | 'stub';

// What the box actually IS in Go. This is language-specific on purpose: a reader
// coming from another language needs to know a "package" is a directory of Go
// files sharing a namespace, a "type" is a struct, a "func" is a function.
type Symbol = 'input' | 'package' | 'type' | 'func' | 'directory';

const SYMBOL_LABEL: Record<Symbol, string> = {
  input: 'input',
  package: 'Go package',
  type: 'Go type',
  func: 'func',
  directory: 'directory',
};

const SYMBOL_GLOSS: Record<Symbol, string> = {
  input: 'data fed in from outside the program',
  package: 'a directory of Go files sharing one namespace',
  type: 'a named type (here, a struct)',
  func: 'a function',
  directory: 'a folder of files on disk',
};

type Box = {
  id: string;
  label: string;
  kind: Kind;
  symbol: Symbol;
  role: string; // one line: what this step does in the flow (flowchart level)
  doc?: string; // the real doc comment, copied verbatim from the source
  href?: string;
};

// The file that kicks a lane off — the flowchart's "start here".
type Entry = { label: string; href: string };

const KIND: Record<Kind, { border: string; bg: string; dashed?: boolean }> = {
  data: { border: '#64748b', bg: 'rgba(100,116,139,0.12)' },
  code: { border: '#16a34a', bg: 'rgba(22,163,74,0.12)' },
  value: { border: '#2563eb', bg: 'rgba(37,99,235,0.12)' },
  stub: { border: '#d97706', bg: 'rgba(217,119,6,0.10)', dashed: true },
};

const TEST_ENTRY: Entry = {
  label: 'corpus_test.go calls Load',
  href: `${GH}/internal/corpus/corpus_test.go#L37`,
};

const TEST: Box[] = [
  {
    id: 'md',
    label: 'corpus/*.md',
    kind: 'data',
    symbol: 'directory',
    role: 'The fixture pages on disk. Each frontmatter states the findings that page should produce.',
    doc: 'Each fixture is a small page that plants one defect, and the result we expect from it lives in the same file’s frontmatter, so the two stay in sync.',
    href: `${GH}/corpus`,
  },
  {
    id: 'load',
    label: 'corpus.Load',
    kind: 'code',
    symbol: 'func',
    role: 'Walks corpus/ and parses each page into a Fixture that carries its expected findings.',
    doc: 'Load reads and parses every fixture under root. It does not validate, so callers can load without rule-checking.',
    href: `${GH}/internal/corpus/corpus.go#L111`,
  },
  {
    id: 'fix',
    label: '[]corpus.Fixture',
    kind: 'value',
    symbol: 'type',
    role: 'The expected answers, in memory: a page on disk becomes the findings we assert against.',
    doc: 'Fixture is one corpus page plus the result we expect from it.',
    href: `${GH}/internal/corpus/corpus.go#L27`,
  },
];

const RUN_ENTRY: Entry = {
  label: 'no caller yet — a cmd/ main will call review.Review',
  href: `${GH}/internal/review/doc.go`,
};

const RUN: Box[] = [
  {
    id: 'site',
    label: 'a built docs site',
    kind: 'data',
    symbol: 'input',
    role: 'The rendered site a visitor actually sees — HTML, screenshots, links. The runtime input.',
  },
  {
    id: 'review',
    label: 'review.Review',
    kind: 'stub',
    symbol: 'func',
    role: 'The planned entry function. review is the spine package; Review() will run each pass and collect their findings. Today only the package doc (doc.go) exists — there is no Review function yet.',
    doc: 'Package review is the spine: it runs each pass over a site and collects their findings into one report. It is not built yet; this file marks the package and holds the map, so the shape shows up in the tree before the code does.',
    href: `${GH}/internal/review/doc.go#L1`,
  },
  {
    id: 'found',
    label: '[]finding.Finding',
    kind: 'value',
    symbol: 'type',
    role: 'What the passes actually saw, in the same vocabulary the fixtures use.',
    doc: 'Finding is one reported defect: its category, where on the page it sits, and what should happen when it is raised.',
    href: `${GH}/internal/finding/finding.go#L64`,
  },
  {
    id: 'report',
    label: 'FormatReport',
    kind: 'stub',
    symbol: 'func',
    role: 'Planned: turns the findings into the output a user reads. No code yet — named only in the review doc.',
  },
];

const PASSES: Box[] = [
  {
    id: 'links',
    label: 'links.Check',
    kind: 'stub',
    symbol: 'func',
    role: 'Deterministic, no model: broken internal links, anchors, missing images.',
    doc: 'links.Check — deterministic, no model. (planned; named in the review doc)',
  },
  {
    id: 'render',
    label: 'render.Check',
    kind: 'stub',
    symbol: 'func',
    role: 'Model: looks at screenshots — truncation, flat colour, weak hierarchy.',
    doc: 'render.Check — model: looks at screenshots. (planned; named in the review doc)',
  },
  {
    id: 'clarity',
    label: 'clarity.Check',
    kind: 'stub',
    symbol: 'func',
    role: 'Model: reads the prose — unexpanded acronyms, assumed jargon.',
    doc: 'clarity.Check — model: reads the prose. (planned; named in the review doc)',
  },
  {
    id: 'mismatch',
    label: 'mismatch.Check',
    kind: 'stub',
    symbol: 'func',
    role: 'Mixed: stale refs and claims that contradict the page or the code.',
    doc: 'mismatch.Check — mixed. (planned; named in the review doc)',
  },
];

// The shared vocabulary both sides speak. This is the PACKAGE finding, not a
// single type: the same file defines the Finding type, the 25 Category codes,
// and the Result enum. That is the whole point of the box — one word-list.
const FINDING: Box = {
  id: 'finding',
  label: 'package finding',
  kind: 'value',
  symbol: 'package',
  role: 'The shared vocabulary. One package holding three things: the finding.Finding type (one defect), the finding.Category list (25 codes), and finding.Result (blocking vs report). Both lanes speak it, so review does not care how a pass reached its answer.',
  doc: 'Package finding defines a reported defect: its category, and whether that category can block a build or is only reported.',
  href: `${GH}/internal/finding/finding.go`,
};

function BoxView({
  box,
  active,
  onHover,
}: {
  box: Box;
  active: boolean;
  onHover: (box: Box | null) => void;
}) {
  const style = KIND[box.kind];
  const inner = (
    <div
      onMouseEnter={() => onHover(box)}
      onMouseLeave={() => onHover(null)}
      style={{
        border: `2px ${style.dashed ? 'dashed' : 'solid'} ${style.border}`,
        background: style.bg,
        borderRadius: 9,
        padding: '8px 12px',
        fontFamily: 'var(--rp-font-mono, ui-monospace, monospace)',
        fontSize: 13,
        fontWeight: 600,
        color: 'var(--rp-c-text-1)',
        whiteSpace: 'nowrap',
        cursor: box.href ? 'pointer' : 'default',
        outline: active ? `2px solid ${style.border}` : 'none',
        outlineOffset: 2,
        transition: 'outline 0.1s',
      }}
    >
      {box.label}
      <span style={{ marginLeft: 6, fontSize: 9, fontWeight: 500, opacity: 0.75, textTransform: 'uppercase', letterSpacing: 0.4 }}>
        {SYMBOL_LABEL[box.symbol]}
      </span>
      {box.kind === 'stub' && <span style={{ marginLeft: 6, fontSize: 10, color: '#d97706' }}>stub</span>}
    </div>
  );
  return box.href ? (
    <a href={box.href} target="_blank" rel="noreferrer" style={{ textDecoration: 'none' }}>
      {inner}
    </a>
  ) : (
    inner
  );
}

function Arrow({ label }: { label?: string }) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', color: 'var(--rp-c-text-2)', minWidth: 34 }}>
      {label && <span style={{ fontSize: 10 }}>{label}</span>}
      <span style={{ fontSize: 18, lineHeight: 1 }}>→</span>
    </div>
  );
}

function EntryChip({ entry }: { entry: Entry }) {
  return (
    <a
      href={entry.href}
      target="_blank"
      rel="noreferrer"
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 5,
        fontSize: 11,
        color: 'var(--rp-c-text-2)',
        textDecoration: 'none',
        border: '1px dashed var(--rp-c-divider)',
        borderRadius: 999,
        padding: '2px 9px',
        marginBottom: 8,
        width: 'fit-content',
      }}
    >
      <span style={{ fontSize: 11 }}>▶ start:</span>
      <span style={{ fontFamily: 'var(--rp-font-mono, ui-monospace, monospace)' }}>{entry.label}</span>
    </a>
  );
}

function Lane({
  title,
  subtitle,
  entry,
  boxes,
  active,
  onHover,
}: {
  title: string;
  subtitle: string;
  entry: Entry;
  boxes: Box[];
  active: string | null;
  onHover: (box: Box | null) => void;
}) {
  return (
    <div style={{ marginBottom: 8 }}>
      <div style={{ fontSize: 12, fontWeight: 700, color: 'var(--rp-c-text-1)' }}>{title}</div>
      <div style={{ fontSize: 12, color: 'var(--rp-c-text-2)', marginBottom: 8 }}>{subtitle}</div>
      <EntryChip entry={entry} />
      <div style={{ display: 'flex', alignItems: 'center', flexWrap: 'wrap', gap: 4 }}>
        {boxes.map((box, index) => (
          <React.Fragment key={box.id}>
            <BoxView box={box} active={active === box.id} onHover={onHover} />
            {index < boxes.length - 1 && <Arrow />}
          </React.Fragment>
        ))}
      </div>
    </div>
  );
}

export default function Pipeline() {
  const [hovered, setHovered] = useState<Box | null>(null);
  const active = hovered?.id ?? null;

  return (
    <div
      style={{
        border: '1px solid var(--rp-c-divider)',
        borderRadius: 12,
        padding: 16,
        margin: '20px 0',
        background: 'var(--rp-c-bg)',
      }}
    >
      <Lane
        title="Test side — built"
        subtitle="fixture pages become the answers we assert against"
        entry={TEST_ENTRY}
        boxes={TEST}
        active={active}
        onHover={setHovered}
      />

      <Lane
        title="Runtime side — mostly a stub"
        subtitle="a real site becomes the findings we report"
        entry={RUN_ENTRY}
        boxes={RUN}
        active={active}
        onHover={setHovered}
      />

      {/* the passes hang off review.Review */}
      <div style={{ marginLeft: 24, paddingLeft: 16, borderLeft: '2px dashed #d97706', margin: '4px 0 12px 24px' }}>
        <div style={{ fontSize: 12, color: 'var(--rp-c-text-2)', marginBottom: 6 }}>
          review.Review fans out to the passes (none built yet); each returns <code>finding.Finding</code> values:
        </div>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
          {PASSES.map((box) => (
            <BoxView key={box.id} box={box} active={active === box.id} onHover={setHovered} />
          ))}
        </div>
      </div>

      {/* the shared vocabulary both sides speak */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, borderTop: '1px dashed var(--rp-c-divider)', paddingTop: 12 }}>
        <span style={{ fontSize: 12, color: 'var(--rp-c-text-2)' }}>both sides speak</span>
        <BoxView box={FINDING} active={active === FINDING.id} onHover={setHovered} />
        <span style={{ fontSize: 12, color: 'var(--rp-c-text-2)' }}>
          — so the test side can compare a pass’s findings against a fixture’s expected findings.
        </span>
      </div>

      {/* live detail panel: role + the real doc comment ("duck string") */}
      <div
        style={{
          marginTop: 12,
          minHeight: 96,
          borderRadius: 8,
          background: 'var(--rp-c-bg-soft)',
          padding: '10px 12px',
          fontSize: 13,
          color: 'var(--rp-c-text-1)',
        }}
      >
        {hovered ? (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
              <strong style={{ fontFamily: 'var(--rp-font-mono, ui-monospace, monospace)' }}>{hovered.label}</strong>
              <span
                style={{
                  fontSize: 10,
                  textTransform: 'uppercase',
                  letterSpacing: 0.4,
                  border: '1px solid var(--rp-c-divider)',
                  borderRadius: 4,
                  padding: '1px 6px',
                  color: 'var(--rp-c-text-2)',
                }}
                title={SYMBOL_GLOSS[hovered.symbol]}
              >
                {SYMBOL_LABEL[hovered.symbol]}
              </span>
              <span style={{ fontSize: 11, color: 'var(--rp-c-text-2)' }}>— {SYMBOL_GLOSS[hovered.symbol]}</span>
            </div>

            <span>{hovered.role}</span>

            {hovered.doc && (
              <div
                style={{
                  fontFamily: 'var(--rp-font-mono, ui-monospace, monospace)',
                  fontSize: 12,
                  color: 'var(--rp-c-text-2)',
                  borderLeft: '3px solid var(--rp-c-divider)',
                  paddingLeft: 10,
                  whiteSpace: 'pre-wrap',
                }}
              >
                {'// ' + hovered.doc}
              </div>
            )}

            <span style={{ fontSize: 11, color: 'var(--rp-c-text-2)' }}>
              doc comment copied from the source
              {hovered.href && <> · click the box to open the code</>}
            </span>
          </div>
        ) : (
          <span style={{ color: 'var(--rp-c-text-2)' }}>
            Hover a box for its role, what kind of Go thing it is, and its real doc comment. Solid = built,
            dashed = not built yet.
          </span>
        )}
      </div>
    </div>
  );
}
