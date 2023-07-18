package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-playground/assert/v2"
)

func Test_coursePage(t *testing.T) {

	data, err := openData()
	if err != nil {
		t.Fatal(err)
	}

	r := setupRouter(data)

	for _, course := range data {
		c := course
		t.Run(strconv.Itoa(c.Codice), func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", fmt.Sprintf("/courses/%d", c.Codice), nil)

			r.ServeHTTP(w, req)

			assert.Equal(t, 200, w.Code)
		})

	}
}
