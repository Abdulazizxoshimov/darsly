package logger

import "testing"

func TestUseJSONEncoder_Production(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	if !useJSONEncoder() {
		t.Fatal("production'da JSON encoder bo'lishi kerak")
	}
}
