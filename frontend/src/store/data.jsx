import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import * as lessonsApi from '../api/lessons'
import * as joinApi from '../api/join'
import * as waitingApi from '../api/waitingroom'
import * as recordingsApi from '../api/recordings'
import * as notificationsApi from '../api/notifications'
import * as pollsApi from '../api/polls'
import * as chatApi from '../api/chat'
import * as userApi from '../api/user'
import * as appApi from '../api/app'
import * as blocklistApi from '../api/blocklist'

/* -------- App config (ochiq) -------- */
export function useAppConfig() {
  return useQuery({
    queryKey: ['app-config'],
    queryFn: appApi.getAppConfig,
    staleTime: 5 * 60_000,
    retry: 1,
  })
}

/* -------- Lessons -------- */
export function useLessons(params) {
  return useQuery({
    queryKey: ['lessons', params || {}],
    queryFn: () => lessonsApi.listLessons(params),
  })
}
export function useLesson(id) {
  return useQuery({ queryKey: ['lesson', id], queryFn: () => lessonsApi.getLesson(id), enabled: !!id })
}
export function useCreateLesson() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: lessonsApi.createLesson,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['lessons'] }),
  })
}
export function useUpdateLesson() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, body }) => lessonsApi.updateLesson(id, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['lessons'] }),
  })
}
export function useDeleteLesson() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: lessonsApi.deleteLesson,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['lessons'] }),
  })
}

/* -------- Join / waiting -------- */
export function useJoinPreview(slug, opts = {}) {
  return useQuery({
    queryKey: ['join-preview', slug],
    queryFn: () => joinApi.previewJoinLink(slug),
    enabled: !!slug,
    retry: false,
    // Dars-oldi kutish sahifasi shu so'rovni davriy takrorlab statusni kuzatadi
    // (scheduled → live bo'lganda avto-kirish).
    refetchInterval: opts.refetchInterval,
  })
}
export function useWaitingStatus(requestId, enabled = true) {
  return useQuery({
    queryKey: ['waiting-status', requestId],
    queryFn: () => waitingApi.getWaitingStatus(requestId),
    enabled: !!requestId && enabled,
    refetchInterval: 2000,
  })
}
export function useWaiting(lessonId, enabled) {
  return useQuery({
    queryKey: ['waiting', lessonId],
    queryFn: () => waitingApi.listWaiting(lessonId),
    enabled: !!lessonId && enabled,
    refetchInterval: 4000,
  })
}
export function useAdmit() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: waitingApi.admitWaiting,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['waiting'] }),
  })
}
export function useReject() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: waitingApi.rejectWaiting,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['waiting'] }),
  })
}
// Butun navbatni bir so'rovda kiritish. Natija sonlari (`total/admitted/failed`)
// chaqiruvchiga qaytadi — u foydalanuvchiga ROSTINI aytadi.
export function useAdmitAll() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: waitingApi.admitAllWaiting,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['waiting'] }),
  })
}

/* -------- Recordings -------- */
export function useRecordings(lessonId) {
  return useQuery({
    queryKey: ['recordings', lessonId],
    queryFn: () => recordingsApi.listRecordings(lessonId),
    enabled: !!lessonId,
  })
}

/* -------- Notifications -------- */
export function useNotifications(unread = false) {
  return useQuery({
    queryKey: ['notifications', unread],
    queryFn: () => notificationsApi.listNotifications(unread),
  })
}
export function useUnreadCount() {
  return useQuery({
    queryKey: ['unread-count'],
    queryFn: notificationsApi.unreadCount,
    refetchInterval: 30000,
  })
}
export function useMarkRead() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: notificationsApi.markRead,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['notifications'] })
      qc.invalidateQueries({ queryKey: ['unread-count'] })
    },
  })
}
export function useMarkAllRead() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: notificationsApi.markAllRead,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['notifications'] })
      qc.invalidateQueries({ queryKey: ['unread-count'] })
    },
  })
}

/* -------- Polls (host) -------- */
export function usePolls(lessonId) {
  return useQuery({ queryKey: ['polls', lessonId], queryFn: () => pollsApi.listPolls(lessonId), enabled: !!lessonId })
}
export function usePollResults(pollId, token, opts = {}) {
  return useQuery({
    queryKey: ['poll-results', pollId],
    queryFn: () => pollsApi.pollResults(pollId, token),
    enabled: !!pollId && !!token && (opts.enabled ?? true),
    refetchInterval: opts.refetchInterval,
    // O'quvchida e'lon qilinmagan natija 403 beradi — bu KUTILGAN javob,
    // xato emas. Qayta urinish faqat serverni bezovta qilardi.
    retry: opts.retry ?? 1,
  })
}
export function useCreatePoll() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ lessonId, question, options, resultsVisibility }) =>
      pollsApi.createPoll(lessonId, question, options, resultsVisibility),
    onSuccess: (_d, v) => qc.invalidateQueries({ queryKey: ['polls', v.lessonId] }),
  })
}
export function useClosePoll() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: pollsApi.closePoll,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['polls'] }),
  })
}
// Natijani e'lon qilish — poll ro'yxati (`results_published_at`) VA natija
// keshini yangilaydi: tugma bosilgan zahoti diagramma ustozda ham yangilanadi.
export function usePublishPoll() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ lessonId, pollId }) => pollsApi.publishPoll(lessonId, pollId),
    onSuccess: (res, v) => {
      qc.invalidateQueries({ queryKey: ['polls', v.lessonId] })
      if (res) qc.setQueryData(['poll-results', v.pollId], res)
    },
  })
}

/* -------- Chat moderatsiyasi -------- */
// Xabar tarixdan butunlay o'chadi — kesh ham darhol tozalanadi (jonli xonada
// esa `chat_deleted` data-xabari bir vaqtning o'zida hammadan olib tashlaydi).
export function useDeleteChatMessage() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ lessonId, messageId }) => chatApi.deleteChatMessage(lessonId, messageId),
    onSuccess: (_d, v) => {
      qc.setQueryData(['chat-history', v.lessonId], (old) =>
        Array.isArray(old) ? old.filter((m) => m.id !== v.messageId) : old,
      )
    },
  })
}

/* -------- Profile -------- */
export function useUpdateProfile() {
  return useMutation({ mutationFn: userApi.updateProfile })
}
export function useChangePassword() {
  return useMutation({ mutationFn: ({ current, next }) => userApi.changePassword(current, next) })
}

/* -------- Blocklist (mentor) -------- */
export function useBlocklist() {
  return useQuery({
    queryKey: ['blocklist'],
    queryFn: blocklistApi.listBlocklist,
  })
}
export function useUnblock() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: blocklistApi.unblock,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['blocklist'] }),
  })
}

/* -------- Users (admin) -------- */
export function useUsers(params, opts = {}) {
  return useQuery({
    queryKey: ['users', params || {}],
    queryFn: () => userApi.listUsers(params),
    // Admin bo'lmaganda so'rov umuman ketmaydi (403 shovqini bo'lmasin).
    enabled: opts.enabled ?? true,
  })
}
export function useCreateUser() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: userApi.createUser,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['users'] }),
  })
}
export function useDeleteUser() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: userApi.deleteUser,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['users'] }),
  })
}
