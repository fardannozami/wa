import { Outlet, NavLink, useNavigate } from 'react-router-dom'
import { authApi } from '../api/client'
import { useAuthStore } from '../hooks/useAuth'

export default function Layout() {
  const navigate = useNavigate()
  const { logout } = useAuthStore()

  const handleLogout = async () => {
    try {
      await authApi.logout()
    } catch (e) {
      console.error(e)
    }
    logout()
    navigate('/login')
  }

  return (
    <div className="app">
      <aside className="sidebar">
        <div className="sidebar-logo">💬 WA SaaS</div>
        <nav>
          <NavLink to="/dashboard" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`}>
            📊 Dashboard
          </NavLink>
          <NavLink to="/device" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`}>
            📱 Device
          </NavLink>
          <NavLink to="/contacts" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`}>
            📇 Contacts
          </NavLink>
          <NavLink to="/campaigns" className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`}>
            📢 Campaigns
          </NavLink>
        </nav>
        <div style={{ marginTop: 'auto', padding: '20px' }}>
          <button onClick={handleLogout} className="btn btn-secondary" style={{ width: '100%' }}>
            🚪 Logout
          </button>
        </div>
      </aside>
      <main className="main-content">
        <Outlet />
      </main>
    </div>
  )
}
