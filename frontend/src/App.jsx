import { useEffect, useState } from 'react'
import { BrowserRouter, Routes, Route, Navigate, useNavigate, useSearchParams } from 'react-router-dom'
import { Toaster } from 'react-hot-toast'
import { useAuthStore } from './hooks/useAuth'
import Layout from './components/Layout'
import Login from './pages/Login'
import Dashboard from './pages/Dashboard'
import Device from './pages/Device'
import Contacts from './pages/Contacts'
import Campaigns from './pages/Campaigns'
import AdminDashboard from './pages/AdminDashboard'

function OAuthCallback() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const { setAuth } = useAuthStore()
  const [error, setError] = useState('')

  useEffect(() => {
    const token = searchParams.get('token')
    if (token) {
      localStorage.setItem('token', token)
      setAuth(true)
      navigate('/dashboard')
    } else {
      setError('No token received')
    }
  }, [searchParams, navigate, setAuth])

  if (error) {
    return (
      <div className="login-page">
        <div className="card login-card">
          <h2 style={{ color: 'red' }}>{error}</h2>
          <button onClick={() => navigate('/login')} className="btn btn-primary" style={{ marginTop: '20px' }}>
            Back to Login
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="login-page">
      <div className="card login-card">
        <h2>Logging you in...</h2>
      </div>
    </div>
  )
}

function PrivateRoute({ children }) {
  const { isAuthenticated, loading } = useAuthStore()
  
  if (loading) {
    return (
      <div className="loading-screen">
        <div className="spinner" />
      </div>
    )
  }
  
  return isAuthenticated ? children : <Navigate to="/login" />
}

function AdminRoute({ children }) {
  const { user, isAuthenticated, loading } = useAuthStore()
  
  if (loading) {
    return (
      <div className="loading-screen">
        <div className="spinner" />
      </div>
    )
  }
  
  if (!isAuthenticated || !user?.is_admin) {
    return <Navigate to="/dashboard" />
  }
  
  return children
}

function App() {
  const { checkAuth, loading } = useAuthStore()

  useEffect(() => {
    checkAuth()
  }, [])

  if (loading) {
    return (
      <div className="loading-screen">
        <div className="spinner" />
      </div>
    )
  }

  return (
    <BrowserRouter>
      <Toaster position="top-center" />
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/oauth/callback" element={<OAuthCallback />} />
        <Route path="/" element={<PrivateRoute><Layout /></PrivateRoute>}>
          <Route index element={<Navigate to="/dashboard" replace />} />
          <Route path="dashboard" element={<Dashboard />} />
          <Route path="device" element={<Device />} />
          <Route path="contacts" element={<Contacts />} />
          <Route path="campaigns" element={<Campaigns />} />
          <Route path="admin" element={<AdminRoute><AdminDashboard /></AdminRoute>} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}

export default App
