package jwt_generator

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

type cache struct {
	f       *os.File
	count   int64
	scanner *bufio.Scanner
}

func newCache(f *os.File) (*cache, error) {
	c := cache{f: f}

	c.scanner = bufio.NewScanner(c.f)
	c.scanner.Split(bufio.ScanLines)
	if c.scanner.Scan() {
		meta := c.scanner.Bytes()
		if len(meta) < 8 {
			return nil, fmt.Errorf("jwt_generator: corrupt jwt cache, wanted 8 bytes got %d", len(meta))
		}
		c.count = int64(binary.LittleEndian.Uint64(meta[0:8]))

		return &c, nil
	}
	return &c, nil
}

func (c *cache) getJwtCount() int64 {
	return c.count
}

func (c *cache) get(count int64) (<-chan string, <-chan error) {
	recv := make(chan string, 1000000)
	errs := make(chan error, 1)

	// set to beginning of file to read jwt amount
	if _, err := c.f.Seek(0, 0); err != nil {
		errs <- err
		close(errs)
		close(recv)
		return recv, errs
	}

	// scan first line to skip as not a jwt but is int64 representing number of jwts
	if !c.scanner.Scan() {
		errs <- fmt.Errorf("jwt_generator: retrieving; not able to read first line of cache; %v", c.scanner.Err())
		close(errs)
		close(recv)
		return recv, errs
	}

	meta := c.scanner.Bytes()
	if len(meta) < 8 {
		errs <- fmt.Errorf("jwt_generator: retrieving; corrupt jwt cache, wanted 8 bytes got %d", len(meta))
		close(errs)
		close(recv)
		return recv, errs
	}

	if count > int64(binary.LittleEndian.Uint64(meta[0:8])) {
		errs <- errors.New("jwt_generator: retrieving; not enough jwts stored in cache")
		close(errs)
		close(recv)
		return recv, errs
	}

	go c.retrieve(count, recv, errs)
	// allow some time to prime cache so workers aren't waiting for jwts
	time.Sleep(1 * time.Second)
	return recv, errs
}

func (c *cache) retrieve(count int64, recv chan<- string, errs chan<- error) {
	var i int64 = 0

	for i = 0; i < count; i++ {
		if c.scanner.Scan() {
			recv <- string(c.scanner.Bytes())
			continue
		}
		// reached EOF or err
		if err := c.scanner.Err(); err != nil {
			errs <- err
			close(errs)
		}
		break
	}
	close(recv)
}

func (c *cache) save(tokens []string) error {
	stat, err := c.f.Stat()
	if err != nil {
		return err
	}

	add := len(tokens)
	var pos int64 = 10
	if stat.Size() > 0 {
		pos = stat.Size()
	}
	if _, err := c.f.WriteAt([]byte(strings.Join(tokens, "\n")+"\n"), pos); err != nil {
		return err
	}

	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(int64(add)+c.count))
	_, err = c.f.WriteAt(b, 0)
	if err != nil {
		return err
	}
	_, err = c.f.WriteAt([]byte{byte('\n')}, 9)
	if err != nil {
		return err
	}

	if err := c.f.Sync(); err != nil {
		return err
	}
	c.count = int64(add) + c.count
	return nil
}
