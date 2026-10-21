package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/VaiTon/unibocalendar/unibo_integ"
	"github.com/go-playground/assert/v2"
)

func Test_indexPageCourseNumbers(t *testing.T) {
	data := unibo_integ.CoursesMap{
		8009: {Codice: 8009, Descrizione: "Informatica", Tipologia: "Laurea", Campus: "Bologna", AnnoAccademico: "2025/2026"},
		6640: {Codice: 6640, Descrizione: "Informatica", Tipologia: "Laurea", Campus: "Bologna", AnnoAccademico: "2025/2026"},
	}
	r := setupRouter(data)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	body := w.Body.String()
	for code := range data {
		if !strings.Contains(body, fmt.Sprintf(">%d</td>", code)) {
			t.Errorf("course %d is missing its visible number", code)
		}
		if !strings.Contains(body, fmt.Sprintf("data-course=\"%d Laurea in Informatica\"", code)) {
			t.Errorf("course %d is missing its number in the search data", code)
		}
		if !strings.Contains(body, fmt.Sprintf("href=\"/courses/%d\"", code)) {
			t.Errorf("course %d is missing its link", code)
		}
	}
}

func Test_coursePage(t *testing.T) {

	downloadOpenDataIfNewer()

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
