package jwt_signer

import (
	"errors"
	"github.com/domsolutions/gopayloader/pkgs/jwt-signer/definition"
	"github.com/domsolutions/gopayloader/pkgs/jwt-signer/ecdsa"
	ed25519 "github.com/domsolutions/gopayloader/pkgs/jwt-signer/ed25519"
	"github.com/domsolutions/gopayloader/pkgs/jwt-signer/rsa256"
)

type SignerGenerator func(privKey []byte, kid string) (definition.Signer, error)

// TODO add more signers

var signers = []SignerGenerator{ecdsa.CreateSigner, ed25519.CreateSigner, rsa256.CreateSigner}

func CreateSigner(privKey []byte, kid string) (definition.Signer, error) {
	var signer definition.Signer
	var err error
	for _, s := range signers {
		signer, err = s(privKey, kid)
		if err != nil {
			continue
		}
		return signer, nil
	}

	return nil, errors.New("no supported jwt signer")
}
