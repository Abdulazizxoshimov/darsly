import { Loader2 } from 'lucide-react'

const VARIANT = {
  primary: '',
  secondary: 'btn--secondary',
  ghost: 'btn--ghost',
  danger: 'btn--danger',
}

export function Button({
  variant = 'primary',
  size,
  loading,
  disabled,
  className = '',
  children,
  ...props
}) {
  const cls = ['btn', VARIANT[variant], size === 'sm' ? 'btn--sm' : size === 'lg' ? 'btn--lg' : '', className]
    .filter(Boolean)
    .join(' ')
  return (
    <button className={cls} disabled={disabled || loading} {...props}>
      {loading && <Loader2 size={16} className="spin-inline" style={{ animation: 'spin 0.7s linear infinite' }} />}
      {children}
    </button>
  )
}
