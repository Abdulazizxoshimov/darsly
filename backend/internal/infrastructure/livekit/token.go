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
//	isHost=true  → host: publish (barcha manbalar) + subscribe + roomAdmin (mute/remove) + roomRecord
//	isHost=false → participant: publish (FAQAT kamera+mikrofon, ekran ulashish yo'q) + subscribe
//
// Token muddati rolga qarab: host uchun uzun (dars davomiyligi), ishtirokchi
// uchun qisqa — batafsil `participantTokenTTL` izohida.
func (c *Client) AccessToken(roomName, identity, displayName string, isHost bool) (string, error) {
	grant := &auth.VideoGrant{
		RoomJoin:     true,
		Room:         roomName,
		CanSubscribe: boolPtr(true),
		// ZOOM MODELI (PRODUCT.md D-intervyu): o'quvchi ham mikrofon va kamerani
		// ERKIN yoqa oladi — publish hammaga ochiq. Ovoz tartibi token bilan emas,
		// dars sozlamalari bilan boshqariladi (mute_on_entry / allow_self_unmute,
		// server webhook'da qo'llaydi). Data-channel (chat/reaksiya) hammaga ochiq.
		CanPublish:     boolPtr(true),
		CanPublishData: boolPtr(true),
	}
	role := RoleParticipant
	ttl := c.participantTTL()
	if isHost {
		grant.RoomAdmin = true  // boshqalarni mute qilish / chiqarib yuborish
		grant.RoomRecord = true // yozib olishni boshqarish (Egress)
		role = RoleHost
		ttl = c.tokenTTL
	} else {
		// EKRAN ULASHISH o'quvchiga BERILMAYDI (screen_share/screen_share_audio
		// ro'yxatda yo'q) — ekranni faqat ustoz ulashadi, aks holda o'quvchi
		// darsni buzish vektoriga ega bo'lardi. Diqqat: bo'sh ro'yxat LiveKit'da
		// "BARCHA manbalar" degani, shuning uchun ro'yxat aniq berilishi SHART.
		grant.SetCanPublishSources(studentPublishSources)
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
