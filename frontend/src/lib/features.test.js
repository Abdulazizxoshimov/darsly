import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { isLiveRoomSupported } from './features'

// F-5 — brauzer jonli darsni ololadimi. Ikki xil gate:
//  · PUBLISHER (default) — QATTIQ: RTCPeerConnection + getUserMedia shart;
//  · GUEST/o'quvchi (needsPublish:false) — YUMSHOQ: faqat RTCPeerConnection.
//
// Nega yumshoq gate muhim: `navigator.mediaDevices` XAVFSIZ BO'LMAGAN originda
// (http + IP, localhost emas) brauzer tomonidan UMUMAN berilmaydi. Qattiq gate
// bunday LAN/dev muhitida o'quvchini noto'g'ri "brauzer eski" xabariga urardi —
// aslida u xonaga kirib dars ko'ra/eshita olardi.

const origRTC = window.RTCPeerConnection
const origMD = Object.getOwnPropertyDescriptor(navigator, 'mediaDevices')

function setRTC(ok) {
  if (ok) window.RTCPeerConnection = function () {}
  else delete window.RTCPeerConnection
}
function setMediaDevices(value) {
  Object.defineProperty(navigator, 'mediaDevices', { configurable: true, value })
}

beforeEach(() => {
  setRTC(true)
  setMediaDevices({ getUserMedia: () => {} })
})
afterEach(() => {
  window.RTCPeerConnection = origRTC
  if (origMD) Object.defineProperty(navigator, 'mediaDevices', origMD)
})

describe('isLiveRoomSupported — publisher (qattiq gate)', () => {
  it('RTC + getUserMedia bor → qo‘llab-quvvatlanadi', () => {
    expect(isLiveRoomSupported()).toBe(true)
    expect(isLiveRoomSupported({ needsPublish: true })).toBe(true)
  })

  // Bug: WebRTC yo'q brauzerda room.connect() tushunarsiz xato beradi —
  // oldindan bloklab tushuntirish kerak.
  it('RTCPeerConnection yo‘q → qo‘llab-quvvatlanmaydi', () => {
    setRTC(false)
    expect(isLiveRoomSupported()).toBe(false)
  })

  // Bug: publisher (ustoz) mediaDevices'siz kamera/mikrofon bera olmaydi —
  // uni xonaga kiritib keyin "qurilma yo'q" deyish yomon UX.
  it('mediaDevices yo‘q → publisher uchun qo‘llab-quvvatlanmaydi', () => {
    setMediaDevices(undefined)
    expect(isLiveRoomSupported({ needsPublish: true })).toBe(false)
  })

  it('getUserMedia funksiya emas → qo‘llab-quvvatlanmaydi', () => {
    setMediaDevices({})
    expect(isLiveRoomSupported({ needsPublish: true })).toBe(false)
  })
})

describe('isLiveRoomSupported — guest/o‘quvchi (yumshoq gate)', () => {
  // ⭐ Asosiy F-5 farqi: o'quvchida mediaDevices bo'lmasa ham (insecure origin),
  // RTC bo'lsa XONAGA KIRADI — faqat publish tugmalari o'chiq turadi.
  // Bug: yumshoq gate qattiq bo'lib qolsa, LAN sinovida o'quvchi noto'g'ri
  // "brauzer eski" ekraniga urilardi.
  it('mediaDevices yo‘q bo‘lsa ham RTC bilan qo‘llab-quvvatlanadi', () => {
    setMediaDevices(undefined)
    expect(isLiveRoomSupported({ needsPublish: false })).toBe(true)
  })

  // Bug: RTC ham bo'lmasa o'quvchi ham xonaga ulana olmaydi — bu haqiqiy blok.
  it('RTC ham yo‘q bo‘lsa o‘quvchi uchun ham qo‘llab-quvvatlanmaydi', () => {
    setRTC(false)
    setMediaDevices(undefined)
    expect(isLiveRoomSupported({ needsPublish: false })).toBe(false)
  })
})
