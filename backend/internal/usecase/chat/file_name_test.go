package chat

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// Uzun ko'p-baytli nom rune chegarasida kesilishi va yaroqli UTF-8 bo'lib qolishi kerak
// (avval name[len-200:] belgini bo'lib, yaroqsiz UTF-8 -> 500 berardi).
func TestValidateFile_LongMultibyteNameStaysValidUTF8(t *testing.T) {
	name := strings.Repeat("я", 150) + ".png" // 304 bayt
	png := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 32)...)
	vf, err := validateFile(name, int64(len(png)), strings.NewReader(string(png)))
	if err != nil {
		t.Fatal(err)
	}
	if !utf8.ValidString(vf.name) {
		t.Fatalf("nom yaroqsiz UTF-8: %q", vf.name)
	}
	if len(vf.name) > 200 || !strings.HasSuffix(vf.name, ".png") {
		t.Fatalf("nom noto'g'ri kesildi: len=%d", len(vf.name))
	}
}
