import { useEffect, useState } from 'react'
import { campaignApi, contactApi, deviceApi, statsApi } from '../api/client'

export default function Dashboard() {
  const [stats, setStats] = useState({
    contacts: 0,
    campaigns: 0,
    sent: 0,
    deviceConnected: false,
  })
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadStats()
  }, [])

  const loadStats = async () => {
    try {
      const [{ data: statsData }, { data: deviceData }] = await Promise.all([
        statsApi.get(),
        deviceApi.getStatus(),
      ])

      setStats({
        contacts: statsData.contacts || 0,
        campaigns: statsData.campaigns || 0,
        sent: statsData.sent || 0,
        deviceConnected: deviceData.status === 'connected',
      })
    } catch (e) {
      console.error(e)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div>
      <div className="page-header">
        <h1 className="page-title">Dashboard</h1>
        <p style={{ color: 'var(--text-muted)', marginTop: '4px', fontSize: '14px' }}>
          Overview of your WhatsApp campaign activity
        </p>
      </div>

      <div className="stats-grid">
        <div className="stat-card">
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
            <div>
              <div className="stat-label">Total Contacts</div>
              <div className="stat-value">{loading ? '—' : stats.contacts.toLocaleString()}</div>
            </div>
            <div style={{ padding: '10px', background: 'rgba(16, 185, 129, 0.1)', borderRadius: 'var(--radius-md)' }}>
              <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="var(--accent)" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
                <circle cx="9" cy="7" r="4" />
                <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
                <path d="M16 3.13a4 4 0 0 1 0 7.75" />
              </svg>
            </div>
          </div>
        </div>
        <div className="stat-card">
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
            <div>
              <div className="stat-label">Total Campaigns</div>
              <div className="stat-value">{loading ? '—' : stats.campaigns.toLocaleString()}</div>
            </div>
            <div style={{ padding: '10px', background: 'rgba(6, 182, 212, 0.1)', borderRadius: 'var(--radius-md)' }}>
              <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="#06b6d4" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5" />
                <path d="M19.07 4.93a10 10 0 0 1 0 14.14" />
                <path d="M15.54 8.46a5 5 0 0 1 0 7.07" />
              </svg>
            </div>
          </div>
        </div>
        <div className="stat-card">
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
            <div>
              <div className="stat-label">Messages Sent</div>
              <div className="stat-value">{loading ? '—' : stats.sent.toLocaleString()}</div>
            </div>
            <div style={{ padding: '10px', background: 'rgba(139, 92, 246, 0.1)', borderRadius: 'var(--radius-md)' }}>
              <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="#8b5cf6" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <line x1="22" y1="2" x2="11" y2="13" />
                <polygon points="22 2 15 22 11 13 2 9 22 2" />
              </svg>
            </div>
          </div>
        </div>
        <div className="stat-card">
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
            <div>
              <div className="stat-label">Device Status</div>
              <div style={{ marginTop: '8px' }}>
                <span className={`status-badge ${stats.deviceConnected ? 'status-connected' : 'status-disconnected'}`}>
                  <span style={{ 
                    width: '8px', 
                    height: '8px', 
                    borderRadius: '50%', 
                    background: stats.deviceConnected ? 'var(--success)' : 'var(--error)',
                    display: 'inline-block',
                    animation: stats.deviceConnected ? 'pulse 2s infinite' : 'none'
                  }} />
                  {stats.deviceConnected ? 'Connected' : 'Disconnected'}
                </span>
              </div>
            </div>
            <div style={{ padding: '10px', background: stats.deviceConnected ? 'rgba(34, 197, 94, 0.1)' : 'rgba(239, 68, 68, 0.1)', borderRadius: 'var(--radius-md)' }}>
              <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke={stats.deviceConnected ? 'var(--success)' : 'var(--error)'} strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <rect x="5" y="2" width="14" height="20" rx="2" ry="2" />
                <line x1="12" y1="18" x2="12.01" y2="18" />
              </svg>
            </div>
          </div>
        </div>
      </div>

      <div className="card">
        <h3 style={{ fontSize: '16px', fontWeight: '600', marginBottom: '16px' }}>🚀 Getting Started</h3>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))', gap: '16px' }}>
          {[
            { step: '01', title: 'Connect Device', desc: 'Scan QR code to link your WhatsApp' },
            { step: '02', title: 'Import Contacts', desc: 'Add contacts manually or import CSV' },
            { step: '03', title: 'Create Campaign', desc: 'Design your message template' },
            { step: '04', title: 'Send & Track', desc: 'Launch your campaign and monitor results' },
          ].map((item) => (
            <div key={item.step} style={{ 
              padding: '16px', 
              background: 'var(--bg-primary)', 
              borderRadius: 'var(--radius-md)',
              border: '1px solid var(--border)',
              display: 'flex',
              gap: '14px',
              alignItems: 'flex-start'
            }}>
              <span style={{ 
                fontSize: '12px', 
                fontWeight: '700', 
                color: 'var(--accent)', 
                background: 'var(--accent-glow)',
                padding: '4px 8px',
                borderRadius: 'var(--radius-sm)',
                letterSpacing: '0.05em',
                flexShrink: 0
              }}>
                {item.step}
              </span>
              <div>
                <div style={{ fontWeight: '600', fontSize: '14px', marginBottom: '2px' }}>{item.title}</div>
                <div style={{ fontSize: '13px', color: 'var(--text-muted)' }}>{item.desc}</div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
