package casbin

import (
	_ "embed"
	"fmt"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	stringadapter "github.com/casbin/casbin/v2/persist/string-adapter"
)

// RBAC modeli va siyosati BINARGA JOYLASHTIRILGAN (`go:embed`).
//
// # Nega diskdan o'qish tashlandi
//
// Avval ular `internal/pkg/casbin/model.conf` degan NISBIY yo'l bilan
// yuklanardi. Bu ilova FAQAT `backend/` katalogidan ishga tushirilgandagina
// ishlaydi degani edi: boshqa cwd'dan (systemd `WorkingDirectory` boshqa
// bo'lsa, konteynerda ish katalogi o'zgarsa, yoki `go run` boshqa joydan
// chaqirilsa) enforcer yasalmasdi va butun RBAC qatlami ishga tushmasdi —
// ya'ni himoya joylashtirish tafsilotiga bog'liq edi.
//
// Embed bilan bu sinf muammosi yo'qoladi: fayllar kompilyatsiya vaqtida
// binarga kiradi va ishga tushirish joyi ahamiyatsiz bo'ladi. Yon foyda:
// siyosat fayli deploy'da tushib qolishi ham mumkin emas.
var (
	//go:embed model.conf
	modelConf string
	//go:embed policy.csv
	policyCSV string
)

// NewEnforcer joylashtirilgan model va siyosat bilan tayyor enforcer qaytaradi.
func NewEnforcer() (*casbin.Enforcer, error) {
	m, err := model.NewModelFromString(modelConf)
	if err != nil {
		return nil, fmt.Errorf("casbin: model parse: %w", err)
	}
	e, err := casbin.NewEnforcer(m, stringadapter.NewAdapter(policyCSV))
	if err != nil {
		return nil, fmt.Errorf("casbin: enforcer: %w", err)
	}
	return e, nil
}

// NewEnforcerFromFiles — diskdan yuklash. Faqat maxsus holatlar uchun:
// siyosatni qayta build qilmasdan almashtirish kerak bo'lsa (masalan incident
// paytida operatsion tuzatish). Odatiy yo'l — [NewEnforcer].
func NewEnforcerFromFiles(modelPath, policyPath string) (*casbin.Enforcer, error) {
	return casbin.NewEnforcer(modelPath, policyPath)
}
