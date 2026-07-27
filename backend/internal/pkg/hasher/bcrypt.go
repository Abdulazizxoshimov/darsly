package hasher

import (
	"runtime"

	"golang.org/x/crypto/bcrypt"
)

// Hasher is the interface for password hashing and verification.
type Hasher interface {
	Hash(password string) (string, error)
	Check(plain, hashed string) bool
}

type bcryptHasher struct {
	cost int
	// sem — bcrypt CPU-bound; bir vaqtda ishlaydigan hisoblashlar sonini cheklaydi.
	// 1000 talaba bir vaqtda login qilsa, cheklovsiz 1000 goroutine barcha yadroni
	// ~30s band qilib butun serverni bo'g'ardi. Semaphore bilan bir vaqtda ≤NumCPU
	// bcrypt ishlaydi — qolganlar navbatda kutadi, server javob berishda davom etadi.
	sem chan struct{}
}

// New returns a Hasher backed by bcrypt with the given cost.
// Cost 10-11 tavsiya etiladi (login/join hot-path'da CPU muvozanati uchun).
func New(cost int) Hasher {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = bcrypt.DefaultCost
	}
	n := max(runtime.NumCPU(), 1)
	return &bcryptHasher{cost: cost, sem: make(chan struct{}, n)}
}

func (h *bcryptHasher) Hash(password string) (string, error) {
	h.sem <- struct{}{}
	defer func() { <-h.sem }()
	b, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Check returns true if plain matches the stored hash. Cost saqlangan hash ichida
// bo'lgani uchun eski (yuqori-cost) hashlar ham to'g'ri tekshiriladi.
func (h *bcryptHasher) Check(plain, hashed string) bool {
	h.sem <- struct{}{}
	defer func() { <-h.sem }()
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain)) == nil
}
