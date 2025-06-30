package unibo_integ

import (
	"net/http"
)

type transport struct {
	http.RoundTripper
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", "CalendarBot")
	return t.RoundTripper.RoundTrip(req)
}

// httpClient is the http httpClient used to make requests.
// It is used to set a custom User-Agent.
var httpClient = http.Client{Transport: &transport{http.DefaultTransport}}
