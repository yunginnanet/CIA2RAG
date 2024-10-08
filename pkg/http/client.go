package http

import (
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

type Client struct {
	*http.Client
}

var DefaultClient = &Client{http.DefaultClient}

var i = &atomic.Int64{}

func init() {
	i.Store(0)
}

func getI() int {
	if i.Load() > 4 {
		i.Store(-1)
	}
	mi := i.Add(1)
	if mi < 0 {
		mi = 0
	}
	if mi > 4 {
		mi = 4
	}
	return int(mi)
}

var im = map[int]string{
	0: "spooky",
	1: "george-foreman",
	2: "varth-dader",
	3: "war-stars",
	4: "elusive",
}

func getUA() string {
	mi := getI()
	base := im[mi]
	if mi%2 == 0 {
		base = strings.ToUpper(base)
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	return base + fmt.Sprintf("/%d.%d", rng.Intn(85), rng.Intn(10))
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", getUA())
	return c.Client.Do(req)
}
