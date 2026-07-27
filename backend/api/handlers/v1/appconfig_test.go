package v1_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/api/handlers"
	appv1 "github.com/zoom/darsly/api/handlers/v1"
	"github.com/zoom/darsly/internal/entity"
)

// BE-5: ochiq /app-config endpointi. Mobil klient (side-load APK) shu javobga
// qarab majburiy yangilashni amalga oshiradi — shakl va maydon nomlari kontrakt.
func TestGetAppConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := &handlers.Handler{
		AppConfig: entity.AppConfig{
			Android: entity.AppPlatformConfig{
				MinVersion:    "1.0.0",
				LatestVersion: "1.2.0",
				APKURL:        "https://app.194.163.139.242.sslip.io/download/darsly-mentor.apk",
				ForceUpdate:   true,
				ReleaseNotes:  "Kamera tuzatildi",
			},
		},
	}

	r := gin.New()
	r.GET("/api/v1/app-config", appv1.GetAppConfig(h))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/app-config", nil))

	require.Equal(t, http.StatusOK, w.Code)

	var got struct {
		Data struct {
			Android struct {
				MinVersion    string `json:"min_version"`
				LatestVersion string `json:"latest_version"`
				APKURL        string `json:"apk_url"`
				ForceUpdate   bool   `json:"force_update"`
				ReleaseNotes  string `json:"release_notes"`
			} `json:"android"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))

	require.Equal(t, "1.0.0", got.Data.Android.MinVersion)
	require.Equal(t, "1.2.0", got.Data.Android.LatestVersion)
	require.Equal(t, "https://app.194.163.139.242.sslip.io/download/darsly-mentor.apk", got.Data.Android.APKURL)
	require.True(t, got.Data.Android.ForceUpdate)
	require.Equal(t, "Kamera tuzatildi", got.Data.Android.ReleaseNotes)
}

// Endpoint auth'siz bo'lgani uchun (router'da `protected` guruhidan tashqarida)
// token bo'lmasa ham 200 qaytishi kerak — bo'sh config'da ham crash bo'lmaydi.
func TestGetAppConfig_NoAuthAndEmptyConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/api/v1/app-config", appv1.GetAppConfig(&handlers.Handler{}))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/app-config", nil))

	require.Equal(t, http.StatusOK, w.Code)
	require.JSONEq(t,
		`{"data":{"android":{"min_version":"","latest_version":"","apk_url":"","force_update":false,"release_notes":""}}}`,
		w.Body.String())
}
