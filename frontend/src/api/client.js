import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import axios from 'axios'
import { useAuthStore } from '../hooks/useAuth'

const api = axios.create({
  baseURL: '/api/v1',
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

export const authApi = {
  googleLogin: () => api.get('/auth/google'),
  login: () => api.post('/auth/login'),
  logout: () => api.post('/auth/logout'),
  me: () => api.get('/auth/me'),
}

export const deviceApi = {
  get: () => api.get('/device'),
  connect: () => api.post('/device/connect'),
  disconnect: () => api.post('/device/disconnect'),
  getStatus: () => api.get('/device/status'),
  getGroups: () => api.get('/device/groups'),
  importGroup: (groupJid) => api.post('/device/import-group', { group_jid: groupJid }),
}

export const messageApi = {
  send: (phone, message) => api.post('/messages', { phone, message }),
}

export const contactApi = {
  list: (page = 1, limit = 20, query = '') => api.get(`/contacts?page=${page}&limit=${limit}${query ? '&' + query : ''}`),
  create: (data) => api.post('/contacts', data),
  update: (id, data) => api.put(`/contacts/${id}`, data),
  delete: (id) => api.delete(`/contacts/${id}`),
  import: (formData) => api.post('/contacts/import', formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  }),
}

export const groupApi = {
  list: () => api.get('/groups'),
  create: (data) => api.post('/groups', data),
  update: (id, data) => api.put(`/groups/${id}`, data),
  delete: (id) => api.delete(`/groups/${id}`),
}

export const campaignApi = {
  list: (page = 1, limit = 20) => api.get(`/campaigns?page=${page}&limit=${limit}`),
  create: (data) => api.post('/campaigns', data),
  update: (id, data) => api.put(`/campaigns/${id}`, data),
  get: (id) => api.get(`/campaigns/${id}`),
  getMessages: (id) => api.get(`/campaigns/${id}/messages`),
  send: (id, data) => api.post(`/campaigns/${id}/send`, data),
  resendMessage: (messageId) => api.post(`/messages/${messageId}/resend`, {}),
  delete: (id) => api.delete(`/campaigns/${id}`),
}

export const adminApi = {
  getStats: () => api.get('/admin/stats'),
  listUsers: () => api.get('/admin/users'),
}

export default api
