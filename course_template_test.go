package main

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/VaiTon/unibocalendar/unibo_integ"
	"github.com/cartabinaria/unibo-go/curriculum"
)

func Test_courseCalendarGuides(t *testing.T) {
	curr := curriculum.Curriculum{Value: "A", Label: "Curriculum A"}
	page := createMyRender().Instance("course", map[string]any{
		"Course": &unibo_integ.Course{Codice: 8009, Descrizione: "Informatica", DurataAnni: 1},
		"Curricula": map[int]curriculum.Curricula{
			1: {curr},
		},
		"Teachings": subjectMap{
			1: {curr: {{Code: "101", Name: "Programmazione"}}},
		},
	})
	w := httptest.NewRecorder()
	if err := page.Render(w); err != nil {
		t.Fatal(err)
	}

	body := w.Body.String()
	for _, text := range []string{
		"Usare Notion Calendar e TimeTree",
		"senza una sincronizzazione diretta",
		"stesso account Google",
		"app mobile",
		"non si aggiorna automaticamente",
		"https://www.notion.com/help/manage-your-calendars-and-events",
		"https://support.timetreeapp.com/hc/en-us/articles/360000629341-How-to-Use-Other-Calendars-such-as-Google-Calendar",
		"/cal/8009/1?curr=A",
		"/exams/8009/1?curr=A",
	} {
		if !strings.Contains(body, text) {
			t.Errorf("course page is missing %q", text)
		}
	}
	if strings.Count(body, "Usare Notion Calendar e TimeTree") != 1 {
		t.Error("calendar guides should appear once per course page")
	}
}
