package archive

import (
	"testing"

	"github.com/zoom/darsly/internal/entity"
)

func TestTelegramArchiveURL(t *testing.T) {
	mid := int64(42)
	super := int64(-1002287699171) // superguruh: -100 + 2287699171
	reg := int64(-4123456)         // oddiy guruh (-100 prefiksi yo'q)
	zero := int64(0)

	cases := []struct {
		name string
		rec  *entity.Recording
		want string // "" → nil kutiladi
	}{
		{"superguruh", &entity.Recording{TelegramMessageID: &mid, TelegramChatID: &super}, "https://t.me/c/2287699171/42"},
		{"xabar yo'q", &entity.Recording{TelegramChatID: &super}, ""},
		{"chat yo'q", &entity.Recording{TelegramMessageID: &mid}, ""},
		{"oddiy guruh", &entity.Recording{TelegramMessageID: &mid, TelegramChatID: &reg}, ""},
		{"musbat chat", &entity.Recording{TelegramMessageID: &mid, TelegramChatID: &zero}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := telegramArchiveURL(c.rec)
			if c.want == "" {
				if got != nil {
					t.Fatalf("nil kutilgan, oldi: %q", *got)
				}
				return
			}
			if got == nil || *got != c.want {
				t.Fatalf("kutilgan %q, oldi: %v", c.want, got)
			}
		})
	}
}
