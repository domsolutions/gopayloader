package jwt_generator

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	config "github.com/domsolutions/gopayloader/config"
	jwt_signer "github.com/domsolutions/gopayloader/pkgs/jwt-signer"
	"github.com/domsolutions/gopayloader/pkgs/jwt-signer/definition"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/pterm/pterm"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	batchSize = 1000000
)

type Config struct {
	Ctx                 context.Context
	Kid                 string
	JwtKeyPath          string
	jwtKeyBlob          []byte
	JwtSub              string
	JwtCustomClaimsJSON string
	JwtIss              string
	JwtAud              string
	JwtsFilename        string
	signer              definition.Signer
	store               *cache
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

// GetUserSuppliedJWTs Gets a count number of JWTs from a file, reusing them if not enough exist to match count
func GetUserSuppliedJWTs(fname string, count int64) (<-chan string, <-chan error) {
	recv := make(chan string, 1000000)
	errs := make(chan error, 1)
	go getUserJWTS(fname, count, errs, recv)

	// give goroutine time to prime channel with jwts for workers
	time.Sleep(1 * time.Second)
	return recv, errs
}

func getUserJWTS(fname string, count int64, errs chan<- error, jwts chan<- string) {
	defer func() {
		close(errs)
		close(jwts)
	}()

	file, err := os.OpenFile(fname, os.O_RDONLY, os.ModePerm)
	if err != nil {
		errs <- fmt.Errorf("jwt_generator: retrieving; failed to open file containing JWTs; %v", err)
		return
	}
	defer file.Close()

	jwtsSent := int64(0)
	for jwtsSent < count {
		// Set pointer to beginning of file
		if _, err := file.Seek(0, 0); err != nil {
			errs <- err
			return
		}
		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanLines)

		fileValid := false
		for scanner.Scan() {
			fileValid = true
			jwts <- string(scanner.Bytes())
			jwtsSent++
			// Stop reading file when enough JWTs have been fetched
			if jwtsSent == count {
				return
			}
		}

		if !fileValid {
			errs <- fmt.Errorf("jwt_generator: retrieving; file doesn't contain a JWT")
			return
		}
		// Loops if user asked for more requests than there were JWTs in the file, so JWTs get reused
	}
}

func (j *JWTGenerator) getFileName(dir string) string {
	hash := sha256.New()
	hash.Write([]byte(j.config.JwtAud))
	hash.Write([]byte(j.config.JwtIss))
	hash.Write([]byte(j.config.JwtSub))
	hash.Write([]byte(j.config.JwtCustomClaimsJSON))
	strippedKey := strings.ReplaceAll(strings.ReplaceAll(string(j.config.jwtKeyBlob), "\r", ""), "\n", "") // Replace \r and \n to have the same value in Windows and Linux
	hash.Write([]byte(strippedKey))
	hash.Write([]byte(j.config.Kid))
	return filepath.Join(dir, "gopayloader-jwtstore-"+hex.EncodeToString(hash.Sum(nil))+".txt")
}

func (j *JWTGenerator) Generate(reqJwtCount int64, dir string, retrying bool) error {
	if err := j.config.validate(); err != nil {
		return err
	}

	fname := j.getFileName(dir)
	f, err := os.OpenFile(fname, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("jwt: failed to create/open file to store jwts; %v", err)
	}
	cache, err := newCache(f)
	if err != nil {
		if retrying {
			return err
		}
		f.Close()
		pterm.Error.Printf("jwt cache %s file corrupt, attempting to delete and recreate; got error; %v \n", fname, err)
		if err := os.Remove(fname); err != nil {
			pterm.Error.Printf("Couldn't remove cache file %s; %v", fname, err)
			return err
		}
		return j.Generate(reqJwtCount, dir, true)
	}
	j.config.store = cache
	if cache.count > 0 {
		pterm.Info.Printf("Found %d jwts in cache\n", cache.count)
	}

	if err := j.batchGenSave(reqJwtCount, batchSize); err != nil {
		return err
	}
	return nil
}

func (j *JWTGenerator) JWTS(count int64) (<-chan string, <-chan error) {
	return j.config.store.get(count)
}

func (j *JWTGenerator) batchGenSave(reqJwtAmount, batchSize int64) error {
	toGenerate := reqJwtAmount - j.config.store.getJwtCount()
	if toGenerate <= 0 {
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

	pterm.Info.Printf("Generating batch of %d JWTs and saving to disk\n", limit)
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
			if err := j.config.store.save(tokens); err != nil {
				return err
			}
			tokens = tokens[:0]
		}
	}

	if j.config.store.getJwtCount() == reqJwtAmount {
		// all jwts generated
		return nil
	}

	return j.batchGenSave(reqJwtAmount, batchSize)
}

func (j *JWTGenerator) generate(limit int64, errs chan<- error, response chan<- []string) {
	tokens := make([]string, limit, limit)
	var err error
	var i int64 = 0

	claims := j.commonClaims() // Claims common to all JWTs, computed only once
	for i = 0; i < limit; i++ {
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

	if j.config.JwtCustomClaimsJSON != "" {
		// At this point the JSON in JwtCustomClaimsJSON has already been validated, but checking for errors again in case the workflow changes in the future
		jwtCustomClaimsMap, err := config.JwtCustomClaimsJSONStringToMap(j.config.JwtCustomClaimsJSON)
		if err != nil {
			return claims // Return claims if there's an error
		}
		for key, value := range jwtCustomClaimsMap {
			if key != "" {
				claims[key] = value
			}
		}
	}
	return claims
}
