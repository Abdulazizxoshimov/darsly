import { useEffect, useRef, useState, useCallback } from 'react'
import { Room, RoomEvent, ConnectionState } from 'livekit-client'

// livekit-client'ga to'g'ridan-to'g'ri ulanadigan hook.
// Alohida state boshqarmaymiz — SDK obyektini o'qiymiz, event'da re-render qilamiz.
// Past-internet: adaptiveStream + dynacast + simulcast (DEPLOYMENT.md talabi).
export function useRoom({ wsUrl, token, publish }) {
  const [room, setRoom] = useState(null)
  const [connState, setConnState] = useState('connecting') // connecting|connected|reconnecting|disconnected
  const [, setTick] = useState(0)
  const bump = useCallback(() => setTick((n) => n + 1), [])

  useEffect(() => {
    const r = new Room({
      adaptiveStream: true,
      dynacast: true,
      publishDefaults: { simulcast: true },
    })

    const audioEls = new Map()

    const onState = (s) => {
      if (s === ConnectionState.Connected) setConnState('connected')
      else if (s === ConnectionState.Reconnecting) setConnState('reconnecting')
      else if (s === ConnectionState.Connecting) setConnState('connecting')
      else if (s === ConnectionState.Disconnected) setConnState('disconnected')
      bump()
    }

    const onTrackSubscribed = (track) => {
      if (track.kind === 'audio') {
        const el = track.attach()
        el.style.display = 'none'
        document.body.appendChild(el)
        audioEls.set(track.sid, el)
      }
      bump()
    }
    const onTrackUnsubscribed = (track) => {
      if (track.kind === 'audio') {
        track.detach().forEach((el) => el.remove())
        audioEls.delete(track.sid)
      }
      bump()
    }

    r.on(RoomEvent.ConnectionStateChanged, onState)
      .on(RoomEvent.ParticipantConnected, bump)
      .on(RoomEvent.ParticipantDisconnected, bump)
      .on(RoomEvent.TrackSubscribed, onTrackSubscribed)
      .on(RoomEvent.TrackUnsubscribed, onTrackUnsubscribed)
      .on(RoomEvent.TrackPublished, bump)
      .on(RoomEvent.TrackUnpublished, bump)
      .on(RoomEvent.LocalTrackPublished, bump)
      .on(RoomEvent.LocalTrackUnpublished, bump)
      .on(RoomEvent.TrackMuted, bump)
      .on(RoomEvent.TrackUnmuted, bump)
      .on(RoomEvent.ActiveSpeakersChanged, bump)
      .on(RoomEvent.ParticipantMetadataChanged, bump)
      .on(RoomEvent.ParticipantPermissionsChanged, bump)

    let cancelled = false
    ;(async () => {
      try {
        await r.connect(wsUrl, token)
        if (cancelled) return
        setRoom(r)
        setConnState('connected')
        if (publish) {
          await r.localParticipant.setCameraEnabled(true).catch(() => {})
          await r.localParticipant.setMicrophoneEnabled(true).catch(() => {})
        }
        await r.startAudio().catch(() => {})
      } catch {
        if (!cancelled) setConnState('disconnected')
      }
    })()

    return () => {
      cancelled = true
      audioEls.forEach((el) => el.remove())
      r.removeAllListeners()
      r.disconnect()
    }
  }, [wsUrl, token, publish, bump])

  return { room, connState, bump }
}

// Xonadagi barcha ishtirokchilar (local birinchi).
export function roomParticipants(room) {
  if (!room) return []
  return [room.localParticipant, ...room.remoteParticipants.values()]
}
