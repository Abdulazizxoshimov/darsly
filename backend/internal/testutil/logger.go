// Package testutil — testlar uchun in-memory faqe'lar va yordamchilar.
package testutil

import "github.com/zoom/darsly/internal/pkg/logger"

// NewLogger testlar uchun jim (error-level) logger qaytaradi.
func NewLogger() logger.Logger {
	return logger.New("error", "test", "test")
}
