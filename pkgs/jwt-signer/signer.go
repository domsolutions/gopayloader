package jwt_signer

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"errors"
	"github.com/domsolutions/gopayloader/pkgs/jwt-signer/definition"
	"github.com/golang-jwt/jwt"
	"github.com/pterm/pterm"
)

func CreateSigner(privKey []byte, kid string) (definition.Signer, error) {
	var signer definition.Signer

	signer, err := createSigner[*ecdsa.PrivateKey](privKey, kid, jwt.ParseECPrivateKeyFromPEM, jwt.SigningMethodES256)
	if err == nil {
		return signer, nil
	}
	signer, err = createSigner[*ecdsa.PrivateKey](privKey, kid, jwt.ParseECPrivateKeyFromPEM, jwt.SigningMethodES384)
	if err == nil {
		return signer, nil
	}
	signer, err = createSigner[*ecdsa.PrivateKey](privKey, kid, jwt.ParseECPrivateKeyFromPEM, jwt.SigningMethodES512)
	if err == nil {
		return signer, nil
	}
	signer, err = createSigner[crypto.PrivateKey](privKey, kid, jwt.ParseEdPrivateKeyFromPEM, jwt.SigningMethodEdDSA)
	if err == nil {
		return signer, nil
	}
	signer, err = createSigner[*rsa.PrivateKey](privKey, kid, jwt.ParseRSAPrivateKeyFromPEM, jwt.SigningMethodRS512)
	if err == nil {
		return signer, nil
	}
	signer, err = createSigner[*rsa.PrivateKey](privKey, kid, jwt.ParseRSAPrivateKeyFromPEM, jwt.SigningMethodRS256)
	if err == nil {
		return signer, nil
	}
	signer, err = createSigner[*rsa.PrivateKey](privKey, kid, jwt.ParseRSAPrivateKeyFromPEM, jwt.SigningMethodRS384)
	if err == nil {
		return signer, nil
	}

	signer, err = createSigner[[]byte](privKey, kid, func(key []byte) ([]byte, error) {
		return key, nil
	}, jwt.SigningMethodHS256)
	if err == nil {
		return signer, nil
	}

	signer, err = createSigner[[]byte](privKey, kid, func(key []byte) ([]byte, error) {
		return key, nil
	}, jwt.SigningMethodHS384)
	if err == nil {
		return signer, nil
	}

	signer, err = createSigner[[]byte](privKey, kid, func(key []byte) ([]byte, error) {
		return key, nil
	}, jwt.SigningMethodHS512)
	if err == nil {
		return signer, nil
	}

	return nil, errors.New("no supported jwt signer")
}

type signer struct {
	kid     string
	privKey any
	method  jwt.SigningMethod
}

func (e *signer) Generate(claims jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(e.method, claims)
	token.Header["kid"] = e.kid

	t, err := token.SignedString(e.privKey)
	if err != nil {
		return "", err
	}
	return t, nil
}

func createSigner[k any](privKey []byte, kid string, genPrivKey func(key []byte) (k, error), method jwt.SigningMethod) (definition.Signer, error) {
	key, err := genPrivKey(privKey)
	if err != nil {
		pterm.Debug.Printf("Failed to parse private key %v", err)
		return nil, err
	}

	s := &signer{
		kid:     kid,
		privKey: key,
		method:  method,
	}
	claim := make(jwt.MapClaims)
	claim["test"] = true
	if _, err := s.Generate(claim); err != nil {
		pterm.Debug.Printf("Failed to generate jwt %v", err)
		return nil, err
	}
	return s, nil
}
