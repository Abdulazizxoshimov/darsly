import { useState } from 'react'
import { Navigate } from 'react-router-dom'
import { Plus, Trash2, UserPlus } from 'lucide-react'
import { useApp } from '../store/app'
import { useCreateUser, useDeleteUser, useUsers } from '../store/data'
import { errorText } from '../api/api'
import { Button } from '../components/Button'
import { Field } from '../components/Field'
import { Modal } from '../components/Modal'
import { Avatar } from '../components/Avatar'
import { PageLoader } from '../components/Spinner'
import { toast } from '../lib/toast'

// Foydalanuvchilarni boshqarish — FAQAT admin (backend RBAC ham shuni talab
// qiladi). Ro'yxat `UserShort` qaytaradi: {id, full_name, email, avatar_url?,
// color} — rol/holat ro'yxat javobida YO'Q, shuning uchun jadvalda ko'rsatilmaydi.
export function UsersView() {
  const { user } = useApp()
  const [createOpen, setCreateOpen] = useState(false)
  const isAdmin = user?.role === 'admin'
  const { data, isLoading, isError } = useUsers({ limit: 100 }, { enabled: isAdmin })
  const del = useDeleteUser()

  // Admin bo'lmagan foydalanuvchi URL'ni qo'lda tersa — bosh sahifaga.
  if (user && !isAdmin) return <Navigate to="/app" replace />

  const users = data?.data || []

  async function removeUser(u) {
    if (!window.confirm(`${u.full_name} (${u.email}) o'chirilsinmi? Bu amalni qaytarib bo'lmaydi.`)) return
    try {
      await del.mutateAsync(u.id)
      toast.success("Foydalanuvchi o'chirildi")
    } catch (err) {
      toast.error(errorText(err, 'O‘chirib bo‘lmadi'))
    }
  }

  return (
    <div className="page">
      <div className="row between" style={{ marginBottom: 24 }}>
        <div>
          <h1 className="h1">Foydalanuvchilar</h1>
          <p className="text-2" style={{ fontSize: 14, marginTop: 4 }}>
            Hisoblarni yarating va boshqaring
          </p>
        </div>
        <Button onClick={() => setCreateOpen(true)}>
          <Plus size={18} /> Foydalanuvchi qo'shish
        </Button>
      </div>

      {isLoading ? (
        <div style={{ height: 260 }}>
          <PageLoader label="Foydalanuvchilar yuklanmoqda…" />
        </div>
      ) : isError ? (
        <div className="empty">
          <p className="text-2" style={{ fontSize: 14 }}>
            Foydalanuvchilarni yuklab bo'lmadi. Qayta urinib ko'ring.
          </p>
        </div>
      ) : !users.length ? (
        <div className="empty">
          <div className="empty__icon">
            <UserPlus size={28} />
          </div>
          <p className="text-2" style={{ fontSize: 14 }}>Hali foydalanuvchi yo'q</p>
        </div>
      ) : (
        <div className="table-wrap">
          <table className="table">
            <thead>
              <tr>
                <th>Foydalanuvchi</th>
                <th>Email</th>
                <th style={{ textAlign: 'right' }}>Amallar</th>
              </tr>
            </thead>
            <tbody>
              {users.map((u) => (
                <tr key={u.id}>
                  <td>
                    <div className="row gap-3">
                      <Avatar name={u.full_name} color={u.color} src={u.avatar_url} size={32} />
                      <span className="table__title">
                        {u.full_name}
                        {u.id === user?.id && (
                          <span className="muted" style={{ fontWeight: 500, fontSize: 12.5 }}> (siz)</span>
                        )}
                      </span>
                    </div>
                  </td>
                  <td className="table__meta">{u.email}</td>
                  <td>
                    <div className="row-actions">
                      <button
                        className="tbl-btn tbl-btn--danger"
                        onClick={() => removeUser(u)}
                        // O'zini o'chirish — ma'nosiz va xavfli (sessiya o'ladi).
                        disabled={u.id === user?.id || del.isPending}
                        title={u.id === user?.id ? "O'zingizni o'chira olmaysiz" : "O'chirish"}
                        aria-label={`${u.full_name}ni o'chirish`}
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

      <CreateUserModal open={createOpen} onClose={() => setCreateOpen(false)} />
    </div>
  )
}

// Yangi hisob (asosan mentor) yaratish — backend `CreateUserReq`:
// {full_name, email, password, role}.
function CreateUserModal({ open, onClose }) {
  const create = useCreateUser()
  const [fullName, setFullName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [role, setRole] = useState('mentor')

  function reset() {
    setFullName('')
    setEmail('')
    setPassword('')
    setRole('mentor')
  }

  async function submit(e) {
    e.preventDefault()
    try {
      await create.mutateAsync({ full_name: fullName, email, password, role })
      toast.success('Foydalanuvchi yaratildi')
      reset()
      onClose()
    } catch (err) {
      toast.error(errorText(err, 'Foydalanuvchi yaratilmadi'))
    }
  }

  return (
    <Modal open={open} onClose={onClose} title="Foydalanuvchi qo'shish" width={440}>
      <form onSubmit={submit} className="col gap-4">
        <Field
          label="To'liq ism"
          value={fullName}
          onChange={(e) => setFullName(e.target.value)}
          placeholder="Ism Familiya"
          required
          minLength={2}
        />
        <Field
          label="Email"
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          placeholder="email@misol.uz"
          required
        />
        <Field
          label="Parol"
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder="Kamida 8 belgi"
          required
          minLength={8}
        />
        <div className="field">
          <label className="field__label" htmlFor="new-user-role">Rol</label>
          <select
            id="new-user-role"
            className="input"
            value={role}
            onChange={(e) => setRole(e.target.value)}
            style={{ cursor: 'pointer' }}
          >
            <option value="mentor">Ustoz (mentor)</option>
            <option value="student">O'quvchi</option>
            <option value="admin">Administrator</option>
          </select>
        </div>
        <div className="row gap-3" style={{ marginTop: 8 }}>
          <Button type="button" variant="ghost" onClick={onClose} className="grow">
            Bekor qilish
          </Button>
          <Button type="submit" loading={create.isPending} className="grow">
            Yaratish
          </Button>
        </div>
      </form>
    </Modal>
  )
}
