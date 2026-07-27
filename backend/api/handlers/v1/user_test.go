package v1_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/api/handlers"
	userv1 "github.com/zoom/darsly/api/handlers/v1"
	"github.com/zoom/darsly/api/middleware"
	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/pkg/hasher"
	"github.com/zoom/darsly/internal/testutil"
	"github.com/zoom/darsly/internal/usecase/user"
)

// testUserID — haqiqiy foydalanuvchi ID'si UUID bo'ladi (uuid.NewString). Usecase
// endi ID shaklini tekshirgani uchun (yaroqsiz UUID → 404, 500 emas) testda ham
// realistik qiymat kerak.
const testUserID = "3f1a8c22-6b0e-4f3a-9c11-2d7e5a904f80"

// Privilege escalation himoyasi: student `PUT /users/me {"role":"mentor"}` bilan
// o'zini mentor qila OLMASLIGI kerak (handler Role'ni tozalaydi).
func TestUpdateCurrentUser_CannotEscalateRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	users := testutil.NewFakeUserRepo()
	require.NoError(t, users.Create(context.Background(), &entity.User{
		ID: testUserID, Email: "s@x.uz", FullName: "Student", Role: "student", IsActive: true,
	}))
	h := &handlers.Handler{User: user.New(users, hasher.New(4), testutil.NewFakeTokenMaker(), testutil.NewLogger())}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middleware.CtxUserID, testUserID)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/users/me",
		strings.NewReader(`{"full_name":"Hacker","role":"mentor"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	userv1.UpdateCurrentUser(h)(c)

	require.Equal(t, http.StatusOK, w.Code)
	u, err := users.GetByID(context.Background(), testUserID)
	require.NoError(t, err)
	require.Equal(t, "student", u.Role, "student o'zini mentor qila OLMASLIGI kerak")
	require.Equal(t, "Hacker", u.FullName, "boshqa maydonlar yangilanadi")
}
