import { useEffect, useState } from 'react'
import { adminApi } from '../api/client'
import toast from 'react-hot-toast'

export default function AdminDashboard() {
  const [stats, setStats] = useState({
    total_users: 0,
    total_messages: 0,
  })
  const [metrics, setMetrics] = useState(null)
  const [users, setUsers] = useState([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadData()
    
    // Poll metrics every 3 seconds
    const interval = setInterval(loadMetrics, 3000)
    return () => clearInterval(interval)
  }, [])

  const loadData = async () => {
    setLoading(true)
    try {
      const [statsRes, usersRes] = await Promise.all([
        adminApi.getStats(),
        adminApi.listUsers(),
      ])
      setStats(statsRes.data)
      setUsers(usersRes.data)
      await loadMetrics()
    } catch (e) {
      console.error(e)
      toast.error('Failed to load admin data')
    } finally {
      setLoading(false)
    }
  }

  const loadMetrics = async () => {
    try {
      const res = await adminApi.getMetrics()
      setMetrics(res.data)
    } catch (e) {
      console.error('Failed to load metrics', e)
    }
  }

  const formatUptime = (seconds) => {
    const d = Math.floor(seconds / (3600 * 24))
    const h = Math.floor((seconds % (3600 * 24)) / 3600)
    const m = Math.floor((seconds % 3600) / 60)
    const s = Math.floor(seconds % 60)
    
    const parts = []
    if (d > 0) parts.push(`${d}d`)
    if (h > 0) parts.push(`${h}h`)
    if (m > 0) parts.push(`${m}m`)
    parts.push(`${s}s`)
    return parts.join(' ')
  }

  return (
    <div>
      <div className="page-header">
        <h1 className="page-title">Admin Dashboard</h1>
        <p style={{ color: 'var(--text-muted)', marginTop: '4px', fontSize: '14px' }}>
          Platform-wide overview and server health
        </p>
      </div>

      <div className="stats-grid">
        <div className="stat-card">
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
            <div>
              <div className="stat-label">Total Users</div>
              <div className="stat-value">{loading ? '—' : stats.total_users.toLocaleString()}</div>
            </div>
            <div style={{ padding: '10px', background: 'rgba(16, 185, 129, 0.1)', borderRadius: 'var(--radius-md)' }}>
              <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="var(--success)" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
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
              <div className="stat-label">Total Messages Sent</div>
              <div className="stat-value">{loading ? '—' : stats.total_messages.toLocaleString()}</div>
            </div>
            <div style={{ padding: '10px', background: 'rgba(139, 92, 246, 0.1)', borderRadius: 'var(--radius-md)' }}>
              <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="#8b5cf6" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <line x1="22" y1="2" x2="11" y2="13" />
                <polygon points="22 2 15 22 11 13 2 9 22 2" />
              </svg>
            </div>
          </div>
        </div>
      </div>

      <div className="stats-grid" style={{ marginTop: '24px' }}>
        <div className="stat-card">
          <div className="stat-label">Server Memory</div>
          {metrics ? (
            <div style={{ marginTop: '12px' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '13px', marginBottom: '8px' }}>
                <span style={{ color: 'var(--text-muted)' }}>Allocated</span>
                <span style={{ fontWeight: '600' }}>{metrics.memory.alloc} MB</span>
              </div>
              <div style={{ height: '6px', background: 'var(--border)', borderRadius: '3px' }}>
                <div style={{ 
                  height: '100%', 
                  width: `${Math.min((metrics.memory.alloc / metrics.memory.sys) * 100, 100)}%`, 
                  background: 'var(--accent)', 
                  borderRadius: '3px',
                  transition: 'width 0.5s ease'
                }} />
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '11px', marginTop: '6px', color: 'var(--text-muted)' }}>
                <span>Total Sys: {metrics.memory.sys} MB</span>
                <span>GCs: {metrics.memory.num_gc}</span>
              </div>
            </div>
          ) : <div style={{ padding: '20px 0', textAlign: 'center' }}>Loading metrics...</div>}
        </div>

        <div className="stat-card">
          <div className="stat-label">System Resources</div>
          {metrics ? (
            <div style={{ marginTop: '12px', display: 'flex', flexDirection: 'column', gap: '12px' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                  <div style={{ width: '32px', height: '32px', borderRadius: '50%', background: 'rgba(6, 182, 212, 0.1)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#06b6d4" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M12 2v20M2 12h20" /></svg>
                  </div>
                  <span style={{ fontSize: '13px', color: 'var(--text-muted)' }}>Goroutines</span>
                </div>
                <span style={{ fontWeight: '600' }}>{metrics.goroutines}</span>
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                  <div style={{ width: '32px', height: '32px', borderRadius: '50%', background: 'rgba(245, 158, 11, 0.1)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#f59e0b" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="4" y="4" width="16" height="16" rx="2" /><path d="M9 9h6v6H9z" /></svg>
                  </div>
                  <span style={{ fontSize: '13px', color: 'var(--text-muted)' }}>Logical CPUs</span>
                </div>
                <span style={{ fontWeight: '600' }}>{metrics.cpus}</span>
              </div>
            </div>
          ) : <div style={{ padding: '20px 0', textAlign: 'center' }}>Loading metrics...</div>}
        </div>

        <div className="stat-card">
          <div className="stat-label">Server Uptime</div>
          {metrics ? (
            <div style={{ marginTop: '12px' }}>
              <div style={{ fontSize: '24px', fontWeight: '700', color: 'var(--accent)' }}>
                {formatUptime(metrics.uptime)}
              </div>
              <div style={{ fontSize: '12px', color: 'var(--text-muted)', marginTop: '8px', display: 'flex', alignItems: 'center', gap: '6px' }}>
                <span style={{ width: '8px', height: '8px', borderRadius: '50%', background: 'var(--success)', display: 'inline-block' }} />
                Server is active
              </div>
            </div>
          ) : <div style={{ padding: '20px 0', textAlign: 'center' }}>Loading metrics...</div>}
        </div>
      </div>

      <div className="card" style={{ marginTop: '24px' }}>
        <h3 style={{ fontSize: '16px', fontWeight: '600', marginBottom: '16px' }}>User Management</h3>
        <div className="table-container">
          <table className="table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Email</th>
                <th>Status</th>
                <th>Created At</th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr>
                  <td colSpan="4" style={{ textAlign: 'center', padding: '40px' }}>Loading users...</td>
                </tr>
              ) : users.length === 0 ? (
                <tr>
                  <td colSpan="4" style={{ textAlign: 'center', padding: '40px' }}>No users found</td>
                </tr>
              ) : (
                users.map((user) => (
                  <tr key={user.id}>
                    <td>
                      <div style={{ fontWeight: '500' }}>{user.name}</div>
                    </td>
                    <td>{user.email}</td>
                    <td>
                      <span className={`status-badge ${user.is_admin ? 'status-connected' : ''}`} style={{ background: user.is_admin ? 'rgba(16, 185, 129, 0.1)' : 'var(--bg-primary)' }}>
                        {user.is_admin ? 'Admin' : 'User'}
                      </span>
                    </td>
                    <td style={{ color: 'var(--text-muted)', fontSize: '13px' }}>
                      {new Date(user.created_at).toLocaleDateString()}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
