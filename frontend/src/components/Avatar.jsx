import { initials } from '../lib/format'

// Foydalanuvchi rangi bo'lsa ishlatiladi; aks holda brand ohangi.
export function Avatar({ name, color, src, size = 40 }) {
  const style = { width: size, height: size, fontSize: size * 0.38 }
  if (src) {
    return (
      <div className="avatar" style={style}>
        <img src={src} alt={name} />
      </div>
    )
  }
  if (color) {
    style.background = `${color}22`
    style.color = color
  }
  return (
    <div className="avatar" style={style}>
      {initials(name)}
    </div>
  )
}
