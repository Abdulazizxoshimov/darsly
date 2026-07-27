import { setupWorker } from 'msw/browser'
import { handlers } from './handlers'

export const worker = setupWorker(...handlers)

// VITE_USE_MOCK=true bo'lganda main.jsx render'dan oldin chaqiradi.
export async function enableMocking() {
  await worker.start({ onUnhandledRequest: 'bypass' })
}
