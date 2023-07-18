package unibo

import (
	"fmt"
	"io"
	"net/http"
	"testing"
)

func TestGetCurricula(t *testing.T) {
	curricula, err := FetchCurricula(CourseId{Tipologia: "laurea", Id: "informatica"}, 1)
	if err != nil {
		t.Fatalf("Error while fetching curricula: %s", err)
	}

	fmt.Println(curricula)

	url := fmt.Sprintf(
		"https://corsi.unibo.it/%s/%s/orario-lezioni/@@orario_reale_json?anno=%d&curriculum=%s",
		"laurea",
		"informatica",
		1,
		curricula[0].Value,
	)

	res, err := http.Get(url)
	if err != nil {
		t.Fatalf("Error while fetching timetable: %s", err)
	}

	body, _ := io.ReadAll(res.Body)
	fmt.Println(string(body))

}
