// Package audit — xavfsizlik-muhim harakatlarni strukturaviy audit log sifatida yozadi.
// Loki/Grafana'da `event="audit"` bo'yicha so'ralishi mumkin (kim, nima, qachon).
package audit

import (
	"context"

	"github.com/zoom/darsly/internal/pkg/logger"
)

// Record bitta audit hodisasini yozadi.
//
//	action  — harakat nomi, masalan "waitingroom.admit"
//	actorID — harakatni bajargan foydalanuvchi (bo'sh bo'lishi mumkin — tizim)
//	fields  — qo'shimcha kontekst (target_id, lesson_id, ...)
func Record(ctx context.Context, log logger.Logger, action, actorID string, fields ...logger.Field) {
	base := []logger.Field{
		logger.String("event", "audit"),
		logger.String("action", action),
		logger.String("actor_id", actorID),
	}
	// Warn darajasi ATAYLAB: Loki core faqat Warn+ yuboradi (logger.WithLoki), Info esa
	// faqat stdout'da qolardi va audit izi markaziy tizimga yetmasdi. Audit yozuvlari
	// xavfsizlik uchun muhim — shuning uchun Warn (xato degani emas, yetkazish kafolati).
	log.Warn(ctx, "audit", append(base, fields...)...)
}
