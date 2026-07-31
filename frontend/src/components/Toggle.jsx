// Yoqish/o'chirish qatori (label + switch). Dizayn tizimidagi `.toggle`
// klasslari bilan; klaviaturadan ham boshqariladi (Space/Enter).
export function Toggle({ label, checked, onChange }) {
  function flip() {
    onChange(!checked)
  }
  return (
    <div
      className="toggle"
      role="switch"
      aria-checked={checked}
      tabIndex={0}
      onClick={flip}
      onKeyDown={(e) => {
        if (e.key === ' ' || e.key === 'Enter') {
          e.preventDefault()
          flip()
        }
      }}
    >
      <span style={{ fontWeight: 500 }}>{label}</span>
      <span className={`toggle__track ${checked ? 'on' : ''}`}>
        <span className="toggle__knob" />
      </span>
    </div>
  )
}
