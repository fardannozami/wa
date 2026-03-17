import { useEffect, useState, useRef, useCallback } from 'react'
import { deviceApi } from '../api/client'

export default function Device() {
  const [device, setDevice] = useState(null)
  const [status, setStatus] = useState('disconnected')
  const [phone, setPhone] = useState('')
  const [qrCode, setQrCode] = useState(null)
  const [loading, setLoading] = useState(true)
  const [connecting, setConnecting] = useState(false)
  const wsRef = useRef(null)
  const reconnectTimeoutRef = useRef(null)

  const connectWebSocket = useCallback(() => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      console.log('[WS] Already connected')
      return
    }

    console.log('[WS] Connecting...')
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const token = localStorage.getItem('token')
    const wsUrl = `${protocol}//${window.location.host}/api/v1/device/ws?token=${token}`
    console.log('[WS] URL:', wsUrl)

    const ws = new WebSocket(wsUrl)

    ws.onopen = () => {
      console.log('[WS] Connected')
    }

    ws.onerror = (error) => {
      console.error('[WS] Error:', error)
    }

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        console.log('[WS] Received:', data)

        if (data.type === 'qr') {
          setQrCode({ code: data.code, image: data.image })
          setStatus('qr_generated')
        } else if (data.type === 'connected') {
          setStatus('connected')
          setQrCode(null)
        } else if (data.type === 'failed') {
          setStatus('disconnected')
          setQrCode(null)
        } else if (data.type === 'logged_out') {
          setStatus('disconnected')
          setQrCode(null)
          setPhone('')
        }
      } catch (e) {
        console.error('[WS] Failed to parse message:', e)
      }
    }

    ws.onclose = () => {
      console.log('[WS] Disconnected')
      if (status === 'qr_generated' || status === 'connecting') {
        reconnectTimeoutRef.current = setTimeout(() => {
          connectWebSocket()
        }, 3000)
      }
    }

    wsRef.current = ws
  }, [status])

  const disconnectWebSocket = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
    }
    if (wsRef.current) {
      wsRef.current.close()
      wsRef.current = null
    }
  }, [])

  useEffect(() => {
    loadDevice()
  }, [])

  useEffect(() => {
    if (status === 'qr_generated' || status === 'connecting') {
      connectWebSocket()
    } else if (status === 'connected' || status === 'active') {
      disconnectWebSocket()
    }
  }, [status, connectWebSocket, disconnectWebSocket])

  useEffect(() => {
    return () => {
      disconnectWebSocket()
    }
  }, [disconnectWebSocket])

  const loadDevice = async () => {
    try {
      const { data } = await deviceApi.get()
      setDevice(data.device)
      if (data.device) {
        setStatus(data.device.status || 'disconnected')
        setPhone(data.device.phone_number || '')
      }
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
    connectWebSocket()
    await new Promise(resolve => setTimeout(resolve, 500))
    try {
      await deviceApi.connect()
      setStatus('connecting')
    } catch (e) {
      console.error(e)
      disconnectWebSocket()
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
                <img src={`data:image/png;base64,${qrCode.image}`} alt="QR Code" style={{ width: '100%' }} />
              ) : qrCode.code ? (
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', fontFamily: 'monospace', wordBreak: 'break-all' }}>
                  {qrCode.code}
                </div>
              ) : (
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%' }}>
                  Waiting for QR code...
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
