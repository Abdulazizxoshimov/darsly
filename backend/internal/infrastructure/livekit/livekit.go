// Package livekit — LiveKit SFU media server bilan integratsiya.
// Backend faqat (1) xona yaratadi/o'chiradi, (2) kirish tokeni (JWT) generatsiya qiladi.
// Media oqimini LiveKit serverning o'zi boshqaradi.
package livekit

import (
	"strings"
	"time"

	lksdk "github.com/livekit/server-sdk-go/v2"

	"github.com/zoom/darsly/internal/pkg/config"
)

type Client struct {
	apiKey    string
	apiSecret string
	wsURL     string // ws://... — klient (brauzer) shu manzilga ulanadi
	room      *lksdk.RoomServiceClient
	egress    *lksdk.EgressClient
	tokenTTL  time.Duration // kirish tokeni amal qilish muddati (join/reconnect gate)
	layout    string        // Egress kompozitsiya shabloni (qarang: egress.go normalizeLayout)
	enabled   bool
}

// New LiveKit klientini yaratadi. API kalitlar bo'lmasa — o'chirilgan (nop) klient.
func New(cfg config.LiveKitConfig) *Client {
	if cfg.APIKey == "" || cfg.APISecret == "" || cfg.Host == "" {
		return &Client{enabled: false}
	}
	ttl := cfg.TokenTTL
	if ttl <= 0 {
		ttl = 6 * time.Hour // uzun darsda reconnect'da token o'tib ketmasin
	}
	httpURL := toHTTP(cfg.Host)
	return &Client{
		apiKey:    cfg.APIKey,
		apiSecret: cfg.APISecret,
		wsURL:     cfg.Host,
		room:      lksdk.NewRoomServiceClient(httpURL, cfg.APIKey, cfg.APISecret),
		egress:    lksdk.NewEgressClient(httpURL, cfg.APIKey, cfg.APISecret),
		tokenTTL:  ttl,
		layout:    normalizeLayout(cfg.EgressLayout),
		enabled:   true,
	}
}

// Enabled — LiveKit sozlanganmi.
func (c *Client) Enabled() bool { return c.enabled }

// WSURL — brauzer klienti ulanadigan signaling manzili (ws://host).
func (c *Client) WSURL() string { return c.wsURL }

// toHTTP RoomService API chaqiruvlari uchun ws→http, wss→https ga o'giradi.
func toHTTP(u string) string {
	switch {
	case strings.HasPrefix(u, "wss://"):
		return "https://" + strings.TrimPrefix(u, "wss://")
	case strings.HasPrefix(u, "ws://"):
		return "http://" + strings.TrimPrefix(u, "ws://")
	default:
		return u
	}
}
