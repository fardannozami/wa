import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '../hooks/useAuth'

export default function Login() {
  const navigate = useNavigate()
  const { login } = useAuthStore()
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const handleGoogleLogin = async () => {
    setLoading(true)
    setError('')
    
    try {
      const response = await fetch('/api/v1/auth/google', {
        method: 'GET',
        credentials: 'include'
      })
      
      const data = await response.json()
      console.log('Response:', data)
      
      if (data.url) {
        console.log('Redirecting to:', data.url)
        window.location.href = data.url
      } else if (data.error) {
        setError(data.error)
      }
    } catch (e) {
      console.error('Error:', e)
      setError('Google login failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="login-page">
      <div className="card login-card">
        <div className="login-logo">Blastr</div>
        <h1 style={{ fontSize: '20px', fontWeight: '600', marginBottom: '6px', color: 'var(--text-primary)' }}>
          WhatsApp Campaign Platform
        </h1>
        <p className="login-subtitle">
          Connect, engage, and grow your audience<br />with intelligent message campaigns
        </p>
        
        {error && (
          <div style={{ 
            color: 'var(--error)', 
            marginBottom: '16px', 
            padding: '10px 14px',
            background: 'var(--error-bg)',
            borderRadius: 'var(--radius-sm)',
            fontSize: '13px',
            border: '1px solid rgba(239, 68, 68, 0.2)'
          }}>
            {error}
          </div>
        )}
        
        <button 
          onClick={handleGoogleLogin} 
          className="google-btn"
          disabled={loading}
        >
          <svg width="20" height="20" viewBox="0 0 24 24">
            <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z" fill="#4285F4"/>
            <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
            <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05"/>
            <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
          </svg>
          {loading ? 'Connecting...' : 'Sign in with Google'}
        </button>

        <p style={{ marginTop: '24px', fontSize: '12px', color: 'var(--text-muted)' }}>
          By signing in, you agree to our Terms of Service
        </p>
      </div>
    </div>
  )
}
