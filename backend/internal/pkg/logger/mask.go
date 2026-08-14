package logger

import (
	"regexp"
	"strings"
)

var (
	emailRe    = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
	phoneRe    = regexp.MustCompile(`(\+?\d[\d\s\-().]{7,}\d)`)
	tokenRe    = regexp.MustCompile(`(?i)(bearer\s+)[A-Za-z0-9\-._~+/]+=*`)
	passwordRe = regexp.MustCompile(`(?i)("password"\s*:\s*")[^"]*"`)
	cardRe     = regexp.MustCompile(`\b(?:\d[ -]?){15,16}\b`)
)

// MaskPII redacts sensitive patterns from a free-form string.
// Foydalanuvchi kiritgan istalgan matnni logga yozishdan oldin shu funksiyadan o'tkazing.
//
//	logger.SafeString("input", userText)  // ichida MaskPII ishlatiladi
func MaskPII(s string) string {
	s = emailRe.ReplaceAllStringFunc(s, func(m string) string {
		parts := strings.SplitN(m, "@", 2)
		if len(parts[0]) == 0 {
			return "***@" + parts[1]
		}
		return string(parts[0][0]) + "***@" + parts[1]
	})
	s = phoneRe.ReplaceAllString(s, "***PHONE***")
	s = tokenRe.ReplaceAllString(s, "${1}***REDACTED***")
	s = passwordRe.ReplaceAllString(s, `"password":"***REDACTED***"`)
	s = cardRe.ReplaceAllString(s, "***CARD***")
	return s
}

// MaskEmail keeps only the first character and domain.
//
//	MaskEmail("john@example.com")  →  "j***@example.com"
func MaskEmail(email string) string {
	if email == "" {
		return ""
	}
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 || len(parts[0]) == 0 {
		return "***"
	}
	return string(parts[0][0]) + "***@" + parts[1]
}
