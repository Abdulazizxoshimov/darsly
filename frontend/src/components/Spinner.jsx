export function Spinner({ size = 20 }) {
  return <div className="spinner" style={{ width: size, height: size }} />
}

export function PageLoader({ label }) {
  return (
    <div className="page-loader">
      <Spinner size={28} />
      {label && <p>{label}</p>}
    </div>
  )
}
