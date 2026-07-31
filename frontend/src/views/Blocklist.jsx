import { ShieldOff, Trash2 } from 'lucide-react'
import { useBlocklist, useUnblock } from '../store/data'
import { errorText } from '../api/api'
import { Avatar } from '../components/Avatar'
import { PageLoader } from '../components/Spinner'
import { formatDateTime } from '../lib/format'
import { toast } from '../lib/toast'

// Qora ro'yxat — mentorning DOIMIY bloklari (№4): darsdan «Doimiy (barcha
// darslarimdan)» bilan chiqarilganlar. Moslik ism bo'yicha (o'quvchida akkaunt
// yo'q — ongli cheklov), shuning uchun bir xil ismli boshqa odam ham kirolmasligi
// mumkin — ro'yxatni shu sahifadan tozalash mumkin.
export function Blocklist() {
  const { data, isLoading, isError } = useBlocklist()
  const unblock = useUnblock()

  const entries = data || []

  async function remove(entry) {
    if (!window.confirm(`${entry.display_name} blokdan chiqarilsinmi? U darslaringizga yana qo'shila oladi.`)) return
    try {
      await unblock.mutateAsync(entry.id)
      toast.success('Blokdan chiqarildi')
    } catch (err) {
      toast.error(errorText(err, 'Blokdan chiqarib bo‘lmadi'))
    }
  }

  return (
    <div className="page">
      <div style={{ marginBottom: 24 }}>
        <h1 className="h1">Qora ro'yxat</h1>
        <p className="text-2" style={{ fontSize: 14, marginTop: 4 }}>
          Barcha darslaringizdan doimiy chiqarilganlar. Blokdan chiqarilgan odam darsga qayta qo'shila oladi.
        </p>
      </div>

      {isLoading ? (
        <div style={{ height: 260 }}>
          <PageLoader label="Qora ro'yxat yuklanmoqda…" />
        </div>
      ) : isError ? (
        <div className="empty">
          <p className="text-2" style={{ fontSize: 14 }}>
            Qora ro'yxatni yuklab bo'lmadi. Qayta urinib ko'ring.
          </p>
        </div>
      ) : !entries.length ? (
        <div className="empty">
          <div className="empty__icon">
            <ShieldOff size={28} />
          </div>
          <p className="text-2" style={{ fontSize: 14 }}>Qora ro'yxat bo'sh</p>
          <p className="muted" style={{ fontSize: 13, marginTop: 4 }}>
            Darsda ishtirokchini «Doimiy» chiqarish tanlansa, u shu yerda ko'rinadi.
          </p>
        </div>
      ) : (
        <div className="table-wrap">
          <table className="table">
            <thead>
              <tr>
                <th>Ism</th>
                <th>Bloklangan sana</th>
                <th style={{ textAlign: 'right' }}>Amallar</th>
              </tr>
            </thead>
            <tbody>
              {entries.map((e) => (
                <tr key={e.id}>
                  <td>
                    <div className="row gap-3">
                      <Avatar name={e.display_name} size={32} />
                      <span className="table__title">{e.display_name}</span>
                    </div>
                  </td>
                  <td className="table__meta">{formatDateTime(e.created_at)}</td>
                  <td>
                    <div className="row-actions">
                      <button
                        className="tbl-btn tbl-btn--danger"
                        onClick={() => remove(e)}
                        disabled={unblock.isPending}
                        title="Blokdan chiqarish"
                        aria-label={`${e.display_name}ni blokdan chiqarish`}
                      >
                        <Trash2 size={15} />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
