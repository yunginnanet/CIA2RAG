package anythingllm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	spew2 "github.com/davecgh/go-spew/spew"
	"github.com/l0nax/go-spew/spew"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"golang.org/x/sync/semaphore"

	"ciascrape/pkg/bufs"
	seekablebuffer "ciascrape/pkg/bufs/3rd_party"
	"ciascrape/pkg/mu"
)

var (
	pdfRegex      = regexp.MustCompile(pdfRegexPattern)
	pdfGoRoutines = semaphore.NewWeighted(500)

	ErrNoDocuments = errors.New("no documents found")
)

const pdfRegexPattern = `(?m)"application/pdf" src=".*" \/> <a href="(.*\.pdf)" type="application/pdf.*</a>`

var PDFConfig = model.NewDefaultConfiguration()

func init() {
	PDFConfig.Cmd = model.LISTKEYWORDS
	PDFConfig.DecodeAllStreams = true
}

func sliceEmpty(s []string) bool {
	return s == nil || len(s) == 0
}

func getPDFData(url string) []byte {
	var (
		err error
		dat []byte
	)
	url, dat, err = seekPDF(url)
	if err != nil {
		log.Printf("error getting PDF data for '%s': %v", url, err)
		return nil
	}
	return dat
}

func (c *Config) GetPDFLinks(url string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*480)
	defer cancel()
	if err := pdfGoRoutines.Acquire(ctx, 1); err != nil {
		return err
	}

	defer pdfGoRoutines.Release(1)

	log.Printf("getting PDFs from page %s", url)

	handleFailedPDF := func(pdfUrl, pdfName string, buf *bytes.Buffer) (resData []byte) {
		var err error
		log.Printf("error uploading PDF link '%s': %v\nretrying as upload...", pdfUrl, err)
		dat := getPDFData(pdfUrl)
		docDat := c.altUploadPDF(pdfUrl, pdfName, buf, dat)
		if docDat != nil && len(docDat) > 0 {
			log.Printf("retrying as upload successful: \n%s", spew.Sdump(docDat))
			return
		}
		log.Printf("retrying by extracting keywords: '%s'", pdfUrl)
		keyWords := extractKeyWords(dat)
		if sliceEmpty(keyWords) {
			log.Printf("error extracting keywords from PDF '%s': got nil result", pdfUrl)
			return
		}
		if keyWords != nil && len(keyWords) > 0 {
			keyw := strings.Join(keyWords, " ")
			hr := strings.Repeat("-", 15)
			log.Printf("got keywords for '%s': \n%s\n%s\n%s\nuploading...", pdfUrl, hr, keyw, hr)

			if resData, err = c.UploadRaw(pdfUrl, keyw); err != nil {
				log.Printf("error uploading extracted PDF data '%s': %v", pdfUrl, err)
				return
			}
		}
		return
	}

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
			pdfUrl := string(match[1])

			if strings.Contains(pdfUrl, "document") {
				pdfUrl = cleanPDFURL(pdfUrl)
			}

			var resData []byte

			doc, err := c.UploadLink(pdfUrl)
			if err != nil {
				resData = handleFailedPDF(pdfUrl, string(match[0]), buf)
				rtr := &RawTextResp{}
				if err := json.Unmarshal(resData, rtr); err == nil && len(rtr.Documents[0].PageContent) > 0 {
					doc = &rtr.Documents[0]
				}
			}
			if doc != nil {
				spew.Dump(doc)
			}
			if resData != nil && len(resData) > 0 {
				spew.Dump(resData)
			}
		}

		return
	}()

	return nil
}

func (c *Config) altUploadPDF(url string, pdfName string, buf *bytes.Buffer, dats ...[]byte) []byte {
	var err error

	var dat []byte
	if len(dats) == 1 {
		dat = dats[0]
	} else {
		dat = getPDFData(url)
	}

	if dat == nil || len(dat) == 0 {
		return nil
	}

	var res *http.Response

	res, err = c.upload("v1/document/upload", pdfName, bytes.NewReader(dat))

	if err != nil || res == nil {
		if err == nil {
			err = errors.New("upload failed")
		}
		log.Printf("error uploading %d bytes of PDF data: %v", len(dat), err)
		if res != nil {
			_ = res.Body.Close()
			return nil
		}
	}
	var resDat []byte
	buf.Reset()
	if res != nil && res.Body != nil {
		n, err := buf.ReadFrom(res.Body)
		if err != nil {
			log.Printf("http response body read error for PDFs: %v", err)
			_ = res.Body.Close()
			return nil
		}
		_ = res.Body.Close()
		resDat = buf.Bytes()[:n]
	}
	if len(resDat) == 0 {
		log.Printf("http response body for PDFs is empty")
		return nil
	}

	return resDat
}

func extractKeyWords(data []byte) []string {
	sb := &seekablebuffer.Buffer{}
	var n int
	var err error
	n, err = sb.Write(data)
	if err != nil {
		log.Printf("failed to write PDF byte data to buffer: %v", err)
		return []string{}
	}
	if n == 0 {
		log.Printf("0 byte write result when writing PDF data written to buffer")
		return []string{}
	}
	keyWords, err := api.Keywords(sb, PDFConfig)
	if err != nil {
		log.Printf("error extracting keywords from PDFs: %v", err)
		return []string{}
	}
	if len(keyWords) == 0 {
		log.Printf("no keywords extracted from PDFs")
		return []string{}
	}
	return keyWords
}

func seekPDF(url string) (string, []byte, error) {
	buf := bufs.GetBuffer()
	defer bufs.PutBuffer(buf)

	processRes := func(res *http.Response) ([]byte, error) {
		if res == nil || res.StatusCode != http.StatusOK || res.Body == nil {
			return nil, fmt.Errorf("invalid PDF response:\n%s", spew2.Sdump(res))
		}
		var n int64
		var err error
		if n, err = buf.ReadFrom(res.Body); err != nil || n == 0 {
			if n == 0 && err == nil {
				err = fmt.Errorf("empty PDF response body")
			}
			return nil, fmt.Errorf("error reading PDF response body: %w", err)
		}
		defer func() {
			if res != nil && res.Body != nil {
				_ = res.Body.Close()
			}
		}()
		data := make([]byte, n)
		n2 := copy(data, buf.Bytes()[:n])
		if int64(n2) != n {
			log.Printf("error copying PDF response body: %d != %d", n2, n)
			if n2 == 0 {
				return nil, fmt.Errorf("error copying PDF response body: %d != %d", n2, n)
			}
		}
		bufs.PutBuffer(buf)
		return data, nil
	}

	var (
		res *http.Response
		dat []byte
		err error
	)

	if !strings.HasSuffix(url, ".pdf") {
		url = url + ".pdf"
	}

	if res, err = http.DefaultClient.Get(url); err == nil {
		if dat, err = processRes(res); err == nil {
			return url, dat, err
		}
	}

	url = cleanPDFURL(url)

	if res, err = http.DefaultClient.Get(url); err == nil {
		if dat, err = processRes(res); err == nil {
			return cleanPDFURL(url), dat, err
		}
	}

	return url, nil, err
}

func cleanPDFURL(url string) string {
	file := strings.Split(url, "/")[len(strings.Split(url, "/"))-1]
	url = strings.TrimSuffix(url, file)
	file = strings.TrimSuffix(file, ".pdf")
	file = strings.TrimSuffix(file, ".PDF")
	file = strings.ToUpper(file)
	file = file + ".pdf"
	url = url + file
	url = strings.ReplaceAll(url, "//", "/")
	url = strings.ReplaceAll(url, "https:/", "https://")
	//goland:noinspection HttpUrlsUsage
	url = strings.ReplaceAll(url, "http:/", "http://")

	pdfUrl := strings.Replace(url, "readingroom/document", "readingroom/docs", 1)

	if !strings.HasSuffix(pdfUrl, ".pdf") {
		pdfUrl = pdfUrl + ".pdf"
	}
	return pdfUrl
}
