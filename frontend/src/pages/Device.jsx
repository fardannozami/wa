import { useEffect, useState } from 'react'
import { deviceApi } from '../api/client'

export default function Device() {
  const [device, setDevice] = useState(null)
  const [status, setStatus] = useState('disconnected')
  const [phone, setPhone] = useState('')
  const [qrCode, setQrCode] = useState(null)
  const [loading, setLoading] = useState(true)
  const [connecting, setConnecting] = useState(false)

  useEffect(() => {
    loadDevice()
    const interval = setInterval(loadStatus, 5000)
    return () => clearInterval(interval)
  }, [])

  const loadDevice = async () => {
    try {
      const { data } = await deviceApi.get()
      setDevice(data.device)
    } catch (e) {
      console.error(e)
    } finally {
      setLoading(false)
    }
  }

  const loadStatus = async () => {
    try {
      const { data } = await deviceApi.getStatus()
      setStatus(data.status || 'disconnected')
      setPhone(data.phone || '')
    } catch (e) {
      console.error(e)
    }
  }

  const handleConnect = async () => {
    setConnecting(true)
    try {
      const { data } = await deviceApi.connect()
      setQrCode(data)
      setStatus(data.status || 'qr_generated')
    } catch (e) {
      console.error(e)
    } finally {
      setConnecting(false)
    }
  }

  const handleDisconnect = async () => {
    try {
      await deviceApi.disconnect()
      setStatus('disconnected')
      setPhone('')
      setQrCode(null)
    } catch (e) {
      console.error(e)
    }
  }

  const isConnected = status === 'connected' || status === 'active'

  return (
    <div>
      <div className="page-header">
        <h1 className="page-title">WhatsApp Device</h1>
      </div>

      <div className="card">
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '20px' }}>
          <div>
            <h3>Connection Status</h3>
            <span className={`status-badge ${isConnected ? 'status-connected' : 'status-disconnected'}`}>
              {status}
            </span>
            {phone && <p style={{ marginTop: '8px' }}>📱 {phone}</p>}
          </div>
          {!isConnected ? (
            <button onClick={handleConnect} disabled={connecting} className="btn btn-primary">
              {connecting ? 'Generating QR...' : 'Connect Device'}
            </button>
          ) : (
            <button onClick={handleDisconnect} className="btn btn-danger">
              Disconnect
            </button>
          )}
        </div>

        {qrCode && !isConnected && (
          <div className="qr-container">
            <p>Scan this QR code with your WhatsApp:</p>
            <div className="qr-image">
              {qrCode.image ? (
                <img src={qrCode.image} alt="QR Code" style={{ width: '100%' }} />
              ) : (
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%' }}>
                  {qrCode.code}
                </div>
              )}
            </div>
            <p style={{ color: '#666', fontSize: '14px' }}>
              QR code expires in 60 seconds. Refresh to get a new one.
            </p>
          </div>
        )}

        {!qrCode && !isConnected && !loading && (
          <div style={{ textAlign: 'center', padding: '40px', color: '#666' }}>
            Click "Connect Device" to generate a QR code for WhatsApp
          </div>
        )}
      </div>

      <div className="card" style={{ marginTop: '20px' }}>
        <h3>Instructions</h3>
        <ol style={{ marginTop: '12px', paddingLeft: '20px', lineHeight: '1.8' }}>
          <li>Click "Connect Device" to generate a QR code</li>
          <li>Open WhatsApp on your phone</li>
          <li>Go to Settings → Linked Devices</li>
          <li>Scan the QR code displayed above</li>
          <li>Wait for the connection to be established</li>
        </ol>
      </div>
    </div>
  )
}
