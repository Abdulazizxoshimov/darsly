package token

import "github.com/golang-jwt/jwt/v5"

// Claims holds the parsed, typed fields extracted from a validated JWT.
type Claims struct {
	Sub       string
	SessionID string
	Role      string
	JTI       string
}

func mapToClaims(raw jwt.MapClaims) *Claims {
	str := func(key string) string {
		v, _ := raw[key].(string)
		return v
	}
	return &Claims{
		Sub:       str("sub"),
		SessionID: str("sid"),
		Role:      str("role"),
		JTI:       str("jti"),
	}
}
