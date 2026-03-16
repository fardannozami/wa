import { useEffect, useState } from 'react'
import { campaignApi, contactApi } from '../api/client'

export default function Campaigns() {
  const [campaigns, setCampaigns] = useState([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const [showModal, setShowModal] = useState(false)
  const [contacts, setContacts] = useState([])
  const [selectedContacts, setSelectedContacts] = useState([])
  const [formData, setFormData] = useState({ name: '', template: '' })

  useEffect(() => {
    loadCampaigns()
  }, [page])

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

  const loadContacts = async () => {
    try {
      const { data } = await contactApi.list(1, 100)
      setContacts(data.data || [])
    } catch (e) {
      console.error(e)
    }
  }

  const openModal = async () => {
    await loadContacts()
    setSelectedContacts([])
    setShowModal(true)
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    try {
      await campaignApi.create({
        ...formData,
        contact_ids: selectedContacts,
      })
      setShowModal(false)
      setFormData({ name: '', template: '' })
      setSelectedContacts([])
      loadCampaigns()
    } catch (e) {
      console.error(e)
    }
  }

  const handleDelete = async (id) => {
    if (!confirm('Are you sure you want to delete this campaign?')) return
    try {
      await campaignApi.delete(id)
      loadCampaigns()
    } catch (e) {
      console.error(e)
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
                      <button onClick={() => handleDelete(campaign.id)} className="btn btn-danger" style={{ padding: '6px 12px', fontSize: '12px' }}>
                        Delete
                      </button>
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
        <div className="modal-overlay" onClick={() => setShowModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3 className="modal-title">Create Campaign</h3>
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
                  <textarea
                    className="form-input"
                    rows={4}
                    value={formData.template}
                    onChange={(e) => setFormData({ ...formData, template: e.target.value })}
                    placeholder="Hello {{name}}, this is your message..."
                    required
                  />
                  <small style={{ color: '#666' }}>Use {"{{name}}"} to personalize messages</small>
                </div>
                <div className="form-group">
                  <label className="form-label">Select Contacts</label>
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
                  <small style={{ color: '#666' }}>{selectedContacts.length} contacts selected</small>
                </div>
              </div>
              <div className="modal-footer">
                <button type="button" onClick={() => setShowModal(false)} className="btn btn-secondary">Cancel</button>
                <button type="submit" className="btn btn-primary" disabled={selectedContacts.length === 0}>
                  Create Campaign
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
