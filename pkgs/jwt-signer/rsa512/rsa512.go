package rsa256

import (
	"crypto/rsa"
	"github.com/domsolutions/gopayloader/pkgs/jwt-signer/definition"
	"github.com/golang-jwt/jwt"
	"github.com/pterm/pterm"
)

type SignerRSA512 struct {
	kid     string
	privKey *rsa.PrivateKey
}

func Signer(privKey []byte, kid string) (definition.Signer, error) {
	rsaKey, err := jwt.ParseRSAPrivateKeyFromPEM(privKey)
	if err != nil {
		pterm.Debug.Printf("Failed to parse RSA512 private key %v", err)
		return nil, err
	}

	s := &SignerRSA512{
		kid:     kid,
		privKey: rsaKey,
	}
	claim := make(jwt.MapClaims)
	claim["test"] = true
	if _, err := s.Generate(claim); err != nil {
		pterm.Debug.Printf("Failed to generate RSA512 jwt %v", err)
		return nil, err
	}
	return s, nil
}

func (e *SignerRSA512) Generate(claims jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS512, claims)
	token.Header["kid"] = e.kid

	t, err := token.SignedString(e.privKey)
	if err != nil {
		return "", err
	}
	return t, nil
}
