import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface AuthStore {
  token: string | null
  customerId: string | null
  _hydrated: boolean
  setAuth: (token: string, customerId: string) => void
  limparAuth: () => void
}

export const useAuthStore = create<AuthStore>()(
  persist(
    (set) => ({
      token: null,
      customerId: null,
      _hydrated: false,
      setAuth: (token, customerId) => set({ token, customerId }),
      limparAuth: () => set({ token: null, customerId: null }),
    }),
    {
      name: 'platform-auth',
      onRehydrateStorage: () => (state) => {
        if (state) state._hydrated = true
      },
    },
  ),
)
