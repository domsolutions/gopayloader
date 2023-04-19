package ecdsa

import (
	"crypto/ed25519"
	"github.com/domsolutions/gopayloader/pkgs/jwt-signer/definition"
	"github.com/golang-jwt/jwt"
	"github.com/pterm/pterm"
)

type SignerED25519 struct {
	kid     string
	privKey ed25519.PrivateKey
}

func CreateSigner(privKey []byte, kid string) (definition.Signer, error) {
	edKey, err := jwt.ParseEdPrivateKeyFromPEM(privKey)
	if err != nil {
		pterm.Debug.Printf("Failed to parse ED25519 private key %v", err)
		return nil, err
	}

	s := &SignerED25519{
		kid:     kid,
		privKey: edKey.(ed25519.PrivateKey),
	}
	claim := make(jwt.MapClaims)
	claim["test"] = true
	if _, err := s.Generate(claim); err != nil {
		pterm.Debug.Printf("Failed to generate ED25519 jwt %v", err)
		return nil, err
	}
	return s, nil
}

func (e *SignerED25519) Generate(claims jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	token.Header["kid"] = e.kid

	t, err := token.SignedString(e.privKey)
	if err != nil {
		return "", err
	}
	return t, nil
}
