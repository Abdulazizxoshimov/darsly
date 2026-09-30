package v1

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zoom/darsly/internal/testutil"
)

// /ready natijasi TTL ichida keshlanadi: tekshiruv (PG/Redis/MinIO ping) qayta chaqirilmaydi.
func TestReadyCheck_CachesResult(t *testing.T) {
	gin.SetMode(gin.TestMode)
	calls := 0
	var failing bool
	clock := time.Unix(1000, 0)
	h := newReadyHandler(func() error {
		calls++
		if failing {
			return errors.New("down")
		}
		return nil
	}, testutil.NewLogger(), 2*time.Second, func() time.Time { return clock })

	hit := func() int {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/ready", nil)
		h(c)
		return w.Code
	}

	if hit() != 200 || hit() != 200 || calls != 1 {
		t.Fatalf("kesh ishlamadi: calls=%d", calls)
	}
	failing = true
	if hit() != 200 || calls != 1 {
		t.Fatalf("TTL ichida qayta tekshirilmasligi kerak, calls=%d", calls)
	}
	clock = clock.Add(3 * time.Second)
	if hit() != 503 || calls != 2 {
		t.Fatalf("TTL o'tgach qayta tekshirilishi kerak, calls=%d", calls)
	}
}
