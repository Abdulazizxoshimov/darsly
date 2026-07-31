package shared

import "context"

// requestHostKey — HTTP so'rov Host sarlavhasi uchun kontekst kaliti.
//
// NEGA KONTEKST ORQALI: klientga qaytariladigan LiveKit manzili ("auto" rejimda)
// klient API'ga qaysi host bilan kelganiga bog'liq. Bu HTTP qatlamining bilimi,
// lekin token usecase ichida yasaladi — Host'ni parametr sifatida har bir
// interfeys imzosidan o'tkazish o'rniga kontekstda olib boriladi (u shundoq ham
// so'rov qamrovidagi qiymat). Qiymat bo'lmasa bo'sh satr qaytadi va
// livekit.ClientWSURL xavfsiz fallback ishlatadi.
type requestHostKey struct{}

// WithRequestHost so'rov kontekstiga Host'ni joylaydi (middleware chaqiradi).
func WithRequestHost(ctx context.Context, host string) context.Context {
	return context.WithValue(ctx, requestHostKey{}, host)
}

// RequestHost kontekstdagi Host'ni qaytaradi ("" — HTTP kontekstisiz chaqiruv).
func RequestHost(ctx context.Context) string {
	host, _ := ctx.Value(requestHostKey{}).(string)
	return host
}
