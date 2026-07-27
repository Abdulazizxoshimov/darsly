package livekit

import (
	"github.com/livekit/protocol/auth"
)

// AccessToken darsga kirish uchun LiveKit JWT generatsiya qiladi.
//
//	isHost=true  → host: publish + subscribe + roomAdmin (mute/remove) + roomRecord
//	isHost=false → participant: publish + subscribe (cheklangan)
func (c *Client) AccessToken(roomName, identity, displayName string, isHost bool) (string, error) {
	grant := &auth.VideoGrant{
		RoomJoin:     true,
		Room:         roomName,
		CanSubscribe: boolPtr(true),
		// Webinar modeli (1 o'qituvchi, 1000-10000 talaba): FAQAT host media publish qiladi.
		// Student "qo'l ko'targanda" host UpdateParticipant orqali vaqtincha ruxsat beradi
		// (token qayta chiqarmasdan). Data-channel (chat/reaksiya) hammага ochiq.
		CanPublish:     boolPtr(isHost),
		CanPublishData: boolPtr(true),
	}
	if isHost {
		grant.RoomAdmin = true  // boshqalarni mute qilish / chiqarib yuborish
		grant.RoomRecord = true // yozib olishni boshqarish (Egress)
	}

	at := auth.NewAccessToken(c.apiKey, c.apiSecret).
		SetVideoGrant(grant).
		SetIdentity(identity).
		SetName(displayName).
		SetValidFor(c.tokenTTL)

	return at.ToJWT()
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
