import { useEffect, useState, useRef, useCallback } from 'react'
import toast from 'react-hot-toast'
import { campaignApi, contactApi, deviceApi, groupApi } from '../api/client'

export default function Campaigns() {
  const [campaigns, setCampaigns] = useState([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const [showModal, setShowModal] = useState(false)
  const [editingCampaign, setEditingCampaign] = useState(null)
  const [showSendModal, setShowSendModal] = useState(false)
  const [showDetailModal, setShowDetailModal] = useState(false)
  const [detailCampaign, setDetailCampaign] = useState(null)
  const [messages, setMessages] = useState([])
  const [loadingMessages, setLoadingMessages] = useState(false)
  const [selectedCampaign, setSelectedCampaign] = useState(null)
  const [sendType, setSendType] = useState('now')
  const [scheduleDate, setScheduleDate] = useState('')
  const [scheduleTime, setScheduleTime] = useState('')
  const [sending, setSending] = useState(false)
  const [deviceStatus, setDeviceStatus] = useState('disconnected')
  const [contacts, setContacts] = useState([])
  const [selectedContacts, setSelectedContacts] = useState([])
  const [formData, setFormData] = useState({ name: '', template: '' })
  const [groups, setGroups] = useState([])
  const [selectedGroupId, setSelectedGroupId] = useState('')
  const [selectMode, setSelectMode] = useState('group')
  const [loadingContacts, setLoadingContacts] = useState(false)
  const wsRef = useRef(null)
  const templateRef = useRef(null)

  const connectWebSocket = useCallback(() => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      return
    }

    try {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const token = localStorage.getItem('token')
      const wsUrl = `${protocol}//${window.location.host}/api/v1/device/ws?token=${token}`

      const ws = new WebSocket(wsUrl)

      ws.onopen = () => {}

      ws.onerror = (error) => {
        console.error('[Campaigns WS] Error:', error)
      }

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data)
          if (data.type === 'campaign_update') {
            setCampaigns(prev => prev.map(c => 
              c.id === data.campaign_id 
                ? { ...c, status: data.status, success_count: data.success_count, failed_count: data.failed_count }
                : c
            ))
          }
        } catch (e) {
          console.error('WS parse error:', e)
        }
      }

      ws.onclose = () => {
        wsRef.current = null
        setTimeout(() => {
          if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
            connectWebSocket()
          }
        }, 3000)
      }

      wsRef.current = ws
    } catch (e) {
      console.error('[Campaigns WS] Connection error:', e)
    }
  }, [])

  useEffect(() => {
    loadCampaigns()
    loadDeviceStatus()
    connectWebSocket()

    return () => {
      if (wsRef.current) {
        wsRef.current.close()
      }
    }
  }, [page, connectWebSocket])

  const loadDeviceStatus = async () => {
    try {
      const { data } = await deviceApi.get()
      if (data.device) {
        setDeviceStatus(data.device.status || 'disconnected')
      }
    } catch (e) {
      console.error(e)
    }
  }

  const loadCampaigns = async () => {
    setLoading(true)
    try {
      const { data } = await campaignApi.list(page, 20)
      setCampaigns(data.data || [])
      setTotal(data.total || 0)
    } catch (e) {
      console.error(e)
    } finally {
      setLoading(false)
    }
  }

  const openSendModal = (campaign) => {
    setSelectedCampaign(campaign)
    setSendType('now')
    setScheduleDate('')
    setScheduleTime('')
    setShowSendModal(true)
  }

  const handleSend = async () => {
    if (!selectedCampaign) return

    let scheduledAt = null
    if (sendType === 'schedule') {
      if (!scheduleDate || !scheduleTime) {
        toast.error('Please select date and time')
        return
      }
      scheduledAt = new Date(`${scheduleDate}T${scheduleTime}`).toISOString()
    }

    setSending(true)
    try {
      await campaignApi.send(selectedCampaign.id, { scheduled_at: scheduledAt })
      setShowSendModal(false)
      toast.success(scheduledAt ? 'Campaign scheduled!' : 'Campaign started!')
      loadCampaigns()
    } catch (e) {
      console.error(e)
      toast.error('Failed to start campaign: ' + e.response?.data?.error)
    } finally {
      setSending(false)
    }
  }

  const loadContacts = async (groupId) => {
    const targetGroupId = groupId !== undefined ? groupId : selectedGroupId
    if (loadingContacts) return
    setLoadingContacts(true)
    try {
      const params = new URLSearchParams()
      params.append('page', 1)
      params.append('limit', 500)
      if (targetGroupId) {
        params.append('group_id', targetGroupId)
      }
      console.log('Loading contacts with params:', params.toString())
      const { data } = await contactApi.list(1, 500, params.toString())
      setContacts(data.data || [])
      
      if (selectMode === 'group') {
        setSelectedContacts((data.data || []).map(c => c.id))
      } else {
        setSelectedContacts(prev => {
          const currentIds = new Set((data.data || []).map(c => c.id))
          return prev.filter(id => !currentIds.has(id))
        })
      }
    } catch (e) {
      console.error(e)
    } finally {
      setLoadingContacts(false)
    }
  }

  const loadGroups = async () => {
    try {
      const { data } = await groupApi.list()
      setGroups(data.data || [])
    } catch (e) {
      console.error(e)
    }
  }

  const handleSelectModeChange = async (mode) => {
    setSelectMode(mode)
    if (mode === 'group') {
      if (selectedGroupId) {
        await loadContacts(selectedGroupId)
      }
    } else {
      setSelectedContacts([])
      await loadContacts('')
    }
  }

  const handleGroupChange = async (groupId) => {
    setSelectedGroupId(groupId)
    if (selectMode === 'group' && groupId) {
      await loadContacts(groupId)
    }
  }

  const insertPlaceholder = (tag) => {
    const textarea = templateRef.current
    if (!textarea) return

    const start = textarea.selectionStart
    const end = textarea.selectionEnd
    const text = formData.template
    const before = text.substring(0, start)
    const after = text.substring(end)
    const newText = before + tag + after

    setFormData({ ...formData, template: newText })
    
    // Focus back and set cursor after the inserted tag
    setTimeout(() => {
      textarea.focus()
      const newPos = start + tag.length
      textarea.setSelectionRange(newPos, newPos)
    }, 0)
  }

  const openModal = async () => {
    await loadGroups()
    setEditingCampaign(null)
    setFormData({ name: '', template: '' })
    setSelectedContacts([])
    setSelectedGroupId('')
    setSelectMode('group')
    setShowModal(true)
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    
    try {
      if (editingCampaign) {
        await campaignApi.update(editingCampaign.id, {
          ...formData,
          contact_ids: selectedContacts,
        })
      } else {
        await campaignApi.create({
          ...formData,
          contact_ids: selectedContacts,
        })
      }
      setShowModal(false)
      setEditingCampaign(null)
      setFormData({ name: '', template: '' })
      setSelectedContacts([])
      toast.success(editingCampaign ? 'Campaign updated' : 'Campaign created')
      loadCampaigns()
    } catch (e) {
      console.error(e)
    }
  }

  const handleEdit = async (campaign) => {
    try {
      await loadGroups()
      setEditingCampaign(campaign)
      setFormData({ name: campaign.name, template: campaign.template || '' })
      setSelectedContacts([])
      setSelectedGroupId('')
      setSelectMode('manual')
      
      const { data } = await campaignApi.get(campaign.id)
      setSelectedContacts(data.contact_ids || [])
      
      setShowModal(true)
    } catch (e) {
      console.error(e)
      toast.error('Failed to load campaign')
    }
  }

  const openDetailModal = async (campaign) => {
    setDetailCampaign(campaign)
    setShowDetailModal(true)
    setLoadingMessages(true)
    try {
      const { data } = await campaignApi.getMessages(campaign.id)
      setMessages(data.data || [])
    } catch (e) {
      console.error(e)
      toast.error('Failed to load messages')
    } finally {
      setLoadingMessages(false)
    }
  }

  const handleResend = async (messageId) => {
    try {
      await campaignApi.resendMessage(messageId)
      toast.success('Message resent')
      openDetailModal(detailCampaign)
    } catch (e) {
      console.error(e)
      toast.error('Failed to resend message')
    }
  }

  const handleDelete = async (id) => {
    toast((t) => (
      <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
        <span>Delete this campaign?</span>
        <div style={{ display: 'flex', gap: '5px' }}>
          <button 
            onClick={() => { toast.dismiss(t.id); deleteCampaign(id) }}
            style={{ background: '#dc2626', color: 'white', border: 'none', padding: '5px 10px', borderRadius: '4px', cursor: 'pointer' }}
          >
            Delete
          </button>
          <button 
            onClick={() => toast.dismiss(t.id)}
            style={{ background: '#6b7280', color: 'white', border: 'none', padding: '5px 10px', borderRadius: '4px', cursor: 'pointer' }}
          >
            Cancel
          </button>
        </div>
      </div>
    ), { duration: 5000 })
  }

  const deleteCampaign = async (id) => {
    try {
      await campaignApi.delete(id)
      toast.success('Campaign deleted')
      loadCampaigns()
    } catch (e) {
      console.error(e)
      toast.error('Failed to delete campaign')
    }
  }

  const toggleSelectAll = () => {
    if (selectedContacts.length === contacts.length) {
      setSelectedContacts([])
    } else {
      setSelectedContacts(contacts.map(c => c.id))
    }
  }

  const getStatusBadge = (status) => {
    const statusMap = {
      draft: 'status-disconnected',
      scheduled: 'status-running',
      running: 'status-running',
      completed: 'status-completed',
      cancelled: 'status-disconnected',
      failed: 'status-disconnected',
    }
    return statusMap[status] || 'status-disconnected'
  }

  return (
    <div>
      <div className="page-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h1 className="page-title">Campaigns</h1>
        <button onClick={openModal} className="btn btn-primary">
          + Create Campaign
        </button>
      </div>

      <div className="card">
        {loading ? (
          <div style={{ textAlign: 'center', padding: '40px' }}>Loading...</div>
        ) : campaigns.length === 0 ? (
          <div style={{ textAlign: 'center', padding: '40px', color: '#666' }}>
            No campaigns yet. Create your first campaign to start sending messages.
          </div>
        ) : (
          <>
            <table className="table">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Status</th>
                  <th>Total</th>
                  <th>Success</th>
                  <th>Failed</th>
                  <th>Created</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {campaigns.map((campaign) => (
                  <tr key={campaign.id}>
                    <td>{campaign.name}</td>
                    <td>
                      <span className={`status-badge ${getStatusBadge(campaign.status)}`}>
                        {campaign.status}
                      </span>
                    </td>
                    <td>{campaign.total_count || 0}</td>
                    <td>{campaign.success_count || 0}</td>
                    <td>{campaign.failed_count || 0}</td>
                    <td>{new Date(campaign.created_at).toLocaleDateString()}</td>
                    <td>
                      <button 
                        onClick={() => openDetailModal(campaign)} 
                        className="btn btn-secondary" 
                        style={{ padding: '6px 12px', fontSize: '12px', marginRight: '8px' }}
                      >
                        View
                      </button>
                      {campaign.status === 'draft' && (
                        <>
                          <button 
                            onClick={() => openSendModal(campaign)} 
                            className="btn btn-primary" 
                            style={{ padding: '6px 12px', fontSize: '12px', marginRight: '8px' }}
                            disabled={deviceStatus !== 'connected' && deviceStatus !== 'active'}
                          >
                            Send
                          </button>
                          <button 
                            onClick={() => handleEdit(campaign)} 
                            className="btn btn-secondary" 
                            style={{ padding: '6px 12px', fontSize: '12px', marginRight: '8px' }}
                          >
                            Edit
                          </button>
                        </>
                      )}
                      {campaign.status === 'scheduled' && (
                        <>
                          <button 
                            onClick={() => openSendModal(campaign)} 
                            className="btn btn-primary" 
                            style={{ padding: '6px 12px', fontSize: '12px', marginRight: '8px' }}
                            disabled={deviceStatus !== 'connected' && deviceStatus !== 'active'}
                          >
                            Run Now
                          </button>
                        </>
                      )}
                      {(campaign.status !== 'draft' && campaign.status !== 'scheduled') && (
                        <button onClick={() => handleDelete(campaign.id)} className="btn btn-danger" style={{ padding: '6px 12px', fontSize: '12px' }}>
                          Delete
                        </button>
                      )}
                      {(campaign.status === 'draft' || campaign.status === 'scheduled') && (
                        <button onClick={() => handleDelete(campaign.id)} className="btn btn-danger" style={{ padding: '6px 12px', fontSize: '12px' }}>
                          Delete
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            <div style={{ marginTop: '20px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <span>Total: {total} campaigns</span>
              <div>
                <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page === 1} className="btn btn-secondary">
                  Previous
                </button>
                <span style={{ margin: '0 10px' }}>Page {page}</span>
                <button onClick={() => setPage(p => p + 1)} disabled={page * 20 >= total} className="btn btn-secondary">
                  Next
                </button>
              </div>
            </div>
          </>
        )}
      </div>

      {showModal && (
        <div className="modal-overlay" onClick={() => { setShowModal(false); setEditingCampaign(null) }}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3 className="modal-title">{editingCampaign ? 'Edit Campaign' : 'Create Campaign'}</h3>
              <button onClick={() => setShowModal(false)} style={{ background: 'none', border: 'none', fontSize: '20px', cursor: 'pointer' }}>×</button>
            </div>
            <form onSubmit={handleSubmit}>
              <div className="modal-body">
                <div className="form-group">
                  <label className="form-label">Campaign Name</label>
                  <input
                    type="text"
                    className="form-input"
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    required
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">Message Template</label>
                  <div style={{ display: 'flex', gap: '8px', marginBottom: '8px' }}>
                    <button
                      type="button"
                      onClick={() => insertPlaceholder('{{prefix}}')}
                      style={{
                        padding: '4px 8px',
                        fontSize: '12px',
                        background: '#f3f4f6',
                        border: '1px solid #d1d5db',
                        borderRadius: '4px',
                        cursor: 'pointer',
                        color: '#374151'
                      }}
                    >
                      + prefix
                    </button>
                    <button
                      type="button"
                      onClick={() => insertPlaceholder('{{name}}')}
                      style={{
                        padding: '4px 8px',
                        fontSize: '12px',
                        background: '#f3f4f6',
                        border: '1px solid #d1d5db',
                        borderRadius: '4px',
                        cursor: 'pointer',
                        color: '#374151'
                      }}
                    >
                      + name
                    </button>
                  </div>
                  <textarea
                    ref={templateRef}
                    className="form-input"
                    rows={4}
                    value={formData.template}
                    onChange={(e) => setFormData({ ...formData, template: e.target.value })}
                    placeholder="Hello {{prefix}} {{name}}, this is your message..."
                    required
                  />
                  <small style={{ color: '#666' }}>Use {"{{prefix}}"} and {"{{name}}"} to personalize messages</small>
                </div>
                <div className="form-group">
                  <label className="form-label">Select Contacts</label>
                  <div style={{ display: 'flex', gap: '20px', marginBottom: '15px' }}>
                    <label style={{ display: 'flex', alignItems: 'center', gap: '8px', cursor: 'pointer' }}>
                      <input
                        type="radio"
                        name="selectMode"
                        checked={selectMode === 'group'}
                        onChange={() => handleSelectModeChange('group')}
                      />
                      By Group
                    </label>
                    <label style={{ display: 'flex', alignItems: 'center', gap: '8px', cursor: 'pointer' }}>
                      <input
                        type="radio"
                        name="selectMode"
                        checked={selectMode === 'manual'}
                        onChange={() => handleSelectModeChange('manual')}
                      />
                      Manual Select
                    </label>
                  </div>

                  {selectMode === 'group' && (
                    <div style={{ marginBottom: '15px' }}>
                      <label className="form-label">Select Group</label>
                      <select
                        className="form-input"
                        value={selectedGroupId}
                        onChange={(e) => handleGroupChange(e.target.value)}
                      >
                        <option value="">-- Select Group --</option>
                        {groups.map(group => (
                          <option key={group.id} value={group.id}>{group.name}</option>
                        ))}
                      </select>
                      {selectedGroupId && (
                        <small style={{ color: '#666', display: 'block', marginTop: '5px' }}>
                          {contacts.length} contacts in this group
                        </small>
                      )}
                    </div>
                  )}

                  {selectMode === 'manual' && (
                    <>
                      <div style={{ marginBottom: '8px' }}>
                        <button
                          type="button"
                          onClick={toggleSelectAll}
                          style={{
                            padding: '4px 10px',
                            fontSize: '12px',
                            border: '1px solid #d1d5db',
                            borderRadius: '4px',
                            background: 'white',
                            cursor: 'pointer'
                          }}
                        >
                          {selectedContacts.length === contacts.length ? 'Deselect All' : 'Select All'}
                        </button>
                      </div>
                      <div style={{ maxHeight: '200px', overflowY: 'auto', border: '1px solid #e0e0e0', borderRadius: '6px', padding: '10px' }}>
                        {contacts.map((contact) => (
                          <label key={contact.id} style={{ display: 'flex', alignItems: 'center', gap: '8px', padding: '6px 0' }}>
                            <input
                              type="checkbox"
                              checked={selectedContacts.includes(contact.id)}
                              onChange={(e) => {
                                if (e.target.checked) {
                                  setSelectedContacts([...selectedContacts, contact.id])
                                } else {
                                  setSelectedContacts(selectedContacts.filter(id => id !== contact.id))
                                }
                              }}
                            />
                            {contact.name} ({contact.phone})
                          </label>
                        ))}
                      </div>
                    </>
                  )}
                  <small style={{ color: '#666' }}>{selectedContacts.length} contacts selected</small>
                </div>
              </div>
              <div className="modal-footer">
                <button type="button" onClick={() => setShowModal(false)} className="btn btn-secondary">Cancel</button>
                <button type="submit" className="btn btn-primary" disabled={selectedContacts.length === 0}>
                  {editingCampaign ? 'Update Campaign' : 'Create Campaign'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {showSendModal && selectedCampaign && (
        <div className="modal-overlay" onClick={() => setShowSendModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3 className="modal-title">Send Campaign: {selectedCampaign.name}</h3>
              <button onClick={() => setShowSendModal(false)} style={{ background: 'none', border: 'none', fontSize: '20px', cursor: 'pointer' }}>×</button>
            </div>
            <div className="modal-body">
              <div className="form-group">
                <label className="form-label">Send Option</label>
                <div style={{ display: 'flex', gap: '20px', marginTop: '10px' }}>
                  <label style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                    <input
                      type="radio"
                      name="sendType"
                      checked={sendType === 'now'}
                      onChange={() => setSendType('now')}
                    />
                    Send Now
                  </label>
                  <label style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                    <input
                      type="radio"
                      name="sendType"
                      checked={sendType === 'schedule'}
                      onChange={() => setSendType('schedule')}
                    />
                    Schedule
                  </label>
                </div>
              </div>

              {sendType === 'schedule' && (
                <div style={{ display: 'flex', gap: '10px', marginTop: '15px' }}>
                  <div className="form-group" style={{ flex: 1 }}>
                    <label className="form-label">Date</label>
                    <input
                      type="date"
                      className="form-input"
                      value={scheduleDate}
                      onChange={(e) => setScheduleDate(e.target.value)}
                      min={new Date().toISOString().split('T')[0]}
                    />
                  </div>
                  <div className="form-group" style={{ flex: 1 }}>
                    <label className="form-label">Time</label>
                    <input
                      type="time"
                      className="form-input"
                      value={scheduleTime}
                      onChange={(e) => setScheduleTime(e.target.value)}
                    />
                  </div>
                </div>
              )}

              {sendType === 'schedule' && scheduleDate && scheduleTime && (
                <p style={{ marginTop: '10px', color: '#666' }}>
                  Will be sent on: {new Date(`${scheduleDate}T${scheduleTime}`).toLocaleString()}
                </p>
              )}
            </div>
            <div className="modal-footer">
              <button type="button" onClick={() => setShowSendModal(false)} className="btn btn-secondary">Cancel</button>
              <button onClick={handleSend} className="btn btn-primary" disabled={sending}>
                {sending ? 'Sending...' : (sendType === 'now' ? 'Send Now' : 'Schedule')}
              </button>
            </div>
          </div>
        </div>
      )}

      {showDetailModal && detailCampaign && (
        <div className="modal-overlay" onClick={() => setShowDetailModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()} style={{ maxWidth: '600px' }}>
            <div className="modal-header">
              <h3 className="modal-title">Campaign: {detailCampaign.name}</h3>
              <button onClick={() => setShowDetailModal(false)} style={{ background: 'none', border: 'none', fontSize: '20px', cursor: 'pointer' }}>×</button>
            </div>
            <div className="modal-body">
              <div style={{ marginBottom: '15px' }}>
                <strong>Status:</strong> {detailCampaign.status}<br />
                <strong>Total:</strong> {detailCampaign.total_count || 0} | 
                <strong> Success:</strong> {detailCampaign.success_count || 0} | 
                <strong> Failed:</strong> {detailCampaign.failed_count || 0}
              </div>
              
              {loadingMessages ? (
                <div style={{ textAlign: 'center', padding: '20px' }}>Loading messages...</div>
              ) : messages.length === 0 ? (
                <div style={{ textAlign: 'center', padding: '20px', color: '#666' }}>No messages</div>
              ) : (
                <div style={{ maxHeight: '400px', overflowY: 'auto' }}>
                  <table className="table">
                    <thead>
                      <tr>
                        <th>Phone</th>
                        <th>Message</th>
                        <th>Status</th>
                        <th>Action</th>
                      </tr>
                    </thead>
                    <tbody>
                      {messages.map((msg) => (
                        <tr key={msg.id}>
                          <td>{msg.phone}</td>
                          <td style={{ maxWidth: '150px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                            {msg.message}
                          </td>
                          <td>
                            <span className={`status-badge ${msg.status === 'sent' ? 'status-completed' : msg.status === 'failed' ? 'status-disconnected' : 'status-running'}`}>
                              {msg.status}
                            </span>
                          </td>
                          <td>
                            {msg.status === 'failed' && (
                              <button 
                                onClick={() => handleResend(msg.id)} 
                                className="btn btn-primary" 
                                style={{ padding: '4px 8px', fontSize: '11px' }}
                              >
                                Resend
                              </button>
                            )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
