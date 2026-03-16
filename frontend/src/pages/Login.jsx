import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { authApi } from '../api/client'
import { useAuthStore } from '../hooks/useAuth'

export default function Login() {
  const navigate = useNavigate()
  const { login } = useAuthStore()
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const handleDemoLogin = async () => {
    setLoading(true)
    setError('')
    try {
      await login()
      navigate('/dashboard')
    } catch (e) {
      setError(e.response?.data?.error || 'Login failed')
    } finally {
      setLoading(false)
    }
  }

  const handleGoogleLogin = async () => {
    setLoading(true)
    setError('')
    try {
      const { data } = await authApi.googleLogin()
      
      if (data.demo) {
        handleDemoLogin()
        return
      }
      
      if (data.url) {
        window.location.href = data.url
      }
    } catch (e) {
      console.error(e)
      setError('Google login failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="login-page">
      <div className="card login-card">
        <div className="login-logo">💬 WA SaaS</div>
        <h1>WhatsApp Blasting Platform</h1>
        <p className="login-subtitle">Connect with your customers via WhatsApp</p>
        
        {error && <div style={{ color: 'red', marginBottom: '10px' }}>{error}</div>}
        
        <button 
          onClick={handleDemoLogin} 
          className="btn btn-primary" 
          style={{ width: '100%', padding: '14px', marginBottom: '10px' }}
          disabled={loading}
        >
          {loading ? 'Loading...' : 'Demo Login'}
        </button>
        
        <button 
          onClick={handleGoogleLogin} 
          className="btn btn-secondary" 
          style={{ width: '100%', padding: '14px' }}
          disabled={loading}
        >
          Sign in with Google
        </button>
      </div>
    </div>
  )
}
