package unibo

import (
	"fmt"
	"strings"
)

const baseCurriculaIt = "https://corsi.unibo.it/%s/%s/orario-lezioni/@@available_curricula?anno=%d&curricula="
const baseCurriculaEn = "https://corsi.unibo.it/%s/%s/timetable/@@available_curricula?anno=%d&curricula="

type Curriculum struct {
	Selected bool   `json:"selected"`
	Value    string `json:"value"`
	Label    string `json:"label"`
}

type Curricula []Curriculum

func GetCurriculaUrl(course CourseId, year int) string {
	if strings.Contains(course.Tipologia, "cycle") {
		return fmt.Sprintf(baseCurriculaEn, course.Tipologia, course.Id, year)
	}

	return fmt.Sprintf(baseCurriculaIt, course.Tipologia, course.Id, year)
}

func FetchCurricula(course CourseId, year int) (curricula Curricula, err error) {
	url := GetCurriculaUrl(course, year)
	err = getJson(url, &curricula)
	return
}
