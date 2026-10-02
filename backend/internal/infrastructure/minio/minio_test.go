package minio_test

// B-7 — MinIO presigned URL: TTL chegarasi + PUBLIC endpoint'ga imzolash.
//
// Ikki xossa mixlanadi:
//  (1) TTL — havola vaqtinchalik (X-Amz-Expires) va cheksiz EMAS (7 kun AWS chegarasi);
//  (2) imzo PUBLIC endpoint host'iga tushadi — aks holda tashqi qurilma
//      (telefon) ichki `minio:9000` manziliga yeta olmay yozuvni yuklab ololmasdi.
//
// minio-go presign paytida bucket-region'ni aniqlash uchun tarmoqqa chiqadi,
// shuning uchun bu test ishlab turgan dev MinIO'ni (localhost:9020) TALAB qiladi
// va yo'q bo'lsa `SkipOrFail` bilan skip bo'ladi (CI'da TEST_REQUIRE_INFRA yiqitadi).

import (
	"context"
	"net"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/infrastructure/minio"
	"github.com/zoom/darsly/internal/pkg/config"
	"github.com/zoom/darsly/internal/testutil"
)

func minioEndpoint() string {
	if v := os.Getenv("TEST_MINIO_ENDPOINT"); v != "" {
		return v
	}
	return "localhost:9020"
}

// minioPublicEndpoint — ichki endpoint bilan AYNI serverga (bir xil PORT), lekin
// boshqa host spelling (127.0.0.1) bilan: presign tashqi hostga imzolanishini tekshirish
// uchun ikkala host satri farqli, lekin ikkalasi ham ishlab turgan MinIO'ga yetadi.
// Dev: localhost:9020 → 127.0.0.1:9020; CI: localhost:9000 → 127.0.0.1:9000
// (avval port 9020 hardcode edi → CI minio 9000'da bo'lgani uchun region-lookup uzilardi).
func minioPublicEndpoint() string {
	_, port, err := net.SplitHostPort(minioEndpoint())
	if err != nil {
		return "127.0.0.1:9020"
	}
	return net.JoinHostPort("127.0.0.1", port)
}

// realClient — dev MinIO'ga ulangan klient. Yetib bo'lmasa test skip/fail.
// publicEndpoint bo'sh bo'lsa presign ichki endpointga tushadi.
func realClient(t *testing.T, publicEndpoint string) minio.Client {
	t.Helper()
	cfg := config.MinioConfig{
		Endpoint:       minioEndpoint(),
		AccessKey:      "minioadmin",
		SecretKey:      "minioadmin",
		Bucket:         "darsly",
		UseSSL:         false,
		PublicEndpoint: publicEndpoint,
		PublicUseSSL:   false, // dev MinIO http (https location-lookup'ni yiqitardi)
	}
	c, err := minio.New(cfg)
	require.NoError(t, err)
	// Reachability + bucket (region lookup presign uchun kerak).
	if err := c.EnsureBucket(context.Background()); err != nil {
		testutil.SkipOrFail(t, "MinIO mavjud emas (%s): %v", minioEndpoint(), err)
	}
	return c
}

// ⭐ Presigned GET havola PUBLIC endpoint host'iga imzolanadi (ichki emas).
// Bug: ichki hostga imzolansa tashqi qurilma yozuvni yuklab ololmaydi.
func TestPresignedURL_UsesPublicEndpoint(t *testing.T) {
	// Ichki va public AYNI MinIO'ga yetadi, lekin host SATRLARI farqli (qaysi klient
	// imzolagani ko'rinadi). Port minio endpoint'idan olinadi (dev 9020 / CI 9000).
	pub := minioPublicEndpoint()
	c := realClient(t, pub)

	raw, err := c.PresignedURL(context.Background(), "recordings/x.mp4", time.Hour)
	require.NoError(t, err)
	u, err := url.Parse(raw)
	require.NoError(t, err)
	require.Equal(t, pub, u.Host, "havola PUBLIC (tashqi) hostga imzolanishi kerak, ichkiga emas")
	require.Contains(t, u.Path, "recordings/x.mp4")
}

// Public endpoint sozlanmagan bo'lsa — ichki endpoint ishlatiladi (orqaga moslik).
func TestPresignedURL_FallsBackToInternalEndpoint(t *testing.T) {
	c := realClient(t, "") // public yo'q
	raw, err := c.PresignedURL(context.Background(), "recordings/x.mp4", time.Hour)
	require.NoError(t, err)
	u, err := url.Parse(raw)
	require.NoError(t, err)
	require.Equal(t, minioEndpoint(), u.Host, "public bo'lmasa ichki endpoint ishlatilishi kerak")
}

// ⭐ TTL havolaga yoziladi va CHEKSIZ emas: 1 soat → X-Amz-Expires=3600
// (recording.downloadTTL bilan mos), 7 kundan uzun TTL esa rad etiladi.
func TestPresignedURL_TTLIsBoundedAndSigned(t *testing.T) {
	c := realClient(t, "")
	ctx := context.Background()

	raw, err := c.PresignedURL(ctx, "recordings/x.mp4", time.Hour)
	require.NoError(t, err)
	u, err := url.Parse(raw)
	require.NoError(t, err)
	require.Equal(t, "3600", u.Query().Get("X-Amz-Expires"),
		"1 soatlik download TTL X-Amz-Expires=3600 bo'lib imzolanishi kerak")

	// 8 kun (> 7 kun AWS max) → xato. TTL cheksiz bo'lolmasligini isbotlaydi.
	_, err = c.PresignedURL(ctx, "recordings/x.mp4", 8*24*time.Hour)
	require.Error(t, err, "7 kundan uzun TTL rad etilishi kerak (havola cheksiz bo'lolmaydi)")
}

// PUT (client upload) havola ham public endpointga imzolanadi.
func TestPresignedPutURL_UsesPublicEndpoint(t *testing.T) {
	pub := minioPublicEndpoint()
	c := realClient(t, pub)
	raw, err := c.PresignedPutURL(context.Background(), "uploads/rec.mp4", time.Hour)
	require.NoError(t, err)
	u, err := url.Parse(raw)
	require.NoError(t, err)
	require.Equal(t, pub, u.Host, "PUT havola ham tashqi hostga imzolanishi kerak")
}
