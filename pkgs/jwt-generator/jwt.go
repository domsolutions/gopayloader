package jwt_generator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	jwt_signer "github.com/domsolutions/gopayloader/pkgs/jwt-signer"
	"github.com/domsolutions/gopayloader/pkgs/jwt-signer/definition"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/pterm/pterm"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const (
	batchSize = 1000000
)

type Config struct {
	Ctx        context.Context
	Kid        string
	JwtKeyPath string
	jwtKeyBlob []byte
	JwtSub     string
	JwtIss     string
	JwtAud     string
	signer     definition.Signer
}

type JWTGenerator struct {
	config *Config
}

func NewJWTGenerator(config *Config) *JWTGenerator {
	return &JWTGenerator{config: config}
}

func (c *Config) validate() error {
	jwtKey, err := os.ReadFile(c.JwtKeyPath)
	if err != nil {
		return err
	}
	signer, err := jwt_signer.CreateSigner(jwtKey, c.Kid)
	if err != nil {
		return err
	}
	c.signer = signer
	c.jwtKeyBlob = jwtKey
	return nil
}

func (j *JWTGenerator) getFileName(dir string) string {
	hash := sha256.New()
	hash.Write([]byte(j.config.JwtAud))
	hash.Write([]byte(j.config.JwtIss))
	hash.Write([]byte(j.config.JwtSub))
	hash.Write(j.config.jwtKeyBlob)
	hash.Write([]byte(j.config.Kid))
	return filepath.Join(dir, "gopayloader-jwtstore-"+hex.EncodeToString(hash.Sum(nil))+".txt")
}

func (j *JWTGenerator) Generate(reqJwtCount int64, dir string, retry bool) error {
	if err := j.config.validate(); err != nil {
		return err
	}

	fname := j.getFileName(dir)
	f, err := os.OpenFile(fname, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("jwt: failed to create/open file to store jwts; %v", err)
	}
	defer f.Close()
	cache, err := newCache(f)
	if err != nil {
		if retry {
			return err
		}
		f.Close()
		pterm.Error.Printf("jwt cache %s file corrupt, attempting to delete and recreate", fname)
		if err := os.Remove(fname); err != nil {
			pterm.Error.Printf("Couldn't remove cache file %s; %v", fname, err)
			return err
		}
		return j.Generate(reqJwtCount, dir, true)
	}

	if err := j.batchGenSave(reqJwtCount, batchSize, cache); err != nil {
		return err
	}
	return nil
}

func (j *JWTGenerator) batchGenSave(reqJwtAmount, batchSize int64, cache *cache) error {
	toGenerate := reqJwtAmount - cache.getJwtCount()
	if toGenerate == 0 {
		pterm.Debug.Println("No JWTs to generate, enough in cache")
		return nil
	}

	var limit = toGenerate
	if limit > batchSize {
		limit = batchSize
	}
	workers := runtime.NumCPU()
	jobs := limit / int64(workers)

	errs := make(chan error)
	resp := make(chan []string, workers)

	pterm.Debug.Printf("Generating %d JWTs in batch\n", limit)
	for i := 0; i < workers; i++ {
		if i == 0 {
			go j.generate(jobs+(limit%int64(workers)), errs, resp)
			continue
		}
		go j.generate(jobs, errs, resp)
	}

	tokens := make([]string, limit, limit)

	for i := 0; i < workers; i++ {
		select {
		case <-j.config.Ctx.Done():
			// user cancelled
			return errors.New("jwt generation cancelled")
		case err := <-errs:
			return err
		case tokens = <-resp:
			if len(tokens) == 0 {
				continue
			}
			pterm.Debug.Printf("Finished batch %d saving to disk\n", len(tokens))
			if err := cache.save(tokens); err != nil {
				return err
			}
			tokens = tokens[:0]
		}
	}

	if cache.getJwtCount() == reqJwtAmount {
		// all jwts generated
		return nil
	}

	return j.batchGenSave(reqJwtAmount, batchSize, cache)
}

func (j *JWTGenerator) generate(limit int64, errs chan<- error, response chan<- []string) {
	tokens := make([]string, limit, limit)
	var err error
	var i int64 = 0

	for i = 0; i < limit; i++ {
		claims := j.commonClaims()
		claims["jti"] = uuid.New().String()
		tokens[i], err = j.config.signer.Generate(claims)
		if err != nil {
			errs <- err
			return
		}
	}
	response <- tokens
}

func (j *JWTGenerator) commonClaims() jwt.MapClaims {
	claims := make(jwt.MapClaims)
	if j.config.JwtAud != "" {
		claims["aud"] = j.config.JwtAud
	}
	if j.config.JwtSub != "" {
		claims["sub"] = j.config.JwtSub
	}
	if j.config.JwtIss != "" {
		claims["iss"] = j.config.JwtIss
	}
	claims["exp"] = time.Now().Add(24 * time.Hour * 365).Unix()
	return claims
}
