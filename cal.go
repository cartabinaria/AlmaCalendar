package main

import (
	"crypto/sha1"
	"fmt"
	"strings"
	"time"

	"github.com/cartabinaria/unibo-go/exams"
	"github.com/cartabinaria/unibo-go/timetable"

	ics "github.com/arran4/golang-ical"

	"github.com/VaiTon/unibocalendar/unibo_integ"
)

// createCourseCal creates a calendar from the given timetable.
//
// If subjectCodes is not nil, it will be used to filter the timetable by subjects.
func createCourseCal(
	timetable timetable.Timetable,
	course *unibo_integ.Course,
	year int,
	subjectCodes []string,
) (*ics.Calendar, error) {

	// Filter timetable by subjects
	if subjectCodes != nil {
		timetable = filterTimetableBySubjects(timetable, subjectCodes)
	}

	cal := ics.NewCalendar()
	cal.SetMethod(ics.MethodRequest)

	for _, event := range timetable {
		sha := sha1.New()
		_, err := fmt.Fprintf(sha, "%s%s%s", event.CodModulo, event.Start, event.End)
		if err != nil {
			return nil, err
		}

		eventUid := fmt.Sprintf("%x", sha.Sum(nil))

		e := cal.AddEvent(eventUid)
		e.SetOrganizer(event.Teacher)
		e.SetSummary(event.Title)
		e.SetStartAt(event.Start.Time)
		e.SetEndAt(event.End.Time)

		e.SetDtStampTime(time.Now()) // https://www.kanzaki.com/docs/ical/dtstamp.html

		b := strings.Builder{}
		b.WriteString(fmt.Sprintf("Docente: %s\n", event.Teacher))
		if len(event.Classrooms) > 0 {
			classroom := event.Classrooms[0]
			b.WriteString(fmt.Sprintf("Aula: %s\n", classroom.ResourceDesc))
			e.SetLocation(classroom.ResourceDesc)
		}
		b.WriteString(fmt.Sprintf("Cfu: %d\n", event.Cfu))
		b.WriteString(fmt.Sprintf("Periodo: %s\n", event.Interval))
		b.WriteString(fmt.Sprintf("Codice modulo: %s\n", event.CodModulo))

		e.SetDescription(b.String())
	}

	calName := fmt.Sprintf("%s - %d year", course.Descrizione, year)
	cal.SetName(calName)

	calDesc := fmt.Sprintf("Orario delle lezioni del %d anno del corso di %s",
		year, course.Descrizione)
	cal.SetDescription(calDesc)

	return cal, nil
}

func createExamsCal(exams []exams.Exam, title, description string) (*ics.Calendar, error) {
	cal := ics.NewCalendar()
	cal.SetMethod(ics.MethodRequest)

	for _, exam := range exams {
		sha := sha1.New()
		_, err := fmt.Fprintf(sha, "%s%s%s%s", exam.SubjectName, exam.Date, exam.Location, exam.Teacher)
		if err != nil {
			return nil, err
		}

		eventUid := fmt.Sprintf("%x", sha.Sum(nil))

		e := cal.AddEvent(eventUid)
		e.SetOrganizer(exam.Teacher)
		e.SetSummary(exam.SubjectName)
		e.SetStartAt(exam.Date)
		e.SetEndAt(exam.Date.Add(2 * time.Hour))
		e.SetLocation(exam.Location)

		e.SetDtStampTime(time.Now())

		b := strings.Builder{}
		b.WriteString(fmt.Sprintf("Docente: %s\n", exam.Teacher))
		b.WriteString(fmt.Sprintf("Codice: %s\n", exam.SubjectCode))
		b.WriteString(fmt.Sprintf("Tipo: %s\n", exam.Type))

		e.SetDescription(b.String())
	}

	cal.SetName(title)
	cal.SetDescription(description)

	return cal, nil
}
