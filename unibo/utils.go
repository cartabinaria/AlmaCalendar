package unibo

import (
	"encoding/json"
	"net/http"
)

type transport struct {
	http.RoundTripper
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", "CalendarBot")
	return t.RoundTripper.RoundTrip(req)
}

// Client is the http client used to make requests.
// It is used to set a custom User-Agent.
var Client = http.Client{
	Transport: &transport{
		http.DefaultTransport,
	},
}

func getJson(url string, v interface{}) error {
	// Get the resource
	res, err := Client.Get(url)
	if err != nil {
		return err
	}

	// Parse the body
	err = json.NewDecoder(res.Body).Decode(v)
	if err != nil {
		return err
	}

	// Close the body
	err = res.Body.Close()
	if err != nil {
		return err
	}

	return nil
}
