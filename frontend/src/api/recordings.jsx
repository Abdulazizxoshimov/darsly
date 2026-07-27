import { api } from './api'

export function startRecording(lessonId) {
  return api.post(`/lessons/${lessonId}/recording/start`)
}
export function stopRecording(recordingId) {
  return api.post(`/recordings/${recordingId}/stop`)
}
export function listRecordings(lessonId) {
  return api.get(`/lessons/${lessonId}/recordings`)
}
export function downloadRecording(recordingId) {
  return api.get(`/recordings/${recordingId}/download`)
}
