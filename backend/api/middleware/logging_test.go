package middleware

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRedactQuery — H-3: jonli access JWT (?token=) va guest capability (?request_id=)
// loglarga plaintext tushmasligi kerak, qolgan paramlar (page/limit) qoladi.
func TestRedactQuery(t *testing.T) {
	cases := []struct {
		name        string
		raw         string
		mustRedact  []string // bu paramlar qiymati REDACTED bo'lishi kerak
		mustKeep    map[string]string
		wantNotHave string // xom qiymat umuman ko'rinmasligi kerak
	}{
		{
			name:        "token niqoblanadi",
			raw:         "token=eyJhbGciOiJIUzI1NiJ9.secret.sig",
			mustRedact:  []string{"token"},
			wantNotHave: "eyJhbGciOiJIUzI1NiJ9.secret.sig",
		},
		{
			name:        "request_id niqoblanadi",
			raw:         "request_id=550e8400-e29b-41d4-a716-446655440000",
			mustRedact:  []string{"request_id"},
			wantNotHave: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:       "oddiy paramlar qoladi",
			raw:        "page=2&limit=20&token=abc.def.ghi",
			mustRedact: []string{"token"},
			mustKeep:   map[string]string{"page": "2", "limit": "20"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := redactQuery(tc.raw)
			parsed, err := url.ParseQuery(got)
			require.NoError(t, err)
			for _, k := range tc.mustRedact {
				require.Equal(t, "REDACTED", parsed.Get(k), "%s niqoblanishi kerak", k)
			}
			for k, v := range tc.mustKeep {
				require.Equal(t, v, parsed.Get(k), "%s qolishi kerak", k)
			}
			if tc.wantNotHave != "" {
				require.NotContains(t, got, tc.wantNotHave, "xom sezgir qiymat loglanmasligi kerak")
			}
		})
	}
}

// TestRedactQuery_MalformedFailsSafe — parse xato bo'lsa butun query niqoblanadi.
func TestRedactQuery_MalformedFailsSafe(t *testing.T) {
	require.Equal(t, "REDACTED", redactQuery("token=abc%zz"))
}
