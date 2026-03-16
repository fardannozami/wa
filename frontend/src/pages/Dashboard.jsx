import { useEffect, useState } from 'react'
import { campaignApi, contactApi, deviceApi } from '../api/client'

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
      const [contactsRes, campaignsRes, deviceRes] = await Promise.all([
        contactApi.list(1, 1),
        campaignApi.list(1, 1),
        deviceApi.getStatus(),
      ])

      setStats({
        contacts: contactsRes.data.total || 0,
        campaigns: campaignsRes.data.total || 0,
        sent: campaignsRes.data.data?.reduce((sum, c) => sum + (c.success_count || 0), 0) || 0,
        deviceConnected: deviceRes.data.status === 'connected',
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
      </div>

      <div className="stats-grid">
        <div className="stat-card">
          <div className="stat-label">Total Contacts</div>
          <div className="stat-value">{loading ? '...' : stats.contacts}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">Total Campaigns</div>
          <div className="stat-value">{loading ? '...' : stats.campaigns}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">Messages Sent</div>
          <div className="stat-value">{loading ? '...' : stats.sent}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">Device Status</div>
          <div className="stat-value" style={{ fontSize: '16px' }}>
            <span className={`status-badge ${stats.deviceConnected ? 'status-connected' : 'status-disconnected'}`}>
              {stats.deviceConnected ? 'Connected' : 'Disconnected'}
            </span>
          </div>
        </div>
      </div>

      <div className="card">
        <h3>Getting Started</h3>
        <ol style={{ marginTop: '12px', paddingLeft: '20px', lineHeight: '1.8' }}>
          <li>Connect your WhatsApp device by scanning the QR code</li>
          <li>Import your contacts or add them manually</li>
          <li>Create a campaign with your message template</li>
          <li>Send your first blast!</li>
        </ol>
      </div>
    </div>
  )
}
