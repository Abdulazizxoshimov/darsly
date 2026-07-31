// Jonly brend belgisi — "jon pulsi" chizig'i (B konsepsiya, manba: ~/jonly-logolar)
export default function BrandMark({ size = 30 }) {
  return (
    <div className="brand-logo" style={size !== 30 ? { width: size, height: size } : undefined}>
      <svg width={Math.round(size * 0.62)} height={Math.round(size * 0.62)} viewBox="0 0 120 120" aria-hidden="true">
        <path
          d="M16 62 H36 L49 32 L66 88 L77 62 H104"
          fill="none"
          stroke="currentColor"
          strokeWidth="14"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      </svg>
    </div>
  )
}
