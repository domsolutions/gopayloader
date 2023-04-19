package definition

import "github.com/golang-jwt/jwt"

type Signer interface {
	Generate(claims jwt.MapClaims) (string, error)
}
