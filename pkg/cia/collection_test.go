package cia

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"ciascrape/pkg/bufs"
)

const testDataPacked = `H4sIAAAAAAACA+2YUW+iQBDH3/0UG156fVgWFirYQxJbbc70Shvlcs8Iq5BDILDqeZ/+dqGYrZ5Ve+kl9TCKu7AzOzP85xcFACuIljZ4flkZ8GOvKLpSkPruOiOSbRWZlwBkWyhjyyxULW+B0rBevYzIqoDTiMQBEMaQRjRmPjbuQ722qK77aUJJQtkuHghzMu1KKCdeECWzPE3niAWxmLPryI88mAdZpw0VxTDNXFFUrCi6qiiKBrFkf3J7N6B3Cdze+B7cPY5uBxbyWMihfnrM1bHeGiaL+YTkVQ5VLSr7i8om9iYkBsK4ztz7mSbpfM0dQcoqeWH3n10Cp3R5zQLj/nY8b9fmdtiDo/5TnfzoZfIbJ29KM/NmhO204PvsxPF6hqLpExsXLxMSQtjKB9eRviniaRQTqAqa2r+RRb1JTOqLBY38H2tIEn4ykOwWsGjIxMaW5ewT2j1KPT/kd8hCbMpPjaNfpJywWPkqVFm0LDpJg3XpIq/9p0HAtwzsrXvJG8CK5jPxDIxYkBLwYtqVnvp3oJqW3dKVvCyLI9+jUZqgLJhKoMj9rcaYp8EiJgXivhA3LpBgBZmVnCUzibUt2HRWSGlWXCO0Wq1k1k/yLF1uN1uBXteaXIbD1bwT5WcQk2RGwy7GbWweEi13VHVoKRhW16AsHcayCu5vynlV8hY7VrVmA37n+OBN8iFBxLo5DUQevdJ1f9dX/MYUsCjhdIxSt1Kqvv+4Y56uwGYE1SthUiqwVW32bPuRya7zNi/JfnMJHobj8fDRAWO35w4eBo77fwCe1aABfAP4fwJ4prVjAK9pV23zkGj3AV7DsmE2hD+F8G1hQpYkOQPEM6EouF0qpcMQ/81xh+7XQf/yrJkuJv3xmK42TP9ATBe1dgzTVVM39EOi3cd01ZQVvWH6KUw3zulXewfnXCDsbbBvtXwec/ZI7+DRy6QbpDdIfyek72jtGKS3NWwo2iHV7mN6WzVk02igfgrUzfP7oa5qiqLxv3QYaiLVwZee0/8+GrruwAHOo3vmj9uFKjRPYxrMv+vTGEFrR2Fe1XR8SLT7KH/VkTsHGf8bRMEjvCMcAAA=`

var testData string

func getTestData() {
	b64Dec := base64.NewDecoder(base64.StdEncoding, strings.NewReader(testDataPacked))
	gz, _ := gzip.NewReader(b64Dec)
	buf := bufs.GetBuffer()
	n, _ := buf.ReadFrom(gz)
	if n == 0 {
		panic(errors.New("test data is empty"))
	}
	testData = buf.String()
	bufs.PutBuffer(buf)
}

func init() {
	getTestData()
}

func TestEndpointURL_ReturnsCorrectURL(t *testing.T) {
	expected := EndpointCollection() + "testCollection"
	result := EndpointURL("testCollection")
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestPageURL_ReturnsCorrectURL(t *testing.T) {
	for i := 1; i <= 10; i++ {
		expected := EndpointCollection() + "testCollection"
		if i > 1 {
			expected += "?page=" + strconv.Itoa(i)
		}
		result := PageURL("testCollection", i)
		if result != expected {
			t.Errorf("expected %s, got %s", expected, result)
		}
	}
}

func TestNewCollection_SetsNameAndPages(t *testing.T) {
	collection := NewCollection("test")
	if collection.Name != "test" {
		t.Errorf("expected name to be 'test', got %s", collection.Name)
	}
	if collection.Pages == nil {
		t.Errorf("expected pages to be initialized, got nil")
	}
}

func TestGetPages_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/readingroom/collection/test" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Query().Get("page") != "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	EndpointBase = server.URL + "/"
	collection := NewCollection("test")
	err := collection.GetPages()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if len(collection.Pages) == 0 {
		t.Errorf("expected pages to be populated, got empty")
	}
}

func TestGetPages_NoPages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		return
	}))
	defer server.Close()

	EndpointBase = server.URL + "/"
	collection := NewCollection("test")
	err := collection.GetPages()
	if !errors.Is(err, ErrNoPages) {
		t.Errorf("expected error %v, got %v", ErrNoPages, err)
	}
}

func TestParsePage_Success(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		body := bytes.NewBufferString(`<div class="field-content"><a href="/readingroom/document/test">Document</a></div>`)
		res := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(body),
			Request:    &http.Request{URL: &url.URL{Path: "/readingroom/collection/test"}},
		}

		prefix := EndpointBase + "readingroom/document/"

		links, err := ParsePage(res)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(links) == 0 || links[0] != prefix+"test" {
			t.Errorf("expected links to contain '%s', got %v", prefix, links)
		}
	})

	t.Run("test_data", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err := w.Write([]byte(testData))
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				t.Fatalf("failed to write test data: %v", err)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		EndpointBase = server.URL + "/"

		res, err := http.Get(server.URL + "/readingroom/collection/stargate")
		if err != nil {
			t.Fatalf("failed to get test data: %v", err)
		}

		links, err := ParsePage(res)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(links) == 0 {
			t.Errorf("expected links to contain 'test', got %v", links)
		}
		for _, link := range links {
			prefix := EndpointBase + "readingroom/document/"
			if !strings.HasPrefix(link, prefix) {
				t.Errorf("expected links to contain '%s', got %v", prefix, link)
			}
			t.Logf("link: %s", link)
		}
	})
}

func TestParsePage_NoDocuments(t *testing.T) {
	body := bytes.NewBufferString(`<div class="field-content"></div>`)
	res := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(body),
		Request:    &http.Request{URL: &url.URL{Path: "/readingroom/collection/test"}},
	}

	links, err := ParsePage(res)
	if !errors.Is(err, ErrNoDocuments) {
		t.Errorf("expected error %v, got %v", ErrNoDocuments, err)
	}
	if links != nil {
		t.Errorf("expected links to be nil, got %v", links)
	}
}

func TestValidate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		return
	}))
	defer server.Close()

	EndpointBase = server.URL + "/"
	collection := NewCollection("test")
	err := collection.Validate()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestValidate_CollectionNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		return
	}))
	defer server.Close()

	EndpointBase = server.URL + "/"
	collection := NewCollection("test")
	err := collection.Validate()
	if !errors.Is(err, ErrCollectionNotFound) {
		t.Errorf("expected error %v, got %v", ErrCollectionNotFound, err)
	}
}
