package rsa256

import (
	"crypto/rsa"
	"github.com/domsolutions/gopayloader/pkgs/jwt-signer/definition"
	"github.com/golang-jwt/jwt"
	"github.com/pterm/pterm"
)

type SignerRSA256 struct {
	kid     string
	privKey *rsa.PrivateKey
}

func Signer(privKey []byte, kid string) (definition.Signer, error) {
	rsaKey, err := jwt.ParseRSAPrivateKeyFromPEM(privKey)
	if err != nil {
		pterm.Debug.Printf("Failed to parse RSA256 private key %v", err)
		return nil, err
	}

	s := &SignerRSA256{
		kid:     kid,
		privKey: rsaKey,
	}
	claim := make(jwt.MapClaims)
	claim["test"] = true
	if _, err := s.Generate(claim); err != nil {
		pterm.Debug.Printf("Failed to generate RSA256 jwt %v", err)
		return nil, err
	}
	return s, nil
}

func (e *SignerRSA256) Generate(claims jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = e.kid

	t, err := token.SignedString(e.privKey)
	if err != nil {
		return "", err
	}
	return t, nil
}
