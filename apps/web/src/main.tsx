import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router'
import { QueryClient, QueryClientProvider, MutationCache } from '@tanstack/react-query'
import './index.css'
import './store/auth' // initialise configureApi wiring before any fetch
import { toast } from './lib/toast'
import type { AIMutationMeta } from './lib/aiMutation'
import App from './App.tsx'

// MutationCache-level handlers berjalan meskipun komponen pemicunya sudah
// unmount (user pindah halaman saat AI masih bekerja) — beda dengan
// onSuccess/onError di useMutation yang gugur bersama komponennya. Semua aksi
// AI mendeklarasikan meta {successToast, errorToast, invalidate} sehingga:
// hasil selalu di-refresh ke cache, dan user selalu dapat toast saat selesai,
// di halaman mana pun dia berada.
const queryClient: QueryClient = new QueryClient({
  mutationCache: new MutationCache({
    onSuccess: (_data, variables, _ctx, mutation) => {
      const meta = mutation.meta as AIMutationMeta | undefined
      if (!meta) return
      meta.invalidate?.(variables)?.forEach((key) => {
        void queryClient.invalidateQueries({ queryKey: key })
      })
      if (meta.successToast) toast.success(meta.successToast)
    },
    onError: (error, _variables, _ctx, mutation) => {
      const meta = mutation.meta as AIMutationMeta | undefined
      if (!meta?.errorToast) return
      const msg = error instanceof Error && error.message ? error.message : meta.errorToast
      toast.error(msg)
    },
  }),
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      refetchOnWindowFocus: false,
    },
  },
})

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </QueryClientProvider>
  </StrictMode>
)
