// Package livekit — LiveKit SFU media server bilan integratsiya.
// Backend faqat (1) xona yaratadi/o'chiradi, (2) kirish tokeni (JWT) generatsiya qiladi.
// Media oqimini LiveKit serverning o'zi boshqaradi.
package livekit

import (
	"net"
	"net/url"
	"strings"
	"time"

	lksdk "github.com/livekit/server-sdk-go/v2"

	"github.com/zoom/darsly/internal/pkg/config"
)

type Client struct {
	apiKey    string
	apiSecret string
	wsURL     string
	clientURL string // ws://... — klient (brauzer) shu manzilga ulanadi
	room      *lksdk.RoomServiceClient
	egress    *lksdk.EgressClient
	tokenTTL  time.Duration // kirish tokeni amal qilish muddati (join/reconnect gate)
	layout    string        // Egress kompozitsiya shabloni (qarang: egress.go normalizeLayout)
	canvas    int           // Yozuv kadrining tomoni (kvadrat) — qarang: egress.go recCanvasSide
	fps       int           // Yozuv kadr chastotasi — qarang: egress.go recFramerate
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
		clientURL: cfg.ClientURL,
		room:      lksdk.NewRoomServiceClient(httpURL, cfg.APIKey, cfg.APISecret),
		egress:    lksdk.NewEgressClient(httpURL, cfg.APIKey, cfg.APISecret),
		tokenTTL:  ttl,
		layout:    normalizeLayout(cfg.EgressLayout),
		canvas:    cfg.RecordingCanvas,
		fps:       cfg.RecordingFPS,
		enabled:   true,
	}
}

// Enabled — LiveKit sozlanganmi.
func (c *Client) Enabled() bool { return c.enabled }

// WSURL — brauzer klienti ulanadigan signaling manzili (ws://host).
func (c *Client) WSURL() string { return c.wsURL }

// ClientWSURL — KLIENTGA (telefon/brauzer) beriladigan signaling manzili.
//
// [LiveKitConfig.ClientURL] rejimlari:
//   - ""     → [WSURL] (backend va klient bir xil manzil ishlatadi);
//   - URL    → o'sha URL o'zgarishsiz (production: wss://livekit.<domen>);
//   - "auto" → sxema+port [WSURL] dan, host esa klientning API so'rovidagi
//     hostdan. Telefon http://10.x.x.x:8087 desa ws://10.x.x.x:7880 oladi,
//     brauzer localhost desa ws://localhost:7880 — dev'da tarmoq IP o'zgarsa
//     ham hech narsa sozlanmaydi. requestHost bo'sh bo'lsa xavfsiz fallback
//     [WSURL] (masalan, HTTP kontekstisiz chaqiruvlar).
func (c *Client) ClientWSURL(requestHost string) string {
	switch c.clientURL {
	case "":
		return c.wsURL
	case "auto":
		host := requestHost
		if h, _, err := net.SplitHostPort(requestHost); err == nil {
			host = h
		}
		if host == "" {
			return c.wsURL
		}
		u, err := url.Parse(c.wsURL)
		if err != nil {
			return c.wsURL
		}
		port := u.Port()
		if port == "" {
			return u.Scheme + "://" + host
		}
		return u.Scheme + "://" + net.JoinHostPort(host, port)
	default:
		return c.clientURL
	}
}

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
