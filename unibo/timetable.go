package unibo

import (
	"fmt"
	"strings"
	"time"
)

const baseTimetable = "https://corsi.unibo.it/%s/%s/orario-lezioni/@@orario_reale_json?anno=%d"
const baseTimetableEn = "https://corsi.unibo.it/%s/%s/timetable/@@orario_reale_json?anno=%d"

type CalendarTime struct {
	time.Time
}

func (c *CalendarTime) UnmarshalJSON(b []byte) error {
	t, err := time.Parse(`"2006-01-02T15:04:05"`, string(b))
	if err != nil {
		return err
	}

	c.Time = t
	return nil
}

func (c *CalendarTime) MarshalJSON() ([]byte, error) {
	return []byte(c.Format(`"2006-01-02T15:04:05"`)), nil
}

type Aula struct {
	DesRisorsa string `json:"des_risorsa"`
}

type TimetableEvent struct {
	CodModulo         string       `json:"cod_modulo"`
	PeriodoCalendario string       `json:"periodo_calendario"`
	CodSdoppiamento   string       `json:"cod_sdoppiamento"`
	Title             string       `json:"title"`
	ExtCode           string       `json:"extCode"`
	Periodo           string       `json:"periodo"`
	Docente           string       `json:"docente"`
	Cfu               int          `json:"cfu"`
	Teledidattica     bool         `json:"teledidattica"`
	Teams             string       `json:"teams,omitempty"`
	Start             CalendarTime `json:"start"`
	End               CalendarTime `json:"end"`
	Aule              []Aula       `json:"aule"`
}

type Timetable []TimetableEvent

type TimetablePeriod struct {
	Start time.Time
	End   time.Time
}

// GetTimetableUrl returns the URL to fetch the timetable for the given course.
//
// If `curriculum` is not empty, it will be used to filter the timetable.
// If `period` is not nil, it will be used to filter the timetable.
func GetTimetableUrl(course CourseId, curriculum Curriculum, year int, period *TimetablePeriod) string {

	var url string
	if strings.Contains(course.Tipologia, "cycle") {
		url = fmt.Sprintf(baseTimetableEn, course.Tipologia, course.Id, year)
	} else {
		url = fmt.Sprintf(baseTimetable, course.Tipologia, course.Id, year)
	}

	if curriculum != (Curriculum{}) {
		url += fmt.Sprintf("&curricula=%s", curriculum.Value)
	}

	if period != nil {
		url += fmt.Sprintf("&start=%s", period.Start.Format("2006-01-02"))
		url += fmt.Sprintf("&end=%s", period.End.Format("2006-01-02"))
	}

	return url
}

func FetchTimetable(
	course CourseId,
	curriculum Curriculum,
	year int,
	period *TimetablePeriod,
) (timetable Timetable, err error) {
	url := GetTimetableUrl(course, curriculum, year, period)
	err = getJson(url, &timetable)
	return
}
