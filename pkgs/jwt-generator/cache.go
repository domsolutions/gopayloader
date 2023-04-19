package jwt_generator

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
)

type cache struct {
	f     *os.File
	count int64
}

func newCache(f *os.File) (*cache, error) {
	c := cache{f: f}

	scanner := bufio.NewScanner(c.f)
	scanner.Split(bufio.ScanLines)
	if scanner.Scan() {
		meta := scanner.Bytes()
		if len(meta) < 8 {
			return nil, fmt.Errorf("jwt_generator: corrupt jwt cache, wanted 8 bytes got %d", len(meta))
		}
		var err error
		c.count = int64(binary.LittleEndian.Uint64(meta[0:8]))
		if err != nil {
			return nil, err
		}
		return &c, nil
	}
	return &c, nil
}

func (c *cache) getJwtCount() int64 {
	return c.count
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
