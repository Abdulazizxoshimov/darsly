package livekit

import (
	"context"

	"github.com/livekit/protocol/livekit"

	"github.com/zoom/darsly/internal/entity"
)

// SendData xonadagi barcha ishtirokchilarga data-message yuboradi (chat, reaksiya).
// RELIABLE — yetkazish kafolatlangan.
func (c *Client) SendData(ctx context.Context, room string, data []byte) error {
	return c.SendDataTo(ctx, room, data, nil)
}

// SendDataTo xabarni FAQAT ko'rsatilgan ishtirokchilarga yuboradi (shaxsiy chat).
// identities bo'sh bo'lsa — butun xonaga (SendData bilan bir xil).
//
// Maxfiylik shu yerda hal bo'ladi: shaxsiy xabar boshqa klientlarga UMUMAN
// yetib bormaydi. Muqobil — hammaga yuborib, klientda yashirish — xabarni
// brauzer konsolida ochiq qoldirardi, ya'ni "shaxsiy" so'zi yolg'on bo'lardi.
func (c *Client) SendDataTo(ctx context.Context, room string, data []byte, identities []string) error {
	_, err := c.room.SendData(ctx, &livekit.SendDataRequest{
		Room:                  room,
		Data:                  data,
		Kind:                  livekit.DataPacket_RELIABLE,
		DestinationIdentities: identities,
	})
	return err
}

// studentBaseSources — o'quvchiga DOIM ochiq manbalar: FAQAT mikrofon.
// O'quvchi ovozni O'ZI (ustoz ruxsatisiz) yoqib savolni darhol bera oladi —
// ovoz arzon (~30 kbps + DTX), sig'imga ta'siri yo'q. Kamera esa qimmat
// (video har tomoshabinga tarqatiladi — N×N), shuning uchun u alohida ruxsat
// bilan qo'shiladi ([studentVideoSources]).
//
// SCREEN_SHARE ikkala ro'yxatda ham YO'Q — ekranni faqat ustoz ulashadi.
// Diqqat: bo'sh CanPublishSources = "BARCHA manbalar", shuning uchun ro'yxat
// hech qachon bo'sh qolmasligi SHART.
var studentBaseSources = []livekit.TrackSource{
	livekit.TrackSource_MICROPHONE,
}

// studentVideoSources — ustoz "video"ga ruxsat berganda ochiladigan to'plam:
// mikrofon + kamera (ekran ulashish baribir yo'q — u faqat ustozda).
var studentVideoSources = []livekit.TrackSource{
	livekit.TrackSource_CAMERA,
	livekit.TrackSource_MICROPHONE,
}

// SetParticipantPublish o'quvchining KAMERA (video) publish huquqini o'zgartiradi
// (token qayta chiqarmasdan). Mikrofon bunga BOG'LIQ EMAS — u doim ochiq.
//
//	allowCamera=true  → kamera + mikrofon (ustoz videoga ruxsat berdi)
//	allowCamera=false → faqat mikrofon (video bekor; ovoz baribir qoladi)
//
// Ustoz (host) huquqlari bu yerdan emas, token'dan keladi (token.go) — ekran
// ulashish ustozda saqlanadi.
func (c *Client) SetParticipantPublish(ctx context.Context, room, identity string, allowCamera bool) error {
	_, err := c.room.UpdateParticipant(ctx, &livekit.UpdateParticipantRequest{
		Room:       room,
		Identity:   identity,
		Permission: studentPermission(allowCamera),
	})
	return err
}

// studentPermission — o'quvchiga beriladigan huquqlar (test qamrovi uchun ajratilgan).
// CanPublish DOIM true — o'quvchi kamida mikrofonni yoqa olishi shart; kamera
// ruxsati manbalar ro'yxati bilan boshqariladi.
func studentPermission(allowCamera bool) *livekit.ParticipantPermission {
	sources := studentBaseSources
	if allowCamera {
		sources = studentVideoSources
	}
	return &livekit.ParticipantPermission{
		CanSubscribe:      true,
		CanPublish:        true,
		CanPublishSources: sources,
		CanPublishData:    true,
	}
}

// RemoveParticipant ishtirokchini xonadan chiqarib yuboradi (kick).
func (c *Client) RemoveParticipant(ctx context.Context, room, identity string) error {
	_, err := c.room.RemoveParticipant(ctx, &livekit.RoomParticipantIdentity{Room: room, Identity: identity})
	return err
}

// MuteParticipant ishtirokchining track'larini mute qiladi.
// audioOnly=true — faqat mikrofon; false — audio va video.
func (c *Client) MuteParticipant(ctx context.Context, room, identity string, audioOnly bool) error {
	p, err := c.room.GetParticipant(ctx, &livekit.RoomParticipantIdentity{Room: room, Identity: identity})
	if err != nil {
		return err
	}
	for _, t := range p.Tracks {
		if t.Muted {
			continue
		}
		if audioOnly && t.Type != livekit.TrackType_AUDIO {
			continue
		}
		if _, err := c.room.MutePublishedTrack(ctx, &livekit.MuteRoomTrackRequest{
			Room: room, Identity: identity, TrackSid: t.Sid, Muted: true,
		}); err != nil {
			return err
		}
	}
	return nil
}

// ListParticipantViews xonadagi ishtirokchilarni domen-ko'rinishida qaytaradi.
func (c *Client) ListParticipantViews(ctx context.Context, room string) ([]entity.RoomParticipant, error) {
	resp, err := c.room.ListParticipants(ctx, &livekit.ListParticipantsRequest{Room: room})
	if err != nil {
		return nil, err
	}
	out := make([]entity.RoomParticipant, 0, len(resp.Participants))
	for _, p := range resp.Participants {
		v := entity.RoomParticipant{
			Identity:   p.Identity,
			Name:       p.Name,
			JoinedAtMs: p.JoinedAtMs,
			Active:     p.State == livekit.ParticipantInfo_ACTIVE,
			AudioMuted: true, // agar audio track yo'q/hammasi muted bo'lsa true qoladi
			VideoMuted: true,
		}
		hasAudio, hasVideo := false, false
		for _, t := range p.Tracks {
			switch t.Type {
			case livekit.TrackType_AUDIO:
				hasAudio = true
				if !t.Muted {
					v.AudioMuted = false
				}
			case livekit.TrackType_VIDEO:
				hasVideo = true
				if !t.Muted {
					v.VideoMuted = false
				}
			}
		}
		if !hasAudio {
			v.AudioMuted = true
		}
		if !hasVideo {
			v.VideoMuted = true
		}
		out = append(out, v)
	}
	return out, nil
}
