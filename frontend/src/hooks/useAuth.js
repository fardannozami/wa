import { create } from 'zustand'
import api from '../api/client'

export const useAuthStore = create((set, get) => ({
  isAuthenticated: false,
  user: null,
  loading: true,
  initialized: false,
  
  checkAuth: async () => {
    if (get().initialized) return
    
    const token = localStorage.getItem('token')
    if (!token) {
      set({ isAuthenticated: false, user: null, loading: false, initialized: true })
      return
    }
    
    try {
      const { data } = await api.get('/auth/me')
      set({ isAuthenticated: true, user: data.user, loading: false, initialized: true })
    } catch (e) {
      localStorage.removeItem('token')
      set({ isAuthenticated: false, user: null, loading: false, initialized: true })
    }
  },
  
  setAuth: (isAuthenticated, user = null) => set({ isAuthenticated, user }),
  
  
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