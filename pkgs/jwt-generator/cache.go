package jwt_generator

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/pterm/pterm"
	"os"
	"strconv"
	"strings"
	"time"
)

const byteSizeCounter = 20

type cache struct {
	f       *os.File
	count   int64
	scanner *bufio.Scanner
}

func newCache(f *os.File) (*cache, error) {
	c := cache{f: f}

	c.scanner = bufio.NewScanner(c.f)
	// Get count found on first line of the file
	c.scanner.Split(bufio.ScanLines)
	if c.scanner.Scan() {
		bb := make([]byte, byteSizeCounter)
		_, err := f.ReadAt(bb, 0)
		if err != nil {
			return nil, err
		}

		count, err := getCount(bb)
		if err != nil {
			pterm.Error.Printf("Got error reading jwt count from cache; %v", err)
			return nil, err
		}

		c.count = count
		return &c, nil
	}
	return &c, nil
}

func (c *cache) getJwtCount() int64 {
	return c.count
}

func getCount(bb []byte) (int64, error) {
	num := make([]byte, 0)
	for _, m := range bb {
		if m == 0 {
			break
		}
		num = append(num, m)
	}

	s := string(num)
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return i, nil
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

	// scan first line to skip as not a jwt but is int64 representing number of jwts, need to reset scanner also
	// to avoid invalid char
	c.scanner = bufio.NewScanner(c.f)
	c.scanner.Split(bufio.ScanLines)
	if !c.scanner.Scan() {
		errs <- fmt.Errorf("jwt_generator: retrieving; not able to read first line of cache; %v", c.scanner.Err())
		close(errs)
		close(recv)
		return recv, errs
	}

	meta := c.scanner.Bytes()
	if len(meta) < byteSizeCounter {
		errs <- fmt.Errorf("jwt_generator: retrieving; corrupt jwt cache, wanted 8 bytes got %d", len(meta))
		close(errs)
		close(recv)
		return recv, errs
	}

	i, err := getCount(meta)
	if err != nil {
		errs <- fmt.Errorf("failed to get jwt count; %v", err)
		close(errs)
		close(recv)
		return recv, errs
	}

	if count > i {
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
	defer func() {
		close(errs)
		close(recv)
	}()

	for i = 0; i < count; i++ {
		if c.scanner.Scan() {
			recv <- string(c.scanner.Bytes())
			continue
		}

		if err := c.scanner.Err(); err != nil {
			errs <- err
			return
		}

		errs <- errors.New("unable to read anymore jwts from file")
		return
	}
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

	newCount := int64(add) + c.count
	s := strconv.FormatInt(newCount, 10)

	b := make([]byte, byteSizeCounter)
	for i, ss := range s {
		b[i] = byte(ss)
	}

	_, err = c.f.WriteAt(b, 0)
	if err != nil {
		return err
	}

	_, err = c.f.WriteAt([]byte{byte('\n')}, byteSizeCounter)
	if err != nil {
		return err
	}

	if err := c.f.Sync(); err != nil {
		return err
	}
	c.count = int64(add) + c.count
	return nil
}
