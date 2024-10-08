package anythingllm

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/l0nax/go-spew/spew"
	"golang.org/x/sync/semaphore"

	"ciascrape/pkg/bufs"
	"ciascrape/pkg/mu"
)

var (
	pdfRegex      = regexp.MustCompile(pdfRegexPattern)
	pdfGoRoutines = semaphore.NewWeighted(15)

	ErrNoDocuments = errors.New("no documents found")
)

const pdfRegexPattern = `(?m)"application/pdf" src=".*" \/> <a href="(.*\.pdf)" type="application/pdf.*</a>`

func (c *Config) GetPDFLinks(url string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*480)
	defer cancel()
	if err := pdfGoRoutines.Acquire(ctx, 1); err != nil {
		return err
	}

	log.Printf("getting PDFs from page %s", url)

	go func() {
		mu.GetMutex("net").RLock()
		res, err := http.Get(url)
		mu.GetMutex("net").RUnlock()

		if err != nil {
			log.Printf("error getting PDFs from page %s: %v", url, err)
			return
		}

		buf := bufs.GetBuffer()
		defer bufs.PutBuffer(buf)

		n, err := buf.ReadFrom(res.Body)
		if err != nil {
			log.Printf("http response body read error for PDFs: %v", err)
			return
		}
		if n == 0 {
			log.Printf("http response body for PDFs is empty")
			return
		}
		data := buf.Bytes()[:n]

		matches := pdfRegex.FindAllSubmatch(data, -1)
		if len(matches) == 0 {
			log.Printf("(PDF CHECK) %v: %s", ErrNoDocuments, res.Request.URL.String())
			return
		}

		for _, match := range matches {
			if len(match) < 2 {
				continue
			}
			if len(bytes.TrimSpace(match[1])) == 0 || !bytes.Contains(match[1], []byte(".pdf")) {
				continue
			}
			log.Printf("found PDF: %s", match[1])
			doc, err := c.UploadLink(string(match[1]))
			if err != nil {
				log.Printf("error uploading PDF link %s: %v", match[1], err)
				continue
			}
			spew.Dump(doc)
		}

		return
	}()

	return nil
}
