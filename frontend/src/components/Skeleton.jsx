// Skelet-yuklanish primitivi (shimmer). Spinner o'rniga kontent shaklini
// oldindan ko'rsatadi — past-tarmoqda kutish sub'ektiv qisqaradi.
// O'lchamlar dinamik bo'lgani uchun inline style bu yerda o'rinli.

export function Skeleton({ width = '100%', height = 12, radius, style }) {
  return (
    <div
      className="skeleton"
      style={{ width, height, ...(radius != null ? { borderRadius: radius } : null), ...style }}
      aria-hidden="true"
    />
  )
}

/** Dashboard yuklanish skeleti — stat kartalari + jadval qatorlari shakli. */
export function DashboardSkeleton({ rows = 6 }) {
  return (
    <div aria-busy="true" aria-label="Yuklanmoqda">
      <div className="stats">
        {Array.from({ length: 4 }, (_, i) => (
          <div className="stat" key={i}>
            <Skeleton width={90} height={12} />
            <Skeleton width={48} height={22} style={{ marginTop: 10 }} />
          </div>
        ))}
      </div>
      <div className="table-wrap" style={{ marginTop: 16 }}>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12, padding: 16 }}>
          {Array.from({ length: rows }, (_, i) => (
            <div key={i} className="row gap-3" style={{ alignItems: 'center' }}>
              <Skeleton width={180} height={14} />
              <Skeleton width={70} height={20} radius={999} />
              <Skeleton width={120} height={12} />
              <Skeleton width={60} height={12} style={{ marginLeft: 'auto' }} />
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
