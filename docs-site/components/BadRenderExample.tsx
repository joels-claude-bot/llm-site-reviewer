import React from 'react';

// A deliberately badly-rendered page. Its job is to be ugly: this fixture exercises the
// rendering category, where the defect only exists in pixels.

const rows = [
  ['Nice', 'NCE', '£186', '£820', '£174', '7', 'Jul', 'Sunny', 'TGV', '4.2', 'Yes', '£1180'],
  ['Marseille', 'MRS', '£162', '£760', '£140', '7', 'Jul', 'Sunny', 'Metro', '4.0', 'Yes', '£1062'],
  ['Lyon', 'LYS', '£148', '£690', '£120', '7', 'Jul', 'Warm', 'Tram', '4.1', 'No', '£958'],
  ['Bordeaux', 'BOD', '£171', '£710', '£130', '7', 'Jul', 'Warm', 'Tram', '3.9', 'Yes', '£1011'],
  ['Toulouse', 'TLS', '£155', '£680', '£125', '7', 'Jul', 'Warm', 'Metro', '4.0', 'No', '£960'],
  ['Montpellier', 'MPL', '£168', '£700', '£135', '7', 'Jul', 'Sunny', 'Tram', '4.3', 'Yes', '£1003'],
];
const cols = ['City', 'Airport', 'Flight', 'Hotel', 'Transfer', 'Nights', 'Month', 'Climate', 'Local', 'Rating', 'Refundable', 'Total'];

export default function BadRenderExample() {
  return (
    <div
      style={{
        border: '1px solid #d8dee9',
        borderRadius: 10,
        overflow: 'hidden',
        boxShadow: '0 4px 16px rgba(0,0,0,0.08)',
        background: '#ffffff',
        fontFamily: 'system-ui, -apple-system, Segoe UI, Roboto, sans-serif',
        margin: '24px 0',
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '8px 12px', background: '#eef1f5', borderBottom: '1px solid #d8dee9' }}>
        <span style={{ width: 11, height: 11, borderRadius: '50%', background: '#ff5f57' }} />
        <span style={{ width: 11, height: 11, borderRadius: '50%', background: '#febc2e' }} />
        <span style={{ width: 11, height: 11, borderRadius: '50%', background: '#28c840' }} />
        <span style={{ marginLeft: 10, flex: 1, background: '#fff', border: '1px solid #d8dee9', borderRadius: 6, padding: '3px 10px', fontSize: 12, color: '#64748b' }}>
          docs.example.com/trips/compare
        </span>
      </div>

      {/* FLAT_COLOUR + WEAK_HIERARCHY: everything one flat grey, every line the same size and weight */}
      <div style={{ position: 'relative', maxHeight: 260, overflow: 'hidden', background: '#f3f4f6' }}>
        <div style={{ padding: '16px 18px', color: '#6b7280', fontSize: 12, lineHeight: 1.7 }}>
          <div>Destination comparison for July 2026</div>
          <div>We looked at six destinations against flights accommodation transfers climate and refund policy and the figures are below and you should read across each row to compare the total cost which is the last column and pick whichever suits although they are all fairly close together and most are under budget except Nice which is the recommended one despite being the most expensive.</div>

          {/* DIAGRAM_SYNTAX_ERROR: the diagram rendered an error box instead of a graph */}
          <div style={{ margin: '10px 0', border: '1px solid #ef4444', background: '#fef2f2', color: '#b91c1c', borderRadius: 6, padding: '8px 10px', fontFamily: 'ui-monospace, Menlo, monospace', fontSize: 11 }}>
            Error: Parse error on line 2 — expected &apos;SEMI&apos;, &apos;NEWLINE&apos; got &apos;ALPHA&apos;
          </div>

          {/* HIGH_DENSITY + TEXT_LEGIBILITY: 12-column table crammed at 9px */}
          <table style={{ borderCollapse: 'collapse', width: '100%', fontSize: 9, color: '#6b7280' }}>
            <thead>
              <tr>{cols.map((c) => <th key={c} style={{ border: '1px solid #d1d5db', padding: '2px 3px', textAlign: 'left', fontWeight: 400 }}>{c}</th>)}</tr>
            </thead>
            <tbody>
              {rows.map((r, i) => (
                <tr key={i}>{r.map((cell, j) => <td key={j} style={{ border: '1px solid #d1d5db', padding: '2px 3px' }}>{cell}</td>)}</tr>
              ))}
            </tbody>
          </table>

          <div style={{ marginTop: 10 }}>Notes on each destination follow, starting with Nice which has the best weather record and the longest promenade and the most direct flights from London but also the highest hotel prices in July because of</div>
        </div>
        {/* TRUNCATION: hard fade + clip mid-sentence */}
        <div style={{ position: 'absolute', left: 0, right: 0, bottom: 0, height: 48, background: 'linear-gradient(rgba(243,244,246,0), #f3f4f6)' }} />
      </div>
    </div>
  );
}
