package livekit

import (
	"context"

	"github.com/livekit/protocol/livekit"

	"github.com/zoom/darsly/internal/entity"
)

// SendData xonadagi barcha ishtirokchilarga data-message yuboradi (chat, reaksiya).
// RELIABLE — yetkazish kafolatlangan.
func (c *Client) SendData(ctx context.Context, room string, data []byte) error {
	_, err := c.room.SendData(ctx, &livekit.SendDataRequest{
		Room: room,
		Data: data,
		Kind: livekit.DataPacket_RELIABLE,
	})
	return err
}

// studentPublishSources — studentga "so'zga ruxsat" berilganda ochiladigan manbalar.
// FAQAT kamera va mikrofon: SCREEN_SHARE ataylab YO'Q (webinar modeli — ekranni
// faqat ustoz ulashadi; aks holda student darsni buzish vektoriga ega bo'ladi).
// Diqqat: LiveKit'da bo'sh CanPublishSources ro'yxati "BARCHA manbalar" degani —
// shuning uchun ro'yxat aniq berilishi SHART.
var studentPublishSources = []livekit.TrackSource{
	livekit.TrackSource_CAMERA,
	livekit.TrackSource_MICROPHONE,
}

// SetParticipantPublish ishtirokchining media publish huquqini o'zgartiradi
// (token qayta chiqarmasdan). Webinar'da "qo'l ko'targan" studentga so'zga ruxsat berish uchun.
// Ustoz (host) huquqlari bu yerdan emas, token'dan keladi (token.go) — ekran ulashish
// ustozda saqlanadi.
func (c *Client) SetParticipantPublish(ctx context.Context, room, identity string, canPublish bool) error {
	_, err := c.room.UpdateParticipant(ctx, &livekit.UpdateParticipantRequest{
		Room:       room,
		Identity:   identity,
		Permission: studentPermission(canPublish),
	})
	return err
}

// studentPermission — studentga beriladigan huquqlar to'plami (test qamrovi uchun ajratilgan).
func studentPermission(canPublish bool) *livekit.ParticipantPermission {
	return &livekit.ParticipantPermission{
		CanSubscribe: true,
		CanPublish:   canPublish,
		// Ruxsat olib tashlanganda ham ro'yxatni beramiz: CanPublish=false bo'lsa
		// manbalar ahamiyatsiz, lekin keyinchalik faqat CanPublish o'zgartirilsa
		// "hammasi ochiq" holatiga qaytib qolish xavfi bo'lmasin.
		CanPublishSources: studentPublishSources,
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
