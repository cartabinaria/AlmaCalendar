package unibo

import (
	"fmt"
)

const baseCurricula = "https://corsi.unibo.it/%s/%s/orario-lezioni/@@available_curricula?anno=%d&curricula="

type Curriculum struct {
	Selected bool   `json:"selected"`
	Value    string `json:"value"`
	Label    string `json:"label"`
}

type Curricula []Curriculum

func GetCurriculaUrl(course CourseWebsiteId, year int) string {
	return fmt.Sprintf(baseCurricula, course.Tipologia, course.Id, year)
}

func FetchCurricula(course CourseWebsiteId, year int) (curricula Curricula, err error) {
	url := GetCurriculaUrl(course, year)
	err = fetchJson(url, &curricula)
	return
}
