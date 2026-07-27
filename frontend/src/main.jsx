import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { QueryClientProvider } from '@tanstack/react-query'
import { queryClient } from './lib/queryClient'
import { AppProvider } from './store/app'
import { Toaster } from './components/Toaster'
import App from './App'
import './styles.css'

function render() {
  createRoot(document.getElementById('root')).render(
    <StrictMode>
      <QueryClientProvider client={queryClient}>
        <BrowserRouter>
          <AppProvider>
            <App />
            <Toaster />
          </AppProvider>
        </BrowserRouter>
      </QueryClientProvider>
    </StrictMode>,
  )
}

// VITE_USE_MOCK=true → MSW mock (backend'siz ishlash). Aks holda to'g'ridan-to'g'ri backend.
if (import.meta.env.VITE_USE_MOCK === 'true') {
  import('./test/browser').then(({ enableMocking }) => enableMocking().then(render))
} else {
  render()
}
