package livekit

import (
	"context"

	"github.com/livekit/protocol/livekit"
)

// EnsureRoom xonani yaratadi (idempotent — mavjud bo'lsa o'shani qaytaradi).
// emptyTimeoutSec — hamma chiqib ketgach xona qancha soniya ochiq turadi.
func (c *Client) EnsureRoom(ctx context.Context, name string, emptyTimeoutSec uint32) error {
	_, err := c.room.CreateRoom(ctx, &livekit.CreateRoomRequest{
		Name:         name,
		EmptyTimeout: emptyTimeoutSec,
	})
	return err
}

// DeleteRoom xonani yopadi va barcha ishtirokchilarni uzadi.
func (c *Client) DeleteRoom(ctx context.Context, name string) error {
	_, err := c.room.DeleteRoom(ctx, &livekit.DeleteRoomRequest{Room: name})
	return err
}

// ListParticipants xonadagi hozirgi ishtirokchilar ro'yxati.
func (c *Client) ListParticipants(ctx context.Context, name string) ([]*livekit.ParticipantInfo, error) {
	resp, err := c.room.ListParticipants(ctx, &livekit.ListParticipantsRequest{Room: name})
	if err != nil {
		return nil, err
	}
	return resp.Participants, nil
}
