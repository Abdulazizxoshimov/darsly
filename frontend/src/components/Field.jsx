export function Field({ label, error, icon, textarea, className = '', ...props }) {
  const inputCls = ['input', icon ? 'input--icon' : '', error ? 'input--error' : '', className]
    .filter(Boolean)
    .join(' ')
  return (
    <div className="field">
      {label && <label className="field__label">{label}</label>}
      <div className="field__wrap">
        {icon && <span className="field__icon">{icon}</span>}
        {textarea ? (
          <textarea className={`textarea ${error ? 'input--error' : ''}`} {...props} />
        ) : (
          <input className={inputCls} {...props} />
        )}
      </div>
      {error && <p className="field__error">{error}</p>}
    </div>
  )
}
