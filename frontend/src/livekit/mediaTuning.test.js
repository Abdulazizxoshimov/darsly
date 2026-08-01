import { describe, expect, it } from 'vitest'
import {
  CAMERA_HIGH,
  CAMERA_LOW,
  CAMERA_PUBLISH,
  PUBLISH_DEFAULTS,
  SCREEN_CAPTURE,
  SCREEN_HIGH,
  SCREEN_LOW,
  SCREEN_MID,
  SCREEN_PUBLISH,
} from './mediaTuning'

/**
 * Past internet uchun media sozlamalari.
 *
 * Bu testlar "kod shundaymi?" degan savolga emas, **mahsulot qaroriga** javob
 * beradi: zaif tarmoqdagi o'quvchi slayd matnini o'qiy oladimi. Ular buzilsa —
 * foydalanuvchi buni faqat yomon internetli hududda sezadi, ya'ni biz emas,
 * o'qituvchi va o'quvchi to'laydi.
 *
 * Ular `mobile/.../MediaTuningTest.kt` ning ko'zgusi: ikki platforma bir xil
 * qarorlar bilan yuritilishi shart.
 */
describe('ekran ulashish narvoni', () => {
  it('uch pog\'onali — orada bo\'shliq qolmaydi', () => {
    // ⭐ Bu bitta aniq nosozlikni qo'riqlaydi: 300 kbps va 1.5 Mbps orasida
    // hech narsa bo'lmasa, 800 kbps li o'quvchi yuqori qatlamni ko'tara olmay
    // TO'G'RIDAN-TO'G'RI eng pastkisiga tushadi. SDK ikkita qo'shimcha preset
    // berilgandagina uch qatlam yasaydi (`publishUtils` — o'qib tekshirilgan).
    expect(PUBLISH_DEFAULTS.screenShareSimulcastLayers).toHaveLength(2)
    expect(PUBLISH_DEFAULTS.screenShareSimulcastLayers).toEqual([SCREEN_LOW, SCREEN_MID])
  })

  it('qatlamlar pastdan yuqoriga tartiblangan', () => {
    const [low, mid] = PUBLISH_DEFAULTS.screenShareSimulcastLayers
    expect(low.encoding.maxBitrate).toBeLessThan(mid.encoding.maxBitrate)
    expect(mid.encoding.maxBitrate).toBeLessThan(SCREEN_HIGH.encoding.maxBitrate)
  })

  it('pastki pog\'onalarda KADR qurbon bo\'ladi, o\'lcham emas', () => {
    // Mahsulot qoidasi raqamda. O'lchov (bir xil ~750 kbps byudjet):
    //   1280×720 @ 15 fps → qirra energiyasi 0.42
    //   1920×1080 @ 8 fps → qirra energiyasi 0.93
    // Matn uchun piksel kadrdan muhimroq, shuning uchun o'lcham hamma
    // pog'onada bir xil (kichraytirish yo'q), fps esa pastga tushadi.
    expect(SCREEN_LOW.width).toBe(SCREEN_HIGH.width)
    expect(SCREEN_MID.width).toBe(SCREEN_HIGH.width)
    expect(SCREEN_LOW.encoding.maxFramerate).toBeLessThan(SCREEN_MID.encoding.maxFramerate)
    expect(SCREEN_MID.encoding.maxFramerate).toBeLessThan(SCREEN_HIGH.encoding.maxFramerate)
  })

  it('eng past pog\'ona zaif 3G ga sig\'adi', () => {
    // Ovoz (RED bilan ~48 kbps) qo'shilganda ham 400 kbps dan oshmasin.
    expect(SCREEN_LOW.encoding.maxBitrate).toBeLessThanOrEqual(300_000)
  })

  it('yuqori pog\'ona tegilmagan — yaxshi internetda sifat pasaymaydi', () => {
    // Narvonni kengaytirish yaxshi kanaldagi o'quvchining sifatini
    // PASAYTIRMASLIGI kerak, aks holda muammoni ko'chirgan bo'lardik.
    expect(PUBLISH_DEFAULTS.screenShareEncoding).toBe(SCREEN_HIGH.encoding)
    expect(SCREEN_HIGH.encoding.maxBitrate).toBe(1_500_000)
  })
})

describe('trek ustuvorligi (Chrome cheklovi bilan)', () => {
  it('ustuvorlik ENG PAST qatlamda turadi', () => {
    // SDK: `canSetPriority = (Firefox && !iOS) || idx === 0`. Ya'ni Chrome
    // faqat 0-indeksdagi presetning ustuvorligini oladi. Avval `priority`
    // eng YUQORI qatlamda edi va Chrome uni jimgina tashlab yuborardi —
    // "ekran kameradan ustun" qoidasi amalda umuman qo'llanmasdi.
    expect(PUBLISH_DEFAULTS.screenShareSimulcastLayers[0].encoding.priority).toBe('high')
    expect(PUBLISH_DEFAULTS.videoSimulcastLayers[0].encoding.priority).toBe('low')
  })
})

describe('kamera — birinchi qurbon', () => {
  it('360p dan yuqoriga chiqmaydi', () => {
    expect(CAMERA_HIGH.height).toBe(360)
    expect(CAMERA_LOW.height).toBe(180)
  })

  it('kamera ekrandan arzon', () => {
    expect(CAMERA_HIGH.encoding.maxBitrate).toBeLessThan(SCREEN_HIGH.encoding.maxBitrate)
  })
})

describe('degradatsiya siyosati trek turiga bog\'liq', () => {
  it('xona darajasida BERILMAGAN', () => {
    // `publishDefaults` xona darajasidagi sozlama: u yerdagi
    // `maintain-resolution` KAMERAGA ham tushib, zaif tarmoqda ustozning
    // yuzini kichraytirish o'rniga MUZLATIB qo'yardi (mobil esa BALANCED
    // ishlatardi — ikki platforma bir-biriga zid edi).
    expect(PUBLISH_DEFAULTS.degradationPreference).toBeUndefined()
  })

  it('ekran o\'lchamni saqlaydi, kamera muvozanatni', () => {
    expect(SCREEN_PUBLISH.degradationPreference).toBe('maintain-resolution')
    expect(CAMERA_PUBLISH.degradationPreference).toBe('balanced')
  })
})

describe('ovoz — oxirigacha yashaydigan oqim', () => {
  it('RED va DTX oshkora yoqilgan', () => {
    // Bular SDK default'i ham, lekin OSHKORA qotirilgan: kimdir kelajakda
    // `publishDefaults` ni boshqa sabab bilan qayta yozsa ular jimgina
    // o'chib ketardi va buni faqat yomon tarmoqda sezish mumkin bo'lardi.
    expect(PUBLISH_DEFAULTS.red).toBe(true)
    expect(PUBLISH_DEFAULTS.dtx).toBe(true)
  })

  it('ovoz ekran oqimining kichik ulushi', () => {
    expect(PUBLISH_DEFAULTS.audioPreset.maxBitrate * 10).toBeLessThan(
      SCREEN_HIGH.encoding.maxBitrate,
    )
  })
})

describe('ekranni qo\'lga olish', () => {
  it('matn rejimi va 1080p chegara oshkora', () => {
    // `contentHint: 'text'` — enkoderga "keskinlik fps'dan muhim" signali.
    // `resolution` SDK default'i bilan bir xil, lekin matn sifati unga
    // to'g'ridan-to'g'ri bog'liq bo'lgani uchun tasodifan o'zgarmasligi kerak.
    expect(SCREEN_CAPTURE.contentHint).toBe('text')
    expect(SCREEN_CAPTURE.resolution.height).toBe(1080)
  })
})
