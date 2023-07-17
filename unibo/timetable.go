package unibo

import (
	"crypto/sha1"
	"fmt"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
)

const baseTimetable = "https://corsi.unibo.it/%s/%s/orario-lezioni/@@orario_reale_json?anno=%d"

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

func GetTimetableUrl(course CourseWebsiteId, anno int, curriculum Curriculum) string {
	url := fmt.Sprintf(baseTimetable, course.Tipologia, course.Id, anno)
	if curriculum != (Curriculum{}) {
		url += fmt.Sprintf("&curricula=%s", curriculum.Value)
	}
	return url
}

func FetchTimetable(course CourseWebsiteId, anno int, curriculum Curriculum) (timetable Timetable, err error) {
	url := GetTimetableUrl(course, anno, curriculum)
	err = fetchJson(url, &timetable)
	return
}

func (t Timetable) ToICS() *ics.Calendar {
	cal := ics.NewCalendar()
	cal.SetMethod(ics.MethodRequest)

	for _, event := range t {
		sha := sha1.New()
		_, err := sha.Write([]byte(fmt.Sprintf("%s%s%s", event.CodModulo, event.Start, event.End)))
		if err != nil {
			return nil
		}
		uid := fmt.Sprintf("%x", sha.Sum(nil))

		e := cal.AddEvent(uid)
		e.SetOrganizer(event.Docente)
		e.SetSummary(event.Title)
		e.SetStartAt(event.Start.Time)
		e.SetEndAt(event.End.Time)

		e.SetDtStampTime(time.Now()) // https://www.kanzaki.com/docs/ical/dtstamp.html

		b := strings.Builder{}

		b.WriteString(fmt.Sprintf("Docente: %s\n", event.Docente))
		if len(event.Aule) > 0 {
			b.WriteString(fmt.Sprintf("Aula: %s\n", event.Aule[0].DesRisorsa))
		}
		b.WriteString(fmt.Sprintf("Cfu: %d\n", event.Cfu))
		b.WriteString(fmt.Sprintf("Periodo: %s\n", event.Periodo))

		e.SetDescription(b.String())

		if len(event.Aule) > 0 {
			e.SetLocation(event.Aule[0].DesRisorsa)
		}
	}

	return cal
}
