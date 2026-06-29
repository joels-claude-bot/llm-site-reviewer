import React from 'react';

const linkStyle: React.CSSProperties = {
  color: '#2563eb',
  textDecoration: 'underline',
};

function FlightCard() {
  return (
    <div
      style={{
        border: '1px solid #cbd5e1',
        borderRadius: 8,
        background: '#f8fafc',
        overflow: 'hidden',
        marginTop: 8,
      }}
    >
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          gap: 12,
          padding: '10px 12px',
          background: '#e0f2fe',
          color: '#0f172a',
          fontSize: 12,
          fontWeight: 700,
        }}
      >
        <span>Flight option</span>
        <span>Skyscanner result</span>
      </div>
      <div style={{ padding: 12 }}>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr auto 1fr', gap: 10, alignItems: 'center' }}>
          <div>
            <div style={{ fontSize: 18, fontWeight: 800, color: '#0f172a' }}>LHR</div>
            <div style={{ fontSize: 12, color: '#64748b' }}>London Heathrow</div>
          </div>
          <div style={{ color: '#64748b', fontSize: 18 }}>→</div>
          <div style={{ textAlign: 'right' }}>
            <div style={{ fontSize: 18, fontWeight: 800, color: '#0f172a' }}>HND</div>
            <div style={{ fontSize: 12, color: '#64748b' }}>Tokyo Haneda</div>
          </div>
        </div>
        <div
          style={{
            marginTop: 12,
            paddingTop: 10,
            borderTop: '1px solid #e2e8f0',
            display: 'flex',
            justifyContent: 'space-between',
            gap: 12,
            fontSize: 13,
          }}
        >
          <span>
            Outbound: <strong>12 July 2026</strong>
          </span>
          <span>
            Price: <strong>£612</strong>
          </span>
        </div>
      </div>
    </div>
  );
}

export default function ReviewedDocExample() {
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
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          padding: '8px 12px',
          background: '#eef1f5',
          borderBottom: '1px solid #d8dee9',
        }}
      >
        <span style={{ width: 11, height: 11, borderRadius: '50%', background: '#ff5f57' }} />
        <span style={{ width: 11, height: 11, borderRadius: '50%', background: '#febc2e' }} />
        <span style={{ width: 11, height: 11, borderRadius: '50%', background: '#28c840' }} />
        <span
          style={{
            marginLeft: 10,
            flex: 1,
            background: '#fff',
            border: '1px solid #d8dee9',
            borderRadius: 6,
            padding: '3px 10px',
            fontSize: 12,
            color: '#64748b',
          }}
        >
          docs.example.com/guide/travel-agent
        </span>
      </div>

      <div style={{ padding: '24px 28px' }}>
        <h2 style={{ margin: '0 0 4px', fontSize: 22, color: '#0f172a' }}>Travel assistant example</h2>
        <p style={{ fontSize: 13, color: '#64748b', margin: '0 0 18px' }}>
          A short docs page that compiled successfully.
        </p>

        <p style={{ margin: '0 0 14px', lineHeight: 1.6 }}>
          Before using airport codes, read the{' '}
          <a style={linkStyle} onClick={(event) => event.preventDefault()} href="#">
            airport-code guide
          </a>
          .
        </p>

        <p style={{ margin: '0 0 6px', lineHeight: 1.6 }}>The request flow is:</p>
        <pre
          style={{
            margin: '0 0 14px',
            background: '#f1f5f9',
            border: '1px solid #e2e8f0',
            borderRadius: 6,
            padding: '10px 12px',
            fontSize: 12.5,
            fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
            color: '#475569',
            whiteSpace: 'pre',
          }}
        >{`graph LR
  User --> Search Agent
  Search Agent --> Flight API`}</pre>

        <p style={{ margin: '0 0 14px', lineHeight: 1.6 }}>
          Send the request through the GDS, then store the chosen itinerary against the PNR.
        </p>

        <div
          style={{
            border: '1px solid #e2e8f0',
            borderRadius: 8,
            padding: 12,
            background: '#ffffff',
          }}
        >
          <div style={{ fontSize: 12, color: '#64748b', marginBottom: 4 }}>User request</div>
          <div style={{ fontWeight: 700, color: '#0f172a', lineHeight: 1.45 }}>
            Find Skyscanner flights from London to Tokyo in June 2026, under £650.
          </div>
          <FlightCard />
        </div>
      </div>
    </div>
  );
}
