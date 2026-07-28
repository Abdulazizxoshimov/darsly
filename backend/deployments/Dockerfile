FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/server ./cmd

# ─── Runtime ──────────────────────────────────────────────────────────────────
FROM alpine:3.19

# ffmpeg — dars yozuvini qayta kodlash uchun (`worker.TranscodeWorker`).
# Image ~80 MB kattalashadi, lekin buning evaziga 2.5 soatlik dars ~1 GB
# o'rniga ~300 MB bo'ladi. ffmpeg bo'lmasa ishchi o'zi ishga tushmaydi —
# ya'ni image'ni ataylab yengil qilish ham mumkin (yozuvlar kattaroq qoladi).
RUN apk add --no-cache ca-certificates tzdata ffmpeg && \
    addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /bin/server /app/server
COPY --from=builder /app/internal/pkg/casbin /app/internal/pkg/casbin/
COPY --from=builder /app/migrations /app/migrations

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/server"]
