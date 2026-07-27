// Oddiy toast emitter — kontekstsiz. Toaster komponenti obuna bo'ladi.
let seq = 0
const listeners = new Set()
let items = []

function emit() {
  for (const l of listeners) l(items)
}

function push(kind, message) {
  const id = ++seq
  items = [...items, { id, kind, message }]
  emit()
  setTimeout(() => {
    items = items.filter((t) => t.id !== id)
    emit()
  }, 4000)
}

export const toast = {
  success: (m) => push('success', m),
  error: (m) => push('error', m),
  info: (m) => push('info', m),
  dismiss: (id) => {
    items = items.filter((t) => t.id !== id)
    emit()
  },
  subscribe: (fn) => {
    listeners.add(fn)
    fn(items)
    return () => listeners.delete(fn)
  },
}
