import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import * as lessonsApi from '../api/lessons'
import * as joinApi from '../api/join'
import * as waitingApi from '../api/waitingroom'
import * as recordingsApi from '../api/recordings'
import * as notificationsApi from '../api/notifications'
import * as pollsApi from '../api/polls'
import * as chatApi from '../api/chat'
import * as archiveApi from '../api/archive'
import * as telegramApi from '../api/telegram'
import * as userApi from '../api/user'
import * as appApi from '../api/app'
import * as blocklistApi from '../api/blocklist'

/* -------- Query kalitlari (yagona manba) --------
 *
 * Kalitlar satr ko'rinishida to'rt-besh faylga tarqalgan edi va bekor qilish
 * (invalidate) shu sababli yarim ishlardi: xonada dars `live` bo'lganda yoki
 * yozuv boshlanganda Dashboard/Jadval/Yozuvlar 30 soniyagacha eski holatni
 * ko'rsatardi (`queryClient.staleTime`). Prefiks kalit (`qk.lessons`) butun
 * oilani bekor qiladi, aniq kalit (`qk.lesson(id)`) — bittasini.
 */
export const qk = {
  appConfig: ['app-config'],
  lessons: ['lessons'],
  lessonList: (params) => ['lessons', params || {}],
  lesson: (id) => ['lesson', id],
  joinPreview: (slug) => ['join-preview', slug],
  waitingStatus: (requestId) => ['waiting-status', requestId],
  waiting: ['waiting'],
  waitingOf: (lessonId) => ['waiting', lessonId],
  recordings: ['recordings'],
  recordingsOf: (lessonId) => ['recordings', lessonId],
  recording: (id) => ['recording', id],
  lessonArchive: (id) => ['lesson-archive', id],
  chatHistory: (lessonId) => ['chat-history', lessonId],
  telegramStatus: ['telegram-status'],
  notifications: ['notifications'],
  notificationList: (unread) => ['notifications', unread],
  unreadCount: ['unread-count'],
  polls: ['polls'],
  pollsOf: (lessonId) => ['polls', lessonId],
  pollResults: (pollId) => ['poll-results', pollId],
  blocklist: ['blocklist'],
  users: ['users'],
  userList: (params) => ['users', params || {}],
}

/* -------- App config (ochiq) -------- */
export function useAppConfig() {
  return useQuery({
    queryKey: qk.appConfig,
    queryFn: appApi.getAppConfig,
    staleTime: 5 * 60_000,
    retry: 1,
  })
}

/* -------- Lessons -------- */
export function useLessons(params) {
  return useQuery({
    queryKey: qk.lessonList(params),
    queryFn: () => lessonsApi.listLessons(params),
  })
}
export function useLesson(id) {
  return useQuery({ queryKey: qk.lesson(id), queryFn: () => lessonsApi.getLesson(id), enabled: !!id })
}
export function useCreateLesson() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: lessonsApi.createLesson,
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.lessons }),
  })
}
export function useUpdateLesson() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, body }) => lessonsApi.updateLesson(id, body),
    onSuccess: (_d, v) => {
      qc.invalidateQueries({ queryKey: qk.lessons })
      qc.invalidateQueries({ queryKey: qk.lesson(v.id) })
    },
  })
}
export function useDeleteLesson() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: lessonsApi.deleteLesson,
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.lessons }),
  })
}

/* -------- Jonli xona (host) -------- */

// Host tokeni darsni SERVERDA `live` ga o'tkazadi — ro'yxat va dars keshi
// eskiradi. Bekor qilinmasa Dashboard ustoz xonadan qaytganda 30 soniyagacha
// «Rejalashtirilgan» deb turardi.
export function useOpenRoom() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: lessonsApi.getHostToken,
    onSuccess: (_d, lessonId) => {
      qc.invalidateQueries({ queryKey: qk.lessons })
      qc.invalidateQueries({ queryKey: qk.lesson(lessonId) })
    },
  })
}

// Yakunlash: dars `ended`, yozuv to'xtaydi (yozuvlar ro'yxati va arxiv ham
// o'zgaradi).
export function useEndLesson() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: lessonsApi.endLesson,
    onSuccess: (_d, lessonId) => {
      qc.invalidateQueries({ queryKey: qk.lessons })
      qc.invalidateQueries({ queryKey: qk.lesson(lessonId) })
      qc.invalidateQueries({ queryKey: qk.recordings })
      qc.invalidateQueries({ queryKey: qk.lessonArchive(lessonId) })
    },
  })
}

export function useStartRecording() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: recordingsApi.startRecording,
    onSuccess: (_d, lessonId) => qc.invalidateQueries({ queryKey: qk.recordingsOf(lessonId) }),
  })
}

// `lessonId` — bekor qilish uchun (endpoint yozuv ID'si bilan ishlaydi).
export function useStopRecording() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ recordingId }) => recordingsApi.stopRecording(recordingId),
    onSuccess: (_d, v) => {
      qc.invalidateQueries({ queryKey: qk.recordingsOf(v.lessonId) })
      qc.invalidateQueries({ queryKey: qk.recording(v.recordingId) })
    },
  })
}

/* -------- Join / waiting -------- */
export function useJoinPreview(slug, opts = {}) {
  return useQuery({
    queryKey: qk.joinPreview(slug),
    queryFn: () => joinApi.previewJoinLink(slug),
    enabled: !!slug,
    retry: false,
    // Dars-oldi kutish sahifasi shu so'rovni davriy takrorlab statusni kuzatadi
    // (scheduled → live bo'lganda avto-kirish).
    refetchInterval: opts.refetchInterval,
  })
}

export const WAITING_POLL_MS = 2000
// Endpoint IP bo'yicha 20 so'rov/s bilan cheklangan (`router.go`); bitta NAT
// ortidagi katta sinf 2 soniyalik poll bilan unga yetib boradi. 429 da poll
// SEKINLASHADI (to'xtamaydi — WS uzilgan bo'lsa admit'ni shu yerdan bilamiz).
export function waitingPollInterval(query) {
  return query.state.error?.status === 429 ? WAITING_POLL_MS * 3 : WAITING_POLL_MS
}
export function useWaitingStatus(requestId, enabled = true) {
  return useQuery({
    queryKey: qk.waitingStatus(requestId),
    queryFn: () => waitingApi.getWaitingStatus(requestId),
    enabled: !!requestId && enabled,
    refetchInterval: waitingPollInterval,
    // 404 — so'rov haqiqatan yo'q (dars yakunlandi/bekor qilindi): qayta
    // urinish ma'nosiz. Qolgan xatolar (429, tarmoq) intervalda o'zi qaytadi.
    retry: (count, err) => err?.status !== 404 && count < 1,
  })
}
export function useWaiting(lessonId, enabled) {
  return useQuery({
    queryKey: qk.waitingOf(lessonId),
    queryFn: () => waitingApi.listWaiting(lessonId),
    enabled: !!lessonId && enabled,
    refetchInterval: 4000,
  })
}
export function useAdmit() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: waitingApi.admitWaiting,
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.waiting }),
  })
}
export function useReject() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: waitingApi.rejectWaiting,
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.waiting }),
  })
}
// Butun navbatni bir so'rovda kiritish. Natija sonlari (`total/admitted/failed`)
// chaqiruvchiga qaytadi — u foydalanuvchiga ROSTINI aytadi.
export function useAdmitAll() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: waitingApi.admitAllWaiting,
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.waiting }),
  })
}

/* -------- Recordings -------- */
export function useRecordings(lessonId) {
  return useQuery({
    queryKey: qk.recordingsOf(lessonId),
    queryFn: () => recordingsApi.listRecordings(lessonId),
    enabled: !!lessonId,
  })
}

/* -------- Dars arxivi (video + chat + materiallar) -------- */

// Javob ichida PRESIGNED havolalar bor (1 soatlik) va server uni
// `Cache-Control: no-store` bilan yuboradi — shuning uchun kesh qisqa umrli.
// Aks holda ustoz sahifani ertasi kuni ochganda "video ochilmadi" degan
// buzuq havolani ko'rardi.
export function useLessonArchive(lessonId) {
  return useQuery({
    queryKey: qk.lessonArchive(lessonId),
    queryFn: () => archiveApi.getLessonArchive(lessonId),
    enabled: !!lessonId,
    staleTime: 60_000,
    gcTime: 5 * 60_000,
    refetchOnWindowFocus: false,
  })
}

// Bitta yozuv holati — `processing`/`restoring` tugagunicha poll qilinadi.
// `intervalMs` ni SERVER dikta qiladi (`poll_after_s`), klient o'zicha emas.
export function useRecordingStatus(recordingId, { enabled = false, intervalMs = 5000 } = {}) {
  return useQuery({
    queryKey: qk.recording(recordingId),
    queryFn: () => recordingsApi.getRecording(recordingId),
    enabled: !!recordingId && enabled,
    // Poll O'ZI TO'XTAYDI: yakuniy holatga (`ready`/`archived`/`failed`/…)
    // yetganda interval `false` bo'ladi. `enabled` ni qayta hisoblash bilan
    // to'xtatish mumkin emas edi — u o'zi shu so'rov natijasiga bog'liq
    // (aylanma bog'liqlik), refetchInterval funksiyasi esa natijani ko'radi.
    refetchInterval: (query) => {
      const s = query.state.data?.status
      if (s === 'processing' || s === 'restoring') return intervalMs
      // `recording` — dars hali yozilmoqda. U ham o'zi yangilanadi, lekin
      // sekinroq: bu holat daqiqalab davom etadi va tez-tez so'rashning
      // ma'nosi yo'q.
      if (s === 'recording') return intervalMs * 3
      return false
    },
    // Fon tabda ham davom etadi: tiklash 30-60 soniya, ustoz shu payt
    // boshqa oynaga o'tib ketishi tabiiy va qaytganda tayyor holatni kutadi.
    refetchIntervalInBackground: true,
    retry: 1,
  })
}

export function useRestoreRecording() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: recordingsApi.restoreRecording,
    onSuccess: (res, recordingId) => {
      // Holatni DARHOL yangilaymiz — birinchi poll javobigacha tugma
      // «Tiklash» bo'lib turmasin (ikkinchi bosish yana so'rov yuborardi).
      qc.setQueryData(qk.recording(recordingId), (old) => ({
        ...(old || {}),
        id: recordingId,
        status: res.status,
      }))
    },
  })
}

/* -------- Telegram bog'lanishi (mentor) -------- */

export function useTelegramStatus({ refetchInterval } = {}) {
  return useQuery({
    queryKey: qk.telegramStatus,
    queryFn: telegramApi.getTelegramStatus,
    // Integratsiya o'chiq bo'lsa javob o'zgarmaydi — bekorga so'ramaymiz.
    staleTime: 60_000,
    refetchInterval,
    retry: 1,
  })
}
export function useTelegramLink() {
  return useMutation({ mutationFn: telegramApi.startTelegramLink })
}
export function useTelegramUnlink() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: telegramApi.unlinkTelegram,
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.telegramStatus }),
  })
}

/* -------- Notifications -------- */
export function useNotifications(unread = false) {
  return useQuery({
    queryKey: qk.notificationList(unread),
    queryFn: () => notificationsApi.listNotifications(unread),
  })
}
export function useUnreadCount() {
  return useQuery({
    queryKey: qk.unreadCount,
    queryFn: notificationsApi.unreadCount,
    refetchInterval: 30000,
  })
}
export function useMarkRead() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: notificationsApi.markRead,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.notifications })
      qc.invalidateQueries({ queryKey: qk.unreadCount })
    },
  })
}
export function useMarkAllRead() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: notificationsApi.markAllRead,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.notifications })
      qc.invalidateQueries({ queryKey: qk.unreadCount })
    },
  })
}

/* -------- Polls (host) -------- */
export function usePolls(lessonId) {
  return useQuery({ queryKey: qk.pollsOf(lessonId), queryFn: () => pollsApi.listPolls(lessonId), enabled: !!lessonId })
}
export function usePollResults(pollId, token, opts = {}) {
  return useQuery({
    queryKey: qk.pollResults(pollId),
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
    onSuccess: (_d, v) => qc.invalidateQueries({ queryKey: qk.pollsOf(v.lessonId) }),
  })
}
export function useClosePoll() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: pollsApi.closePoll,
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.polls }),
  })
}
// Natijani e'lon qilish — poll ro'yxati (`results_published_at`) VA natija
// keshini yangilaydi: tugma bosilgan zahoti diagramma ustozda ham yangilanadi.
export function usePublishPoll() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ lessonId, pollId }) => pollsApi.publishPoll(lessonId, pollId),
    onSuccess: (res, v) => {
      qc.invalidateQueries({ queryKey: qk.pollsOf(v.lessonId) })
      if (res) qc.setQueryData(qk.pollResults(v.pollId), res)
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
      qc.setQueryData(qk.chatHistory(v.lessonId), (old) =>
        Array.isArray(old) ? old.filter((m) => m.id !== v.messageId) : old,
      )
      // Arxiv sahifasi ham SHU xabarni ko'rsatadi. Kesh yangilanmasa
      // o'chirilgan xabar arxivda qolib ketardi (moderatsiya yarim ish
      // bo'lardi) — va qayta yuklash 1 soatlik havolalarni bekorga qayta
      // imzolashga majbur qilardi.
      qc.setQueryData(qk.lessonArchive(v.lessonId), (old) =>
        old && Array.isArray(old.chat)
          ? { ...old, chat: old.chat.filter((m) => m.id !== v.messageId) }
          : old,
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
export function useRequestAccountDeletion() {
  return useMutation({ mutationFn: userApi.requestAccountDeletion })
}

/* -------- Blocklist (mentor) -------- */
export function useBlocklist() {
  return useQuery({
    queryKey: qk.blocklist,
    queryFn: blocklistApi.listBlocklist,
  })
}
export function useUnblock() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: blocklistApi.unblock,
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.blocklist }),
  })
}

/* -------- Users (admin) -------- */
export function useUsers(params, opts = {}) {
  return useQuery({
    queryKey: qk.userList(params),
    queryFn: () => userApi.listUsers(params),
    // Admin bo'lmaganda so'rov umuman ketmaydi (403 shovqini bo'lmasin).
    enabled: opts.enabled ?? true,
  })
}
export function useCreateUser() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: userApi.createUser,
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.users }),
  })
}
export function useDeleteUser() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: userApi.deleteUser,
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.users }),
  })
}
export function useUpdateUser() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, body }) => userApi.updateUser(id, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.users }),
  })
}
export function useSetUserActive() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, active }) =>
      active ? userApi.activateUser(id) : userApi.deactivateUser(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.users }),
  })
}
