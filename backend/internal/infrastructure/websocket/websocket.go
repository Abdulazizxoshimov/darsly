package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"github.com/zoom/darsly/internal/infrastructure/redis"
	"github.com/zoom/darsly/internal/pkg/logger"
	"github.com/zoom/darsly/internal/pkg/metrics"
)

// originPolicy — WS upgrade uchun Origin tekshirish siyosati.
// Env'dan o'qiladi (paketning mavjud uslubi — qarang: WS_MAX_CONNECTIONS).
type originPolicy struct {
	allowed    []string // ruxsat etilgan origin'lar (normallashtirilgan, bo'sh bo'lishi mumkin)
	allowEmpty bool     // Origin header umuman yo'q bo'lsa ruxsat berilsinmi
	isProd     bool
}

// loadOriginPolicy env'dan siyosatni yig'adi.
//
//	WS_ALLOWED_ORIGINS   — vergul bilan ajratilgan ro'yxat (ko'p frontend/domen uchun).
//	                       Bo'sh bo'lsa FRONTEND_BASE_URL ishlatiladi (orqaga moslik).
//	WS_ALLOW_EMPTY_ORIGIN — default true; native mobil klientlar uchun (pastdagi izohga qara).
func loadOriginPolicy() originPolicy {
	raw := os.Getenv("WS_ALLOWED_ORIGINS")
	if strings.TrimSpace(raw) == "" {
		// Orqaga moslik: eski sozlama bilan ishlayotgan deploy'lar buzilmasin.
		raw = os.Getenv("FRONTEND_BASE_URL")
	}
	allowed := make([]string, 0, 2)
	for _, part := range strings.Split(raw, ",") {
		if o := normalizeOrigin(part); o != "" {
			allowed = append(allowed, o)
		}
	}

	allowEmpty := true
	if v := os.Getenv("WS_ALLOW_EMPTY_ORIGIN"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			allowEmpty = b
		}
	}

	return originPolicy{
		allowed:    allowed,
		allowEmpty: allowEmpty,
		isProd:     os.Getenv("APP_ENV") == "production",
	}
}

// normalizeOrigin bo'shliq va oxirgi "/" ni olib tashlaydi ("https://a.uz/" == "https://a.uz").
func normalizeOrigin(s string) string {
	return strings.TrimRight(strings.TrimSpace(s), "/")
}

// allow — bitta Origin qiymatini siyosat bilan tekshiradi.
func (p originPolicy) allow(origin string) bool {
	if normalizeOrigin(origin) == "" {
		// Origin BO'SH — native mobil klient (Android/OkHttp, iOS/NSURLSession) WS
		// handshake'da `Origin` header yubormaydi. Brauzer esa WS uchun Origin'ni
		// HAR DOIM yuboradi (RFC 6455 §4.1, §10.2) — ya'ni bo'sh Origin brauzerdan
		// kelishi mumkin emas, demak CSWSH (cross-site WebSocket hijacking) xavfi yo'q.
		// Autentifikatsiya baribir `?token=` / Bearer JWT orqali bajariladi
		// (api/handlers/v1/ws.go, middleware.Auth), kutish xonasi esa request_id bilan.
		// Kimga qat'iyroq kerak bo'lsa: WS_ALLOW_EMPTY_ORIGIN=false.
		return p.allowEmpty
	}
	if len(p.allowed) == 0 {
		// Allowlist sozlanmagan: dev'da erkin, production'da rad (avvalgi xatti-harakat).
		return !p.isProd
	}
	origin = normalizeOrigin(origin)
	for _, a := range p.allowed {
		if strings.EqualFold(origin, a) {
			return true
		}
	}
	return false
}

func newUpgrader() websocket.Upgrader {
	policy := loadOriginPolicy()
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return policy.allow(r.Header.Get("Origin"))
		},
		Error: writeUpgradeError,
	}
}

// writeUpgradeError — upgrade rad javobini API'ning {code,message} formatiga keltiradi
// (gorilla default'i `text/plain` matn qaytaradi, klientlar esa JSON kutadi).
// apperr/http_status'ga bog'lanmaymiz: infrastructure qatlami api qatlamini import
// qilmasligi kerak — shakl qo'lda takrorlanadi (api/http_status/response.go: errorResponse).
func writeUpgradeError(w http.ResponseWriter, _ *http.Request, status int, _ error) {
	code, msg := "BAD_REQUEST", "websocket upgrade failed"
	switch status {
	case http.StatusForbidden:
		// Sabab tafsiloti berilmaydi (allowlist tarkibi sizib chiqmasin).
		code, msg = "FORBIDDEN", "websocket origin not allowed"
	case http.StatusMethodNotAllowed:
		code, msg = "BAD_REQUEST", "websocket upgrade requires GET"
	case http.StatusInternalServerError:
		code, msg = "INTERNAL_ERROR", "internal server error"
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Sec-Websocket-Version", "13")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": code, "message": msg})
}

// Hub manages all active WebSocket connections with room-based broadcasting.
// Ko'p-instans muhitida (horizontal scale) real-time xabarlar Redis pub/sub orqali
// barcha instanslarga tarqatiladi (EnableFanout). Fanout yoqilmasa — lokal-only.
type Hub struct {
	mu         sync.RWMutex
	clients    map[string]map[*client]struct{} // userID → connections
	rooms      map[string]map[*client]struct{} // room key → connections
	log        logger.Logger
	done       chan struct{}
	count      atomic.Int64 // faol ulanishlar soni (DoS cap uchun)
	maxClients int64        // 0 = cheksiz

	cache      redis.Cache // Redis fan-out (nil bo'lsa lokal-only)
	instanceID string      // shu instans ID'si (o'z xabarini takror yetkazmaslik uchun)
}

type client struct {
	userID string
	conn   *websocket.Conn
	send   chan []byte
	hub    *Hub
	rooms  map[string]struct{}
	roomMu sync.Mutex
}

func NewHub(log logger.Logger) *Hub {
	max := int64(10000) // default global ulanish chegarasi
	if v := os.Getenv("WS_MAX_CONNECTIONS"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			max = n
		}
	}
	return &Hub{
		clients:    make(map[string]map[*client]struct{}),
		rooms:      make(map[string]map[*client]struct{}),
		log:        log,
		done:       make(chan struct{}),
		maxClients: max,
	}
}

// Stop closes all connections cleanly. Call on server shutdown.
func (h *Hub) Stop() {
	close(h.done)
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, conns := range h.clients {
		for c := range conns {
			if c.conn == nil {
				continue
			}
			_ = c.conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, "server shutdown"))
			c.conn.Close()
		}
	}
}

// ServeWS upgrades an HTTP connection to WebSocket and registers the client.
func (h *Hub) ServeWS(ctx context.Context, w http.ResponseWriter, r *http.Request, userID string) {
	// Global ulanish chegarasi (DoS himoyasi) — upgrade'dan oldin tekshiriladi.
	if h.maxClients > 0 && h.count.Load() >= h.maxClients {
		http.Error(w, "too many active connections", http.StatusServiceUnavailable)
		return
	}

	upgrader := newUpgrader()
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Error(ctx, "ws: upgrade failed", logger.Error(err))
		metrics.WSConnections.WithLabelValues("upgrade_failed").Inc()
		return
	}
	metrics.WSConnections.WithLabelValues("opened").Inc()

	c := &client{
		userID: userID,
		conn:   conn,
		send:   make(chan []byte, 256),
		hub:    h,
		rooms:  make(map[string]struct{}),
	}
	h.register(c)
	defer func() {
		h.unregister(c)
		metrics.WSConnections.WithLabelValues("closed").Inc()
	}()

	go c.writePump(ctx)
	c.readPump(ctx)
}

// BroadcastToRoom sends a message to all clients subscribed to a room.
// Room format: "project:<id>", "issue:<id>", "page:<id>", "space:<id>", "user:<id>".
func (h *Hub) BroadcastToRoom(msg Message) {
	if msg.Room == "" {
		return
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	// RLock butun sikl davomida ushlanadi (send-on-closed / concurrent-map race'dan himoya).
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.rooms[msg.Room] {
		select {
		case c.send <- data:
		default:
			metrics.WSDropped.Inc() // slow client — drop rather than block
		}
	}
}

// Count returns the number of active WebSocket connections on this instance.
func (h *Hub) Count() int64 { return h.count.Load() }

// Online returns true if the user has at least one active connection.
func (h *Hub) Online(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID]) > 0
}

func (h *Hub) register(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[c.userID] == nil {
		h.clients[c.userID] = make(map[*client]struct{})
	}
	h.clients[c.userID][c] = struct{}{}
	h.count.Add(1)
}

func (h *Hub) unregister(c *client) {
	c.roomMu.Lock()
	rooms := make([]string, 0, len(c.rooms))
	for r := range c.rooms {
		rooms = append(rooms, r)
	}
	c.roomMu.Unlock()

	h.mu.Lock()
	defer h.mu.Unlock()

	for _, room := range rooms {
		if conns, ok := h.rooms[room]; ok {
			delete(conns, c)
			if len(conns) == 0 {
				delete(h.rooms, room)
			}
		}
	}

	conns, ok := h.clients[c.userID]
	if ok {
		if _, exists := conns[c]; exists {
			delete(conns, c)
			close(c.send)
			h.count.Add(-1)
		}
		if len(conns) == 0 {
			delete(h.clients, c.userID)
		}
	}
}

func (h *Hub) subscribeRoom(c *client, room string) {
	h.mu.Lock()
	if h.rooms[room] == nil {
		h.rooms[room] = make(map[*client]struct{})
	}
	h.rooms[room][c] = struct{}{}
	h.mu.Unlock()

	c.roomMu.Lock()
	c.rooms[room] = struct{}{}
	c.roomMu.Unlock()
}

func (h *Hub) unsubscribeRoom(c *client, room string) {
	h.mu.Lock()
	if conns, ok := h.rooms[room]; ok {
		delete(conns, c)
		if len(conns) == 0 {
			delete(h.rooms, room)
		}
	}
	h.mu.Unlock()

	c.roomMu.Lock()
	delete(c.rooms, room)
	c.roomMu.Unlock()
}

func (c *client) writePump(ctx context.Context) {
	ticker := time.NewTicker(50 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-ctx.Done():
			return
		case <-c.hub.done:
			return
		}
	}
}

func (c *client) readPump(ctx context.Context) {
	_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		var msg Message
		if jsonErr := json.Unmarshal(data, &msg); jsonErr != nil {
			continue
		}
		switch msg.Type {
		case TypeSubscribe:
			if msg.Room != "" {
				c.hub.subscribeRoom(c, msg.Room)
			}
		case TypeUnsubscribe:
			if msg.Room != "" {
				c.hub.unsubscribeRoom(c, msg.Room)
			}
		}
	}
}
