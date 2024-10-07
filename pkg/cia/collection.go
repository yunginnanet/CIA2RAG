package cia

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sync"
	"sync/atomic"
	"time"
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
	done         *atomic.Bool
	maxDocuments int
	mu           sync.RWMutex
}

func NewCollection(name string, maxPagesOpt ...int) *Collection {
	maxPages := maxPagesDefault

	if len(maxPagesOpt) > 0 {
		maxPages = maxPagesOpt[0]
	}

	done := &atomic.Bool{}
	done.Store(false)

	return &Collection{
		Name:         name,
		Pages:        make(map[int]chan string),
		done:         done,
		maxDocuments: maxPages * 20,
	}
}

func (c *Collection) WithMaxPages(maxPages int) *Collection {
	c.maxDocuments = maxPages * 20
	return c
}

func (c *Collection) GetPages() error {
	wg := &sync.WaitGroup{}

	if c.maxDocuments < 20 {
		c.maxDocuments = 20
	}

	for i := 1; ; i++ {
		if i > c.maxDocuments/20 {
			break
		}

		res, err := http.Head(PageURL(c.Name, i))
		if err != nil {
			c.done.Store(true)
			return err
		}
		switch res.StatusCode {
		case http.StatusOK:
			log.Printf("found page %d", i)

			var channel chan string
			var pageCt int

			c.mu.Lock()
			c.Pages[i] = make(chan string, 25)
			channel = c.Pages[i]
			pageCt = len(c.Pages)
			c.mu.Unlock()

			if pageCt*20 >= c.maxDocuments {
				return nil
			}

			go func() {
				wg.Add(1)
				if err := c.GetPage(i, channel, wg); err != nil {
					log.Printf("error getting page %d: %v", i, err)
				}
			}()

			time.Sleep(time.Millisecond * time.Duration(i*35))
			continue

		case http.StatusNotFound:
			if i == 0 {
				return ErrNoPages
			}
			wg.Wait()
			c.done.Store(true)
			return nil
		default:
			c.done.Store(true)
			return fmt.Errorf("%w: %d", ErrBadStatusCode, res.StatusCode)
		}
	}

	wg.Wait()
	c.done.Store(true)
	return nil
}

func (c *Collection) Drain(ctx context.Context) chan string {
	var documents = make(chan string, c.maxDocuments)

	var (
		seenMap = make(map[string]bool)
		seenMu  sync.RWMutex
	)

	go func() {
		defer func() {
			close(documents)
			log.Println("drained all documents")
		}()
		for i := 1; ; i++ {
		try:

			if c.done.Load() {
				return
			}

			c.mu.RLock()
			channel, ok := c.Pages[i]
			c.mu.RUnlock()
			if !ok {
				if c.done.Load() {
					return
				}
				time.Sleep(time.Millisecond * 500)
				goto try
			}
			go func() {
				for page := range channel {

					seenMu.RLock()
					_, seenOK := seenMap[page]
					if seenOK {
						seenMu.RUnlock()
						continue
					}
					seenMu.RUnlock()

					seenMu.Lock()
					seenMap[page] = true
					seenMu.Unlock()

					documents <- page

				}
			}()
		}
	}()

	return documents
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

func (c *Collection) GetPage(i int, channel chan string, wg *sync.WaitGroup) error {
	defer wg.Done()

	time.Sleep(time.Millisecond * (time.Duration(i * 200)))

	log.Printf("getting page %d", i)

	res, err := http.Get(PageURL(c.Name, i))
	if err != nil {
		return err
	}

	log.Printf("parsing page %d", i)

	links, err := ParsePage(res)
	if err != nil {
		_ = res.Body.Close()
		return err
	}

	for _, link := range links {
		log.Printf("page %d found document: %s", i, link)
		channel <- link
	}

	close(channel)

	log.Printf("page %d has %d documents", i, len(links))

	_ = res.Body.Close()

	return nil
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
