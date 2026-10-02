package livekit

import (
	"encoding/json"
	"time"

	"github.com/livekit/protocol/auth"
)

// RoleMetadata — tokenga imzolanadigan ishtirokchi metama'lumoti.
//
// Nega kerak: data-channel xabari kimdan kelganini KLIENT hal qila olmaydi —
// payload ichidagi "men hostman" degan maydonga ishonish ustozni taqlid qilishga
// ochiq eshik (mehmon konsoldan `publishData({sender_name:"Ustoz"})` yuboradi).
// LiveKit esa ishtirokchi metadata'sini TOKENDAN oladi (server imzolagan) va uni
// o'zgartirib bo'lmaydi — `RoomAdmin` grant'i boshqa klientlarga ko'rinmaydi,
// metadata esa ko'rinadi. Shuning uchun "kim host" savolining yagona ishonchli
// javobi shu.
type RoleMetadata struct {
	Role string `json:"role"` // "host" | "participant"
}

// RoleHost / RoleParticipant — metadata `role` maydonining qiymatlari.
// Klientlar bilan shartnoma: `frontend/src/livekit/messaging.js` (isHostParticipant)
// va `mobile/.../RoomDataParser.kt`.
const (
	RoleHost        = "host"
	RoleParticipant = "participant"
)

// AccessToken darsga kirish uchun LiveKit JWT generatsiya qiladi.
//
//	isHost=true  → host: publish (barcha manbalar) + subscribe + roomAdmin (mute/remove)
//	isHost=false → participant: publish (FAQAT kamera+mikrofon, ekran ulashish yo'q) + subscribe
//
// Token muddati rolga qarab: host uchun uzun (dars davomiyligi), ishtirokchi
// uchun qisqa — batafsil `participantTokenTTL` izohida.
func (c *Client) AccessToken(roomName, identity, displayName string, isHost bool) (string, error) {
	grant := &auth.VideoGrant{
		RoomJoin:     true,
		Room:         roomName,
		CanSubscribe: boolPtr(true),
		// Data-channel (chat / reaksiya / qo'l ko'tarish) HAMMAGA ochiq. Media
		// publish (CanPublish) esa rolga qarab quyida beriladi.
		CanPublishData: boolPtr(true),
	}
	role := RoleParticipant
	ttl := c.participantTTL()
	if isHost {
		grant.CanPublish = boolPtr(true) // barcha manbalar (kamera+mikrofon+ekran ulashish)
		grant.RoomAdmin = true           // boshqalarni mute qilish / chiqarib yuborish
		// RoomRecord ATAYLAB yo'q: egressni backend API key bilan boshlaydi,
		// klientga (host brauzeriga) bu grant kerak emas — token sizib ketsa ham
		// begona egress boshlab bo'lmaydi.
		role = RoleHost
		ttl = c.tokenTTL
	} else {
		// WEBINAR (Variant B): o'quvchi MIKROFONNI O'ZI yoqa oladi (ovozli savolni
		// ustozdan so'ramasdan, darhol berish uchun) — CanPublish=true, lekin
		// manbalar FAQAT mikrofon. KAMERA (video) esa ustoz ruxsati bilan:
		// o'quvchi qo'l ko'taradi (`POST /rooms/:id/hand`), ustoz `allow-speak`
		// (→ UpdateParticipant → [studentVideoSources]) bergach kamera qo'shiladi
		// (klientda `ParticipantPermissionsChanged` tetiklaydi).
		//
		// Nega faqat kamera cheklangan: video har tomoshabinga tarqatiladi (N×N),
		// bir necha o'quvchi kamera yoqsa 4-yadroli server to'ladi. Ovoz esa arzon
		// (~30 kbps + DTX). Toza 1→N vebinar ~100 o'quvchini bardosh beradi
		// (o'lchangan). SCREEN_SHARE ikkala holatda ham yo'q — ekranni faqat ustoz.
		grant.CanPublish = boolPtr(true)
		grant.SetCanPublishSources(studentBaseSources)
	}

	md, err := json.Marshal(RoleMetadata{Role: role})
	if err != nil {
		return "", err
	}

	at := auth.NewAccessToken(c.apiKey, c.apiSecret).
		SetVideoGrant(grant).
		SetIdentity(identity).
		SetName(displayName).
		SetMetadata(string(md)).
		SetValidFor(ttl)

	return at.ToJWT()
}

// participantTokenTTLMax — mehmon/o'quvchi tokenining eng uzun muddati.
//
// Nega host'dan qisqa: `RemoveParticipant` LiveKit'da faqat JORIY ulanishni uzadi —
// token qo'lda qoladi va TTL tugagunicha qayta ulanish uchun yaroqli. Ban Redis'da
// bor, lekin uni faqat BIZNING endpoint'lar tekshiradi; buzg'unchi esa to'g'ridan
// to'g'ri `room.connect(wsUrl, eskiToken)` qila oladi va backendga umuman
// murojaat qilmaydi. Qisqa TTL — bu oynani cheklaydi: chiqarilgan ishtirokchi
// ko'pi bilan shuncha vaqt qaytib kira oladi, keyin esa token yangilash uchun
// backendga (ban tekshiruviga) kelishi SHART.
//
// 30 daqiqa — ban oynasi bilan qayta ulanish qulayligi orasidagi muvozanat:
// tarmoq uzilishida (metro, lift) klient eski token bilan jimgina qaytadi.
const participantTokenTTLMax = 30 * time.Minute

// participantTTL — ishtirokchi tokeni muddati: konfiguratsiyadagi TTL va
// `participantTokenTTLMax` dan KICHIGI. Konfiguratsiya qiymati bundan qisqa
// bo'lsa (masalan sinovda) uni uzaytirmaymiz.
func (c *Client) participantTTL() time.Duration {
	if c.tokenTTL > 0 && c.tokenTTL < participantTokenTTLMax {
		return c.tokenTTL
	}
	return participantTokenTTLMax
}

func boolPtr(b bool) *bool { return &b }

// VerifyToken LiveKit access token'ini tekshiradi va identity/name/room qaytaradi.
// Guest'larni (JWT'siz) room-token orqali autentifikatsiya qilish uchun (ovoz berish, chat).
func (c *Client) VerifyToken(token string) (identity, name, room string, err error) {
	v, err := auth.ParseAPIToken(token)
	if err != nil {
		return "", "", "", err
	}
	_, grants, err := v.Verify(c.apiSecret)
	if err != nil {
		return "", "", "", err
	}
	if grants.Video != nil {
		room = grants.Video.Room
	}
	return grants.Identity, grants.Name, room, nil
}
