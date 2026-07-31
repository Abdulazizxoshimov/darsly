package postgres

import "testing"

// LIKE joker belgilari foydalanuvchi kiritmasidan CHIQIB ketmasligi kerak:
// aks holda qidiruvga `%` yozgan odam butun jadvalni oladi va `%_%_%` kabi
// naqshlar indeksni chetlab og'ir skanga aylanadi.
func TestSearchPattern(t *testing.T) {
	cases := []struct{ in, want string }{
		{"matematika", "%matematika%"},
		{"", "%%"},
		// Jokerlar oddiy belgiga aylanadi.
		{"%", `%\%%`},
		{"_", `%\_%`},
		{"100%", `%100\%%`},
		{"a_b", `%a\_b%`},
		// Teskari slash BIRINCHI ekranlanadi — aks holda keyingi
		// almashtirishlar qo'ygan `\` ikki marta ekranlanardi.
		{`a\b`, `%a\\b%`},
		{`\%`, `%\\\%%`},
	}
	for _, c := range cases {
		if got := SearchPattern(c.in); got != c.want {
			t.Errorf("SearchPattern(%q) = %q, kutilgan %q", c.in, got, c.want)
		}
	}
}
