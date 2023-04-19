package ecdsa

import (
	"crypto/ecdsa"
	"crypto/x509"
	"github.com/domsolutions/gopayloader/pkgs/jwt-signer/definition"
	"github.com/golang-jwt/jwt"
	"github.com/pterm/pterm"
)

type SignerECDSA struct {
	pubKey  *x509.Certificate
	kid     string
	privKey *ecdsa.PrivateKey
}

func CreateSigner(privKey []byte, kid string) (definition.Signer, error) {
	ecdsaKey, err := jwt.ParseECPrivateKeyFromPEM(privKey)
	if err != nil {
		pterm.Debug.Printf("Failed to parse ECDSA private key %v", err)
		return nil, err
	}

	s := &SignerECDSA{
		kid:     kid,
		privKey: ecdsaKey,
	}
	claim := make(jwt.MapClaims)
	claim["test"] = true
	if _, err := s.Generate(claim); err != nil {
		pterm.Debug.Printf("Failed to generate ECDSA jwt %v", err)
		return nil, err
	}
	return s, nil
}

func (e *SignerECDSA) Generate(claims jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = e.kid

	t, err := token.SignedString(e.privKey)
	if err != nil {
		return "", err
	}
	return t, nil
}
