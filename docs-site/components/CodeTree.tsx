import React, { useState } from 'react';

// An interactive, annotated file tree for the "How it fits together" page.
// Each node says what it is for in one line; folders expand to reveal the tagged
// symbols inside, each linking to the exact line in the source on GitHub. The
// point is to read the layout by eye and jump straight to the code, not to keep
// a hand-written bullet list in step with the tree.

const GH = 'https://github.com/joels-claude-bot/llm-site-reviewer/blob/main';
const GH_TREE = 'https://github.com/joels-claude-bot/llm-site-reviewer/tree/main';

// role -> colour. These match the pipeline diagram and the //arch: tags.
const ROLE = {
  types: { label: 'vocabulary', bg: '#2563eb' },
  test: { label: 'test side', bg: '#16a34a' },
  spine: { label: 'spine', bg: '#7c3aed' },
  tooling: { label: 'tooling', bg: '#64748b' },
  data: { label: 'data', bg: '#d97706' },
};

type Symbol = { name: string; role: string; href: string; note: string };
type Node = {
  name: string;
  note: string;
  role?: keyof typeof ROLE;
  href?: string;
  stub?: boolean;
  symbols?: Symbol[];
  children?: Node[];
};

const s = (name: string, role: string, file: string, line: number, note: string): Symbol => ({
  name,
  role,
  href: `${GH}/${file}#L${line}`,
  note,
});

const TREE: Node[] = [
  {
    name: 'internal/',
    note: 'the library — pure logic, no main()',
    children: [
      {
        name: 'finding/',
        role: 'types',
        note: 'the shared vocabulary every package speaks',
        href: `${GH}/internal/finding/finding.go`,
        symbols: [
          s('Category (25 codes)', 'types', 'internal/finding/finding.go', 23, 'the failure taxonomy: BROKEN_ANCHOR, ACRONYM_UNEXPANDED, …'),
          s('Finding', 'types', 'internal/finding/finding.go', 73, 'one reported defect: a category and where it is'),
          s('Result', 'types', 'internal/finding/finding.go', 8, 'blocking (fails the build) vs report (noted only)'),
        ],
      },
      {
        name: 'corpus/',
        role: 'test',
        note: 'turns fixture pages into the findings we expect from them',
        href: `${GH}/internal/corpus/corpus.go`,
        symbols: [
          s('Load', 'io', 'internal/corpus/corpus.go', 115, 'walks corpus/ and parses every fixture into a Fixture'),
          s('FixturePaths', 'io', 'internal/corpus/corpus.go', 145, 'lists every fixture file, sorted, for stable output'),
          s('Parse', 'pure', 'internal/corpus/corpus.go', 56, 'decodes one file’s bytes into a Fixture'),
          s('Validate', 'pure', 'internal/corpus/corpus.go', 85, 'checks one fixture against the corpus rules'),
        ],
      },
      {
        name: 'review/',
        role: 'spine',
        note: 'will run each pass over a site and collect the findings',
        href: `${GH}/internal/review/doc.go`,
        stub: true,
        symbols: [
          s('Review', 'spine', 'internal/review/doc.go', 21, 'the spine (documented stub): built site → findings → report'),
        ],
      },
      {
        name: 'codemap/',
        role: 'tooling',
        note: 'reads //arch: tags from the AST to build a map like this one',
        href: `${GH_TREE}/internal/codemap`,
        symbols: [
          s('Extract', 'io', 'internal/codemap/codemap.go', 40, 'parses the source, returns one Entry per tagged symbol'),
          s('Text', 'pure', 'internal/codemap/render.go', 14, 'renders the entries grouped by role, with file:line'),
        ],
      },
    ],
  },
  {
    name: 'cmd/',
    note: 'the command-line entry points — each has a main()',
    children: [
      {
        name: 'inspect/',
        role: 'tooling',
        note: 'dumps what the corpus pipeline produced, to eyeball it',
        href: `${GH}/cmd/inspect/main.go`,
      },
      {
        name: 'codemap/',
        role: 'tooling',
        note: 'prints the architecture map built from the //arch: tags',
        href: `${GH}/cmd/codemap/main.go`,
      },
      {
        name: 'corpusdocs/',
        role: 'tooling',
        note: 'generates the corpus docs page from the fixtures',
        href: `${GH}/cmd/corpusdocs/main.go`,
      },
    ],
  },
  {
    name: 'corpus/',
    role: 'data',
    note: 'the fixture pages themselves — data, not code',
    href: `${GH_TREE}/corpus`,
  },
];

function Badge({ role }: { role: keyof typeof ROLE }) {
  const r = ROLE[role];
  return (
    <span
      style={{
        background: r.bg,
        color: '#fff',
        fontSize: 11,
        fontWeight: 600,
        borderRadius: 5,
        padding: '1px 7px',
        whiteSpace: 'nowrap',
      }}
    >
      {r.label}
    </span>
  );
}

function SymbolRow({ sym }: { sym: Symbol }) {
  return (
    <div style={{ display: 'flex', gap: 8, padding: '3px 0', alignItems: 'baseline', flexWrap: 'wrap' }}>
      <a
        href={sym.href}
        target="_blank"
        rel="noreferrer"
        style={{ fontFamily: 'var(--rp-font-mono, ui-monospace, monospace)', fontSize: 13, fontWeight: 600, color: 'var(--rp-c-brand)' }}
      >
        {sym.name}
      </a>
      <span style={{ fontSize: 12.5, color: 'var(--rp-c-text-2)' }}>{sym.note}</span>
    </div>
  );
}

function TreeNode({ node, depth, last }: { node: Node; depth: number; last: boolean }) {
  const expandable = !!(node.children || node.symbols);
  const [open, setOpen] = useState(depth === 0);

  return (
    <div style={{ position: 'relative' }}>
      <div
        onClick={() => expandable && setOpen((o) => !o)}
        style={{
          display: 'flex',
          alignItems: 'baseline',
          gap: 8,
          padding: '6px 8px',
          borderRadius: 7,
          cursor: expandable ? 'pointer' : 'default',
          flexWrap: 'wrap',
        }}
        onMouseEnter={(event) => (event.currentTarget.style.background = 'var(--rp-c-bg-soft)')}
        onMouseLeave={(event) => (event.currentTarget.style.background = 'transparent')}
      >
        <span style={{ width: 14, color: 'var(--rp-c-text-2)', fontSize: 11 }}>
          {expandable ? (open ? '▾' : '▸') : ''}
        </span>
        {node.href ? (
          <a
            href={node.href}
            target="_blank"
            rel="noreferrer"
            onClick={(event) => event.stopPropagation()}
            style={{ fontFamily: 'var(--rp-font-mono, ui-monospace, monospace)', fontWeight: 700, fontSize: 14, color: 'var(--rp-c-text-1)' }}
          >
            {node.name}
          </a>
        ) : (
          <span style={{ fontFamily: 'var(--rp-font-mono, ui-monospace, monospace)', fontWeight: 700, fontSize: 14, color: 'var(--rp-c-text-1)' }}>
            {node.name}
          </span>
        )}
        {node.role && <Badge role={node.role} />}
        {node.stub && (
          <span style={{ fontSize: 11, fontWeight: 600, color: '#d97706', border: '1px solid #d97706', borderRadius: 5, padding: '0 6px' }}>
            stub
          </span>
        )}
        <span style={{ fontSize: 13, color: 'var(--rp-c-text-2)' }}>{node.note}</span>
      </div>

      {open && expandable && (
        <div
          style={{
            marginLeft: 20,
            paddingLeft: 14,
            borderLeft: '1px dashed var(--rp-c-divider)',
          }}
        >
          {node.symbols?.map((sym) => (
            <SymbolRow key={sym.name} sym={sym} />
          ))}
          {node.children?.map((child, index) => (
            <TreeNode key={child.name} node={child} depth={depth + 1} last={index === node.children!.length - 1} />
          ))}
        </div>
      )}
    </div>
  );
}

export default function CodeTree() {
  return (
    <div
      style={{
        border: '1px solid var(--rp-c-divider)',
        borderRadius: 12,
        padding: '10px 12px',
        margin: '20px 0',
        background: 'var(--rp-c-bg)',
      }}
    >
      <div style={{ fontSize: 12.5, color: 'var(--rp-c-text-2)', padding: '2px 8px 8px' }}>
        Click a folder to expand it. Every name links to the source; each symbol jumps to its line.
      </div>
      {TREE.map((node, index) => (
        <TreeNode key={node.name} node={node} depth={0} last={index === TREE.length - 1} />
      ))}
    </div>
  );
}
