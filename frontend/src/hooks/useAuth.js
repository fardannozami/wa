import { create } from 'zustand'
import api from '../api/client'

export const useAuthStore = create((set, get) => ({
  isAuthenticated: false,
  user: null,
  loading: true,
  
  checkAuth: async () => {
    const token = localStorage.getItem('token')
    if (!token) {
      set({ isAuthenticated: false, user: null, loading: false })
      return
    }
    
    try {
      const { data } = await api.get('/auth/me')
      set({ isAuthenticated: true, user: data.user, loading: false })
    } catch (e) {
      localStorage.removeItem('token')
      set({ isAuthenticated: false, user: null, loading: false })
    }
  },
  
  login: async () => {
    try {
      const { data } = await api.post('/auth/login')
      localStorage.setItem('token', data.token)
      set({ isAuthenticated: true, user: data.user })
      return data
    } catch (e) {
      throw e
    }
  },
  
  logout: async () => {
    try {
      await api.post('/auth/logout')
    } catch (e) {
      console.error(e)
    }
    localStorage.removeItem('token')
    set({ isAuthenticated: false, user: null })
  },
}))