import React from 'react';

const link: React.CSSProperties = { color: '#2563eb', textDecoration: 'underline', cursor: 'pointer' };
const dead = (event: React.MouseEvent) => event.preventDefault();

const card: React.CSSProperties = {
  border: '1px solid #e2e8f0',
  borderRadius: 8,
  background: '#ffffff',
  overflow: 'hidden',
  marginTop: 10,
};
const cardHead: React.CSSProperties = {
  display: 'flex',
  justifyContent: 'space-between',
  gap: 12,
  padding: '8px 12px',
  background: '#e0f2fe',
  color: '#0f172a',
  fontSize: 12,
  fontWeight: 700,
};
const row: React.CSSProperties = {
  display: 'flex',
  justifyContent: 'space-between',
  gap: 12,
  fontSize: 13,
  padding: '4px 0',
};
const codeBlock: React.CSSProperties = {
  margin: '0 0 16px',
  background: '#f1f5f9',
  border: '1px solid #e2e8f0',
  borderRadius: 6,
  padding: '10px 12px',
  fontSize: 12.5,
  fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
  color: '#475569',
  whiteSpace: 'pre',
};

function FlightCard() {
  return (
    <div style={card}>
      <div style={cardHead}>
        <span>Flights</span>
        <span>via an OTA</span>
      </div>
      <div style={{ padding: 12 }}>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr auto 1fr', gap: 10, alignItems: 'center' }}>
          <div>
            <div style={{ fontSize: 18, fontWeight: 800, color: '#0f172a' }}>LHR</div>
            <div style={{ fontSize: 12, color: '#64748b' }}>London Heathrow</div>
          </div>
          <div style={{ color: '#64748b', fontSize: 18 }}>→</div>
          <div style={{ textAlign: 'right' }}>
            <div style={{ fontSize: 18, fontWeight: 800, color: '#0f172a' }}>NCE</div>
            <div style={{ fontSize: 12, color: '#64748b' }}>Nice Côte d&apos;Azur</div>
          </div>
        </div>
        <div style={{ marginTop: 10, paddingTop: 8, borderTop: '1px solid #e2e8f0' }}>
          <div style={row}><span>Outbound</span><strong>Mon 3 Aug 2026</strong></div>
          <div style={row}><span>Return</span><strong>Mon 10 Aug 2026</strong></div>
          <div style={row}><span>Return fare, 1 adult</span><strong>£186</strong></div>
        </div>
      </div>
    </div>
  );
}

function StayCard() {
  return (
    <div style={card}>
      <div style={cardHead}>
        <span>Accommodation</span>
        <span>Hôtel du Soleil, Nice</span>
      </div>
      <div style={{ padding: 12 }}>
        <div style={row}><span>Check-in</span><strong>Sat 1 Aug 2026</strong></div>
        <div style={row}><span>Check-out</span><strong>Sat 8 Aug 2026</strong></div>
        <div style={row}><span>Room</span><strong>Twin · 2 guests</strong></div>
        <div style={row}><span>7 nights</span><strong>£820</strong></div>
      </div>
    </div>
  );
}

export default function HolidayDocExample() {
  return (
    <div
      style={{
        border: '1px solid #d8dee9',
        borderRadius: 10,
        overflow: 'hidden',
        boxShadow: '0 4px 16px rgba(0,0,0,0.08)',
        background: '#ffffff',
        color: '#1e293b',
        fontFamily: 'system-ui, -apple-system, Segoe UI, Roboto, sans-serif',
        margin: '24px 0',
      }}
    >
      {/* browser chrome */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '8px 12px', background: '#eef1f5', borderBottom: '1px solid #d8dee9' }}>
        <span style={{ width: 11, height: 11, borderRadius: '50%', background: '#ff5f57' }} />
        <span style={{ width: 11, height: 11, borderRadius: '50%', background: '#febc2e' }} />
        <span style={{ width: 11, height: 11, borderRadius: '50%', background: '#28c840' }} />
        <span style={{ marginLeft: 10, flex: 1, background: '#fff', border: '1px solid #d8dee9', borderRadius: 6, padding: '3px 10px', fontSize: 12, color: '#64748b' }}>
          docs.example.com/trips/france-july-2026
        </span>
      </div>

      <div style={{ padding: '24px 28px' }}>
        <h2 style={{ margin: '0 0 4px', fontSize: 22, color: '#0f172a' }}>France holiday plan</h2>
        {/* MISSING_WHAT_WHY: dives straight into output without saying what this page is or who it is for */}
        <p style={{ fontSize: 13, color: '#64748b', margin: '0 0 16px' }}>Generated itinerary — compiled and published.</p>

        {/* the driving request */}
        <div style={{ border: '1px solid #e2e8f0', borderRadius: 8, padding: 12, background: '#f8fafc', marginBottom: 16 }}>
          <div style={{ fontSize: 12, color: '#64748b', marginBottom: 4 }}>Request</div>
          <div style={{ fontWeight: 700, color: '#0f172a', lineHeight: 1.45 }}>
            Find flights, accommodation, and a transfer for a holiday to France in <strong>July 2026</strong>,
            for <strong>1 person</strong>, under <strong>£1,000</strong>. Prefer a sunny destination.
          </div>
        </div>

        {/* destination + UNVERIFIABLE claim + INCONSISTENT_TERM (Riviera / Côte d'Azur / south coast) */}
        <p style={{ margin: '0 0 8px', lineHeight: 1.6 }}>
          Recommended destination: <strong>Nice</strong>, on the <strong>Riviera</strong> — the
          sunniest city in Europe, and an easy hop from London.
        </p>

        {/* IMAGE_MISMATCH: map labelled Paris under a Nice recommendation; and MISSING_IMAGE below */}
        <div style={{ display: 'flex', gap: 12, marginBottom: 16 }}>
          <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 8, height: 110, border: '1px dashed #cbd5e1', borderRadius: 8, background: '#f1f5f9', color: '#94a3b8', fontSize: 13 }}>
            <span style={{ fontSize: 22 }}>🖼️</span><span>nice-promenade.jpg</span>
          </div>
          <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', height: 110, border: '1px solid #e2e8f0', borderRadius: 8, background: '#eef2ff', color: '#475569', fontSize: 13 }}>
            📍 Map of <strong style={{ margin: '0 4px' }}>Paris</strong>
          </div>
        </div>

        {/* RENDERED_BROKEN: raw mermaid source instead of a diagram */}
        <p style={{ margin: '0 0 6px', lineHeight: 1.6 }}>Your trip at a glance:</p>
        <pre style={codeBlock}>{`graph LR
  Home --> LHR
  LHR --> Nice
  Nice --> Hotel`}</pre>

        {/* cards with date / pax / budget mismatches */}
        <FlightCard />
        <StayCard />

        {/* SIMPLER: a dense run-on of options that should be a table */}
        <p style={{ margin: '16px 0 8px', lineHeight: 1.6 }}>
          For the room you could take the twin at £820, or the sea-view double at £910, or the budget
          single at £640 which has no breakfast, or a hostel dorm at £210 though that is shared, or an
          apartment at £990 with a kitchen but a three-night minimum and a cleaning fee on top, and
          there is also a half-board option at £1,040 that includes dinner.
        </p>

        {/* transfer + dead internal link + broken anchor + soft-404 + ASSUMED_JARGON + acronyms; EHIC look-alike */}
        <p style={{ margin: '0 0 8px', lineHeight: 1.6 }}>
          From the airport, take the <strong>TGV</strong> into the city, or just bagsy a fixed-fare
          transfer off the rank. See our{' '}
          <a style={link} onClick={dead} href="#">airport transfer guide</a> for taxi options (around
          £174), the <a style={link} onClick={dead} href="#">packing checklist</a> further down, and
          live conditions on the <a style={link} onClick={dead} href="#">weather page</a>. Check the{' '}
          <strong>FCDO</strong> advisory before you travel, book through an{' '}
          <strong>OTA</strong> rather than the airline, and bring your{' '}
          <strong>EHIC (European Health Insurance Card)</strong> for medical cover.
        </p>

        {/* BROKEN_EXTERNAL_LINK look + WRONG_CLAIM budget */}
        <p style={{ margin: '0 0 8px', lineHeight: 1.6 }}>
          Lock it in on{' '}
          <a style={link} onClick={dead} href="#">partner-bookings.example/holiday/nce-2026</a>.{' '}
          <strong style={{ color: '#0f172a' }}>Total comes to £1,180 — comfortably under your £1,000 budget.</strong>
        </p>

        {/* look-alike: example.com placeholder + ntfy.sh placeholder in code */}
        <p style={{ margin: '0 0 6px', lineHeight: 1.6 }}>To rebook or watch the price:</p>
        <pre style={codeBlock}>{`curl https://example.com/book?dest=NCE
curl -d "price drop" ntfy.sh/your-trip-alerts`}</pre>

        {/* look-alike: explicit stub */}
        <h3 style={{ margin: '0 0 4px', fontSize: 15, color: '#0f172a' }}>Winter alternatives</h3>
        <p style={{ margin: '0 0 16px', lineHeight: 1.6, color: '#94a3b8', fontStyle: 'italic' }}>TODO: add winter options.</p>

        {/* BREAKS_STANDARD: footer claims a standard the page violates */}
        <p style={{ margin: 0, paddingTop: 10, borderTop: '1px solid #e2e8f0', fontSize: 12, color: '#64748b' }}>
          All acronyms on this page are expanded on first use, per our documentation standard.
        </p>
      </div>
    </div>
  );
}
