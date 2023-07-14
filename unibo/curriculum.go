package unibo

import (
	"encoding/json"
	"fmt"
)

const baseCurricula = "https://corsi.unibo.it/%s/%s/orario-lezioni/@@available_curricula?anno=%d&curricula="

type Curriculum struct {
	Selected bool   `json:"selected"`
	Value    string `json:"value"`
	Label    string `json:"label"`
}

func GetCurriculaUrl(course CourseWebsiteId, year int) string {
	return fmt.Sprintf(baseCurricula, course.Tipologia, course.Id, year)
}

func GetCurricula(course CourseWebsiteId, year int) ([]Curriculum, error) {
	url := GetCurriculaUrl(course, year)

	response, err := Client.Get(url)
	if err != nil {
		return nil, err
	}

	var curricola []Curriculum
	err = json.NewDecoder(response.Body).Decode(&curricola)

	err = response.Body.Close()
	if err != nil {
		return nil, err
	}

	return curricola, nil
}
