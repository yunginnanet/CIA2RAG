package cia

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sync"
)

var (
	EndpointBase    = "https://www.cia.gov/"
	maxPagesDefault = 1000
)

func EndpointCollection() string {
	return EndpointBase + "readingroom/collection/"
}

const pageRegexPattern = `field-content"><a href="\/readingroom\/document\/(.*)">`

var (
	ErrBadStatusCode      = errors.New("bad status code")
	ErrCollectionNotFound = errors.New("reading room collection does not exist")
	ErrNoPages            = errors.New("no pages found in collection")
	ErrNoDocuments        = errors.New("no documents found in page")
	ErrPageNotFound       = errors.New("page not found in collection")

	pageRegex = regexp.MustCompile(pageRegexPattern)
)

var buffers = &sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(nil)
	},
}

func getBuffer() *bytes.Buffer {
	buf := buffers.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

func putBuffer(buf *bytes.Buffer) {
	buffers.Put(buf)
}

// Collection collects the pages of a reading room collection and provides a channel for each page.
// The nature of this struct means that once the channels are drained, they are gone.
// Therefore this is only useful for scraping a collection once.
type Collection struct {
	Name         string
	Pages        map[int]chan string
	maxDocuments int
	mu           sync.RWMutex
}

func NewCollection(name string, maxPagesOpt ...int) *Collection {
	maxPages := maxPagesDefault

	if len(maxPagesOpt) > 0 {
		maxPages = maxPagesOpt[0]
	}

	return &Collection{
		Name:         name,
		Pages:        make(map[int]chan string),
		maxDocuments: maxPages * 20,
	}
}

func (c *Collection) GetPages() error {
	for i := 0; ; i++ {
		res, err := http.Head(PageURL(c.Name, i))
		if err != nil {
			return err
		}
		switch res.StatusCode {
		case http.StatusOK:

			c.mu.RLock()
			if len(c.Pages) >= c.maxDocuments {
				c.mu.RUnlock()
				return nil
			}

			if _, ok := c.Pages[i]; ok {
				c.mu.RUnlock()
				continue
			}
			c.mu.RUnlock()

			c.mu.Lock()
			c.Pages[i] = make(chan string, 25)
			c.mu.Unlock()
			println("getting page", i)
			go c.GetPage(i)

		case http.StatusNotFound:
			if i == 0 {
				return ErrNoPages
			}
			return nil
		default:
			return fmt.Errorf("%w: %d", ErrBadStatusCode, res.StatusCode)
		}
	}
}

func ParsePage(res *http.Response) ([]string, error) {
	defer func() {
		_ = res.Body.Close()
	}()
	switch res.StatusCode {
	case http.StatusOK:
		break
	case http.StatusNotFound:
		return nil, fmt.Errorf("%w: %s", ErrPageNotFound, res.Request.URL.String())
	default:
		return nil, fmt.Errorf("%w: %d", ErrBadStatusCode, res.StatusCode)
	}

	buf := getBuffer()
	defer putBuffer(buf)

	n, err := buf.ReadFrom(res.Body)
	if err != nil {
		return nil, fmt.Errorf("http response body read error: %w", err)
	}
	if n == 0 {
		return nil, fmt.Errorf("http response body is empty")
	}
	data := buf.Bytes()[:n]

	matches := pageRegex.FindAllSubmatch(data, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrNoDocuments, res.Request.URL.String())
	}

	matchStrings := make([]string, len(matches))
	for i, match := range matches {
		if len(match) < 2 {
			continue
		}
		matchStrings[i] = EndpointBase + "readingroom/document/" + string(match[1])
	}

	return matchStrings, nil
}

func (c *Collection) GetPage(i int) {
	res, err := http.Get(PageURL(c.Name, i))
	if err != nil {
		return
	}

	links, err := ParsePage(res)
	if err != nil {
		_ = res.Body.Close()
		return
	}

	for _, link := range links {
		c.mu.RLock()
		c.Pages[i] <- link
		c.mu.RUnlock()
	}
}

func (c *Collection) Validate() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	res, err := http.Head(EndpointURL(c.Name))
	if err != nil {
		return err
	}

	switch res.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return fmt.Errorf("%w: %s", ErrCollectionNotFound, c.Name)
	default:
		return fmt.Errorf("%w: %d", ErrBadStatusCode, res.StatusCode)
	}
}
