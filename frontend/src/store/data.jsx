import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import * as lessonsApi from '../api/lessons'
import * as joinApi from '../api/join'
import * as waitingApi from '../api/waitingroom'
import * as recordingsApi from '../api/recordings'
import * as notificationsApi from '../api/notifications'
import * as pollsApi from '../api/polls'
import * as userApi from '../api/user'

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
export function useJoinPreview(slug) {
  return useQuery({
    queryKey: ['join-preview', slug],
    queryFn: () => joinApi.previewJoinLink(slug),
    enabled: !!slug,
    retry: false,
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
  })
}
export function useCreatePoll() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ lessonId, question, options }) => pollsApi.createPoll(lessonId, question, options),
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

/* -------- Profile -------- */
export function useUpdateProfile() {
  return useMutation({ mutationFn: userApi.updateProfile })
}
export function useChangePassword() {
  return useMutation({ mutationFn: ({ current, next }) => userApi.changePassword(current, next) })
}
