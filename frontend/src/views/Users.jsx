import { useState } from 'react'
import { Navigate } from 'react-router-dom'
import { Pencil, Plus, Power, Trash2, UserPlus } from 'lucide-react'
import { useApp } from '../store/app'
import {
  useCreateUser,
  useDeleteUser,
  useSetUserActive,
  useUpdateUser,
  useUsers,
} from '../store/data'
import { errorText } from '../api/api'
import { Button } from '../components/Button'
import { Field } from '../components/Field'
import { Modal } from '../components/Modal'
import { Avatar } from '../components/Avatar'
import { PageLoader } from '../components/Spinner'
import { toast } from '../lib/toast'

const ROLE_LABELS = { admin: 'Administrator', mentor: 'Ustoz', student: "O'quvchi", guest: 'Mehmon' }
function roleLabel(role) {
  return ROLE_LABELS[role] || role
}

// Foydalanuvchilarni boshqarish — FAQAT admin (backend RBAC ham shuni talab
// qiladi). Ro'yxat `AdminUserRow` qaytaradi: rol, faollik va "o'chirish
// so'ralgan" belgisi bilan — admin nazorat qila oladi.
export function UsersView() {
  const { user } = useApp()
  const [createOpen, setCreateOpen] = useState(false)
  const [editUser, setEditUser] = useState(null)
  const isAdmin = user?.role === 'admin'
  const { data, isLoading, isError } = useUsers({ limit: 100 }, { enabled: isAdmin })
  const del = useDeleteUser()
  const setActive = useSetUserActive()

  // Admin bo'lmagan foydalanuvchi URL'ni qo'lda tersa — bosh sahifaga.
  if (user && !isAdmin) return <Navigate to="/app" replace />

  const users = data?.data || []
  // "O'chirish so'ralgan" mentorlar yuqorida ko'rinsin — admin darhol ko'radi.
  const sorted = [...users].sort(
    (a, b) => (b.deletion_requested_at ? 1 : 0) - (a.deletion_requested_at ? 1 : 0),
  )

  async function removeUser(u) {
    const requested = !!u.deletion_requested_at
    const msg = requested
      ? `${u.full_name} o'chirishni SO'RAGAN. Tasdiqlab o'chirasizmi? Bu amalni qaytarib bo'lmaydi.`
      : `${u.full_name} (${u.email}) o'chirilsinmi? Bu amalni qaytarib bo'lmaydi.`
    if (!window.confirm(msg)) return
    try {
      await del.mutateAsync(u.id)
      toast.success("Foydalanuvchi o'chirildi")
    } catch (err) {
      toast.error(errorText(err, 'O‘chirib bo‘lmadi'))
    }
  }

  async function toggleActive(u) {
    try {
      await setActive.mutateAsync({ id: u.id, active: !u.is_active })
      toast.success(u.is_active ? 'Nofaol qilindi' : 'Faollashtirildi')
    } catch (err) {
      toast.error(errorText(err, 'Bajarilmadi'))
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
                <th>Rol</th>
                <th>Holat</th>
                <th style={{ textAlign: 'right' }}>Amallar</th>
              </tr>
            </thead>
            <tbody>
              {sorted.map((u) => {
                const requested = !!u.deletion_requested_at
                const isSelf = u.id === user?.id
                return (
                  <tr key={u.id}>
                    <td>
                      <div className="row gap-3">
                        <Avatar name={u.full_name} color={u.color} src={u.avatar_url} size={32} />
                        <span className="table__title">
                          {u.full_name}
                          {isSelf && (
                            <span className="muted" style={{ fontWeight: 500, fontSize: 12.5 }}> (siz)</span>
                          )}
                          {requested && (
                            <span className="badge badge--live" style={{ marginLeft: 8 }}>
                              O'chirish so'ralgan
                            </span>
                          )}
                        </span>
                      </div>
                    </td>
                    <td className="table__meta">{u.email}</td>
                    <td className="table__meta">{roleLabel(u.role)}</td>
                    <td className="table__meta">
                      {u.is_active ? 'Faol' : <span className="muted">Nofaol</span>}
                    </td>
                    <td>
                      <div className="row-actions">
                        <button
                          className="tbl-btn"
                          onClick={() => setEditUser(u)}
                          title="Tahrirlash"
                          aria-label={`${u.full_name}ni tahrirlash`}
                        >
                          <Pencil size={15} />
                        </button>
                        <button
                          className="tbl-btn"
                          onClick={() => toggleActive(u)}
                          disabled={isSelf || setActive.isPending}
                          title={u.is_active ? 'Nofaol qilish' : 'Faollashtirish'}
                          aria-label={`${u.full_name} holatini o'zgartirish`}
                        >
                          <Power size={15} />
                        </button>
                        <button
                          className="tbl-btn tbl-btn--danger"
                          onClick={() => removeUser(u)}
                          // O'zini o'chirish — ma'nosiz va xavfli (sessiya o'ladi).
                          disabled={isSelf || del.isPending}
                          title={isSelf ? "O'zingizni o'chira olmaysiz" : requested ? "Tasdiqlab o'chirish" : "O'chirish"}
                          aria-label={`${u.full_name}ni o'chirish`}
                        >
                          <Trash2 size={15} />
                        </button>
                      </div>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}

      <CreateUserModal open={createOpen} onClose={() => setCreateOpen(false)} />
      {/* key — foydalanuvchi almashganda forma qayta yaratilsin (effect'siz init). */}
      <EditUserModal key={editUser?.id} user={editUser} onClose={() => setEditUser(null)} />
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

// Mentorni tahrirlash — ism va rol. Backend `PUT /users/:id` (UpdateUserReq).
function EditUserModal({ user, onClose }) {
  const update = useUpdateUser()
  // `key={user.id}` (yuqorida) sabab bu komponent har foydalanuvchi uchun qayta
  // yaratiladi — shuning uchun boshlang'ich holatni to'g'ridan-to'g'ri prop'dan
  // olamiz (effect kerak emas).
  const [fullName, setFullName] = useState(user?.full_name ?? '')
  const [role, setRole] = useState(user?.role ?? 'mentor')

  if (!user) return null

  async function submit(e) {
    e.preventDefault()
    try {
      await update.mutateAsync({ id: user.id, body: { full_name: fullName, role } })
      toast.success('Saqlandi')
      onClose()
    } catch (err) {
      toast.error(errorText(err, "Saqlab bo'lmadi"))
    }
  }

  return (
    <Modal open={!!user} onClose={onClose} title="Foydalanuvchini tahrirlash" width={440}>
      <form onSubmit={submit} className="col gap-4">
        <Field
          label="To'liq ism"
          value={fullName}
          onChange={(e) => setFullName(e.target.value)}
          required
          minLength={2}
        />
        <div className="field">
          <label className="field__label" htmlFor="edit-user-role">Rol</label>
          <select
            id="edit-user-role"
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
          <Button type="submit" loading={update.isPending} className="grow">
            Saqlash
          </Button>
        </div>
      </form>
    </Modal>
  )
}
