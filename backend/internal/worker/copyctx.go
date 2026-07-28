package worker

import (
	"context"
	"io"
)

// copyCtx — kontekstga bo'ysunadigan `io.Copy`.
//
// Oddiy `io.Copy` bekor qilishni bilmaydi: server to'xtayotganda 1 GB fayl
// nusxalanishi oxirigacha davom etardi va graceful shutdown cho'zilardi.
func copyCtx(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	return io.Copy(dst, readerCtx{ctx: ctx, r: src})
}

type readerCtx struct {
	ctx context.Context
	r   io.Reader
}

func (r readerCtx) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}
