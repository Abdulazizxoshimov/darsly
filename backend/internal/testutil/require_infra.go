package testutil

import (
	"os"
	"testing"
)

// EnvRequireInfra — o'rnatilgan bo'lsa, infratuzilma (Postgres/Redis) yo'qligi
// testni SKIP emas, YIQITADI.
const EnvRequireInfra = "TEST_REQUIRE_INFRA"

// SkipOrFail — infratuzilmaga bog'liq testlar uchun yagona kirish nuqtasi.
//
// ## Nega bu funksiya bor
// Auditda (F-1) va keyin jonli ish jarayonida bir xil nosozlik ikki marta
// uchradi: Postgres yetib bo'lmaganda 50+ integratsiya testi `t.Skip` qilardi va
// paket baribir `ok` bo'lib ko'rinardi. Ya'ni "testlar o'tdi" degan xulosa
// YOLG'ON edi va buni faqat bajarilish vaqtiga qarab (14 s o'rniga 0.03 s)
// sezish mumkin edi.
//
// Skip'ning o'zi to'g'ri qaror — dasturchi Docker'siz ham `go test ./...`
// ishlata olishi kerak. Xato — bu qarorni CI'ga ham qo'llash. Endi CI
// `TEST_REQUIRE_INFRA=1` beradi va o'sha yerda skip TAQIQLANADI.
//
// Xabar aniq bo'lishi shart: yiqilgan CI'da sabab bir qarashda ko'rinsin.
func SkipOrFail(t *testing.T, format string, args ...any) {
	t.Helper()
	if os.Getenv(EnvRequireInfra) != "" {
		t.Fatalf("infratuzilma majburiy ("+EnvRequireInfra+" o'rnatilgan), lekin yetib bo'lmadi: "+format, args...)
	}
	t.Skipf(format, args...)
}
