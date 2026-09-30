package email

import (
	"context"
	"net"
	"testing"
	"time"
)

// Javob bermaydigan SMTP server worker'ni abadiy osib qo'ymasligi kerak.
func TestSendMailTimeout_StalledServerTimesOut(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_ = c // ulanadi, hech narsa yozmaydi (greeting yo'q)
		}
	}()

	old := smtpTotalBudget
	smtpTotalBudget = 300 * time.Millisecond
	defer func() { smtpTotalBudget = old }()

	start := time.Now()
	err = sendMailTimeout(context.Background(), ln.Addr().String(), "127.0.0.1", nil, "a@b.c", []string{"x@y.z"}, []byte("hi"))
	if err == nil {
		t.Fatal("timeout xatosi kutilgan")
	}
	if d := time.Since(start); d > 3*time.Second {
		t.Fatalf("juda uzoq kutdi: %v", d)
	}
}
