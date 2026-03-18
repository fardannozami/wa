import { useEffect, useState } from 'react'
import toast from 'react-hot-toast'
import { contactApi, messageApi, deviceApi, groupApi } from '../api/client'

export default function Contacts() {
  const [contacts, setContacts] = useState([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const [showModal, setShowModal] = useState(false)
  const [editingContact, setEditingContact] = useState(null)
  const [showMessageModal, setShowMessageModal] = useState(false)
  const [selectedContact, setSelectedContact] = useState(null)
  const [messageText, setMessageText] = useState('')
  const [sending, setSending] = useState(false)
  const [deviceStatus, setDeviceStatus] = useState('disconnected')
  const [formData, setFormData] = useState({ name: '', phone: '', prefix: '', group_ids: [] })
  const [selectedGroup, setSelectedGroup] = useState('')
  const [groups, setGroups] = useState([])
  const [showGroupModal, setShowGroupModal] = useState(false)
  const [groupFormData, setGroupFormData] = useState({ name: '' })
  const [showWhatsappModal, setShowWhatsappModal] = useState(false)
  const [whatsappGroups, setWhatsappGroups] = useState([])
  const [loadingWhatsappGroups, setLoadingWhatsappGroups] = useState(false)
  const [importingFromGroup, setImportingFromGroup] = useState(null)
  const [groupSearch, setGroupSearch] = useState('')

  useEffect(() => {
    loadContacts()
    loadDeviceStatus()
    loadGroups()
  }, [page, selectedGroup])

  const loadGroups = async () => {
    try {
      const { data } = await groupApi.list()
      setGroups(data.data || [])
    } catch (e) {
      console.error(e)
    }
  }

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

  const handleSendMessage = async (e) => {
    e.preventDefault()
    if (!selectedContact) return
    
    setSending(true)
    try {
      await messageApi.send(selectedContact.phone, messageText)
      setShowMessageModal(false)
      setSelectedContact(null)
      setMessageText('')
      toast.success('Message sent successfully!')
    } catch (e) {
      console.error(e)
      toast.error('Failed to send message: ' + e.response?.data?.error)
    } finally {
      setSending(false)
    }
  }

  const openMessageModal = (contact) => {
    setSelectedContact(contact)
    setShowMessageModal(true)
  }

  const loadContacts = async () => {
    setLoading(true)
    try {
      const params = new URLSearchParams()
      params.append('page', page)
      params.append('limit', 20)
      if (selectedGroup) {
        params.append('group_id', selectedGroup)
      }
      const { data } = await contactApi.list(page, 20, params.toString())
      setContacts(data.data || [])
      setTotal(data.total || 0)
    } catch (e) {
      console.error(e)
    } finally {
      setLoading(false)
    }
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    try {
      if (editingContact) {
        await contactApi.update(editingContact.id, formData)
        toast.success('Contact updated')
      } else {
        await contactApi.create(formData)
        toast.success('Contact created')
      }
      setShowModal(false)
      setEditingContact(null)
      setFormData({ name: '', phone: '', prefix: '', group_ids: [] })
      loadContacts()
    } catch (e) {
      console.error(e)
      toast.error('Failed to save contact')
    }
  }

  const openEditModal = (contact) => {
    loadGroups()
    setEditingContact(contact)
    const groupIds = contact.groups ? contact.groups.map(g => g.id) : []
    setFormData({ name: contact.name, phone: contact.phone, prefix: contact.prefix || '', group_ids: groupIds })
    setShowModal(true)
  }

  const openAddModal = () => {
    loadGroups()
    setEditingContact(null)
    setFormData({ name: '', phone: '', prefix: '', group_ids: [] })
    setShowModal(true)
  }

  const handleGroupSubmit = async (e) => {
    e.preventDefault()
    try {
      await groupApi.create(groupFormData)
      toast.success('Group created')
      setShowGroupModal(false)
      setGroupFormData({ name: '' })
      loadGroups()
    } catch (e) {
      console.error(e)
      toast.error('Failed to create group')
    }
  }

  const loadWhatsappGroups = async () => {
    setLoadingWhatsappGroups(true)
    try {
      const { data } = await deviceApi.getGroups()
      setWhatsappGroups(data.data || [])
    } catch (e) {
      console.error(e)
      toast.error('Failed to load WhatsApp groups - device may not be connected')
    } finally {
      setLoadingWhatsappGroups(false)
    }
  }

  const openWhatsappModal = async () => {
    setShowWhatsappModal(true)
    setWhatsappGroups([])
    await loadWhatsappGroups()
  }

  const handleImportFromWhatsappGroup = async (group) => {
    setImportingFromGroup(group.jid)
    try {
      const { data } = await deviceApi.importGroup(group.jid)
      const count = data.data?.imported_count || 0
      
      toast.success(`Imported ${count} contacts from ${group.name}`)
      setShowWhatsappModal(false)
      loadContacts()
    } catch (e) {
      console.error(e)
      toast.error('Failed to import contacts')
    } finally {
      setImportingFromGroup(null)
    }
  }

  const handleDeleteGroup = async (id) => {
    toast((t) => (
      <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
        <span>Delete this group?</span>
        <div style={{ display: 'flex', gap: '5px' }}>
          <button 
            onClick={() => { toast.dismiss(t.id); deleteGroup(id) }}
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

  const deleteGroup = async (id) => {
    try {
      await groupApi.delete(id)
      toast.success('Group deleted')
      loadGroups()
      if (selectedGroup === id) {
        setSelectedGroup('')
      }
    } catch (e) {
      console.error(e)
      toast.error('Failed to delete group')
    }
  }

  const handleDelete = async (id) => {
    toast((t) => (
      <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
        <span>Delete this contact?</span>
        <div style={{ display: 'flex', gap: '5px' }}>
          <button 
            onClick={() => { toast.dismiss(t.id); deleteContact(id) }}
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

  const deleteContact = async (id) => {
    try {
      await contactApi.delete(id)
      toast.success('Contact deleted')
      loadContacts()
    } catch (e) {
      console.error(e)
      toast.error('Failed to delete contact')
    }
  }

  const handleImport = async (e) => {
    const file = e.target.files[0]
    if (!file) return
    
    const formData = new FormData()
    formData.append('file', file)
    
    try {
      const { data } = await contactApi.import(formData)
      loadContacts()
      toast.success(`Imported: ${data.created} new, ${data.updated} updated`)
    } catch (e) {
      console.error(e)
      toast.error('Failed to import contacts')
    }
  }

  const handleExport = async () => {
    try {
      const { data } = await contactApi.list(1, 10000)
      const contacts = data.data || []
      
      const csvContent = [
        ['Name', 'Phone', 'Prefix', 'Groups'].join(','),
        ...contacts.map(c => [
          `"${c.name || ''}"`,
          `"${c.phone || ''}"`,
          `"${c.prefix || ''}"`,
          `"${c.groups && c.groups.length > 0 ? c.groups.map(g => g.name).join(',') : ''}"`
        ].join(','))
      ].join('\n')

      const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
      const link = document.createElement('a')
      link.href = URL.createObjectURL(blob)
      link.download = `contacts_${new Date().toISOString().split('T')[0]}.csv`
      link.click()
      toast.success('Contacts exported!')
    } catch (e) {
      console.error(e)
      toast.error('Failed to export contacts')
    }
  }

  const downloadTemplate = () => {
    const template = [
      'Name,Phone,Prefix,Groups',
      'John Doe,628123456789,Pak,Customer',
      'Jane Smith,628987654321,Bu,"VIP,Premium"',
      'Bob Wilson,628111222333,Boss,"Customer,Premium"'
    ].join('\n')

    const blob = new Blob([template], { type: 'text/csv;charset=utf-8;' })
    const link = document.createElement('a')
    link.href = URL.createObjectURL(blob)
    link.download = 'contacts_template.csv'
    link.click()
  }

  return (
    <div>
      <div className="page-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h1 className="page-title">Contacts</h1>
        <div style={{ display: 'flex', gap: '10px' }}>
          <button onClick={downloadTemplate} className="btn btn-secondary" style={{ padding: '8px 16px' }}>
            Template CSV
          </button>
          <button onClick={handleExport} className="btn btn-secondary" style={{ padding: '8px 16px' }}>
            Export CSV
          </button>
          <button onClick={openWhatsappModal} className="btn btn-secondary" style={{ padding: '8px 16px' }}>
            Import from WhatsApp
          </button>
          <label className="btn btn-secondary">
            Import CSV
            <input type="file" accept=".csv" onChange={handleImport} style={{ display: 'none' }} />
          </label>
          <button onClick={openAddModal} className="btn btn-primary">
            + Add Contact
          </button>
        </div>
      </div>

      <div className="card">
        <div style={{ marginBottom: '20px', display: 'flex', gap: '10px', alignItems: 'center', flexWrap: 'wrap' }}>
          <div style={{ fontWeight: '500' }}>Filter by Group:</div>
          <select 
            value={selectedGroup} 
            onChange={(e) => { setSelectedGroup(e.target.value); setPage(1) }}
            className="form-input"
            style={{ width: 'auto', padding: '6px 12px' }}
          >
            <option value="">All Contacts</option>
            {groups.map(group => (
              <option key={group.id} value={group.id}>{group.name}</option>
            ))}
          </select>
          <button onClick={() => setShowGroupModal(true)} className="btn btn-secondary" style={{ padding: '6px 12px', fontSize: '12px' }}>
            + New Group
          </button>
        </div>

        {loading ? (
          <div style={{ textAlign: 'center', padding: '40px' }}>Loading...</div>
        ) : contacts.length === 0 ? (
          <div style={{ textAlign: 'center', padding: '40px', color: '#666' }}>
            No contacts yet. Add your first contact or import from CSV.
          </div>
        ) : (
          <>
            <table className="table">
              <thead>
                <tr>
                  <th>Prefix</th>
                  <th>Name</th>
                  <th>Phone</th>
                  <th>Group</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {contacts.map((contact) => (
                  <tr key={contact.id}>
                    <td>{contact.prefix || '-'}</td>
                    <td>{contact.name}</td>
                    <td>{contact.phone}</td>
                    <td>{contact.groups && contact.groups.length > 0 ? contact.groups.map(g => g.name).join(', ') : '-'}</td>
                    <td>
                      <button onClick={() => openMessageModal(contact)} className="btn btn-primary" style={{ padding: '6px 12px', fontSize: '12px', marginRight: '8px' }} disabled={deviceStatus !== 'connected' && deviceStatus !== 'active'}>
                        Send
                      </button>
                      <button onClick={() => openEditModal(contact)} className="btn btn-secondary" style={{ padding: '6px 12px', fontSize: '12px', marginRight: '8px' }}>
                        Edit
                      </button>
                      <button onClick={() => handleDelete(contact.id)} className="btn btn-danger" style={{ padding: '6px 12px', fontSize: '12px' }}>
                        Delete
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            <div style={{ marginTop: '20px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <span>Total: {total} contacts</span>
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
              <h3 className="modal-title">{editingContact ? 'Edit Contact' : 'Add Contact'}</h3>
              <button onClick={() => { setShowModal(false); setEditingContact(null) }} style={{ background: 'none', border: 'none', fontSize: '20px', cursor: 'pointer' }}>×</button>
            </div>
            <form onSubmit={handleSubmit}>
              <div className="modal-body">
                                <div className="form-group">
                  <label className="form-label">Prefix</label>
                  <select
                    className="form-input"
                    value={formData.prefix}
                    onChange={(e) => setFormData({ ...formData, prefix: e.target.value })}
                  >
                    <option value="">-</option>
                    <option value="Pak">Pak</option>
                    <option value="Bu">Bu</option>
                    <option value="Bapak">Bapak</option>
                    <option value="Ibu">Ibu</option>
                    <option value="Boss">Boss</option>
                    <option value="Saudara">Saudara</option>
                    <option value="Saudari">Saudari</option>
                    <option value="Tn">Tn</option>
                    <option value="Ny">Ny</option>
                  </select>
                </div>
                <div className="form-group">
                  <label className="form-label">Name</label>
                  <input
                    type="text"
                    className="form-input"
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    required
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">Phone</label>
                  <input
                    type="text"
                    className="form-input"
                    value={formData.phone}
                    onChange={(e) => setFormData({ ...formData, phone: e.target.value })}
                    placeholder="+62812345678"
                    required
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">Groups (select multiple)</label>
                  <div style={{ marginBottom: '10px' }}>
                    <input 
                      type="text" 
                      placeholder="Search groups..." 
                      className="form-input" 
                      style={{ padding: '6px 12px', fontSize: '12px' }}
                      value={groupSearch}
                      onChange={(e) => setGroupSearch(e.target.value)}
                    />
                  </div>
                  <div style={{ 
                    maxHeight: '260px', 
                    overflowY: 'auto', 
                    border: '1px solid #e2e8f0', 
                    borderRadius: '8px', 
                    padding: '8px 4px',
                    background: '#f8fafc'
                  }}>
                    {groups
                      .filter(g => g.name.toLowerCase().includes(groupSearch.toLowerCase()))
                      .map(group => (
                      <label 
                        key={group.id} 
                        style={{ 
                          display: 'flex', 
                          alignItems: 'center', 
                          gap: '10px', 
                          padding: '8px 12px',
                          cursor: 'pointer',
                          borderRadius: '6px',
                          transition: 'background 0.2s ease'
                        }}
                        className="group-list-item"
                        onMouseEnter={(e) => e.target.style.background = '#f1f5f9'}
                        onMouseLeave={(e) => e.target.style.background = 'transparent'}
                      >
                        <input
                          type="checkbox"
                          checked={formData.group_ids.includes(group.id)}
                          onChange={(e) => {
                            if (e.target.checked) {
                              setFormData({ ...formData, group_ids: [...formData.group_ids, group.id] })
                            } else {
                              setFormData({ ...formData, group_ids: formData.group_ids.filter(id => id !== group.id) })
                            }
                          }}
                        />
                        <span style={{ fontSize: '13px', fontWeight: '500' }}>{group.name}</span>
                      </label>
                    ))}
                    {groups.length === 0 && <div style={{ color: '#666', padding: '10px', textAlign: 'center' }}>No groups available</div>}
                    {groups.length > 0 && groups.filter(g => g.name.toLowerCase().includes(groupSearch.toLowerCase())).length === 0 && (
                      <div style={{ color: '#666', padding: '10px', textAlign: 'center' }}>No groups matching "{groupSearch}"</div>
                    )}
                  </div>
                </div>
              </div>
              <div className="modal-footer">
                <button type="button" onClick={() => { setShowModal(false); setEditingContact(null) }} className="btn btn-secondary">Cancel</button>
                <button type="submit" className="btn btn-primary">{editingContact ? 'Update Contact' : 'Add Contact'}</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {showGroupModal && (
        <div className="modal-overlay" onClick={() => setShowGroupModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3 className="modal-title">Create Group</h3>
              <button onClick={() => setShowGroupModal(false)} style={{ background: 'none', border: 'none', fontSize: '20px', cursor: 'pointer' }}>×</button>
            </div>
            <form onSubmit={handleGroupSubmit}>
              <div className="modal-body">
                <div className="form-group">
                  <label className="form-label">Group Name</label>
                  <input
                    type="text"
                    className="form-input"
                    value={groupFormData.name}
                    onChange={(e) => setGroupFormData({ name: e.target.value })}
                    placeholder="e.g., Customer, VIP, etc."
                    required
                  />
                </div>
              </div>
              <div className="modal-footer">
                <button type="button" onClick={() => setShowGroupModal(false)} className="btn btn-secondary">Cancel</button>
                <button type="submit" className="btn btn-primary">Create Group</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {showMessageModal && selectedContact && (
        <div className="modal-overlay" onClick={() => setShowMessageModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3 className="modal-title">Send Message to {selectedContact.name}</h3>
              <button onClick={() => setShowMessageModal(false)} style={{ background: 'none', border: 'none', fontSize: '20px', cursor: 'pointer' }}>×</button>
            </div>
            <form onSubmit={handleSendMessage}>
              <div className="modal-body">
                <div className="form-group">
                  <label className="form-label">Phone</label>
                  <input type="text" className="form-input" value={selectedContact.phone} disabled />
                </div>
                <div className="form-group">
                  <label className="form-label">Message</label>
                  <textarea
                    className="form-input"
                    rows="5"
                    value={messageText}
                    onChange={(e) => setMessageText(e.target.value)}
                    placeholder="Type your message..."
                    required
                  />
                </div>
              </div>
              <div className="modal-footer">
                <button type="button" onClick={() => setShowMessageModal(false)} className="btn btn-secondary">Cancel</button>
                <button type="submit" className="btn btn-primary" disabled={sending}>
                  {sending ? 'Sending...' : 'Send Message'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {showWhatsappModal && (
        <div className="modal-overlay" onClick={() => setShowWhatsappModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3 className="modal-title">Import from WhatsApp Groups</h3>
              <button onClick={() => setShowWhatsappModal(false)} style={{ background: 'none', border: 'none', fontSize: '20px', cursor: 'pointer' }}>×</button>
            </div>
            <div className="modal-body">
              {loadingWhatsappGroups && <div style={{ textAlign: 'center', padding: '20px' }}>Loading groups...</div>}
              {!loadingWhatsappGroups && whatsappGroups.length === 0 && (
                <div style={{ textAlign: 'center', padding: '20px', color: '#666' }}>
                  No WhatsApp groups found. Make sure your device is connected.
                </div>
              )}
              {!loadingWhatsappGroups && whatsappGroups.length > 0 && (
                <div style={{ maxHeight: '400px', overflowY: 'auto' }}>
                  {whatsappGroups.map((group) => (
                    <div key={group.jid} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '12px', borderBottom: '1px solid #eee' }}>
                      <div>
                        <div style={{ fontWeight: '500' }}>{group.name || 'Unnamed Group'}</div>
                        <div style={{ fontSize: '12px', color: '#666' }}>{group.jid}</div>
                      </div>
                      <button onClick={() => handleImportFromWhatsappGroup(group)} className="btn btn-primary" style={{ padding: '6px 12px', fontSize: '12px' }} disabled={importingFromGroup === group.jid}>
                        {importingFromGroup === group.jid ? 'Importing...' : 'Import'}
                      </button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
