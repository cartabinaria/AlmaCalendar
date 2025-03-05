package main

import (
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/cartabinaria/unibo-go/curriculum"
	"github.com/cartabinaria/unibo-go/timetable"

	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog/log"

	"github.com/VaiTon/unibocalendar/unibo_integ"
)

var (
	calcache                    = cache.New(time.Minute*10, time.Minute*30)
	subjectsCacheExpirationTime = time.Hour * 4
	subjectsCache               = cache.New(subjectsCacheExpirationTime, time.Hour*6)
)

type subjectMap = map[int]map[curriculum.Curriculum][]timetable.SimpleSubject

// The return type is a map that for every year of the course map a curriculum
// to a slice of subjects
func getSubjectsMapFromCourseAndCurricula(course *unibo_integ.Course, curricula map[int]curriculum.Curricula) (subjectMap, error) {
	if course == nil {
		return nil, fmt.Errorf("course parameter is nil")
	}

	// To get a curricula from a course we need fetch from the unibo API. Sometimes
	// this could fail, so the curricula is nil. We need to check to avoid crashing
	// the program.
	if curricula == nil {
		return nil, fmt.Errorf("curricula parameter is nil")
	}

	m := make(subjectMap)
	for y, cs := range curricula {
		m[y] = make(map[curriculum.Curriculum][]timetable.SimpleSubject)
		for _, c := range cs {

			var subjects []timetable.SimpleSubject
			key := fmt.Sprintf("%d-%d-%s", course.Codice, y, c.Value)
			if t, found := subjectsCache.Get(key); found {
				m[y][c] = t.([]timetable.SimpleSubject)
				continue
			}

			courseTimetable, err := course.GetTimetable(y, c, nil)
			if err != nil {
				// Can't do much. We return nil so the caller can retry
				return nil, fmt.Errorf("unable to retrieve timetable for subjects: %w", err)
			}

			subjects = courseTimetable.GetSubjects()
			subjectsCache.Set(key, subjects, cache.DefaultExpiration)

			sort.Slice(subjects, func(i, j int) bool {
				return subjects[i].Name < subjects[j].Name
			})

			m[y][c] = subjects
		}
	}

	return m, nil
}

func filterTimetableBySubjects(t timetable.Timetable, codes []string) timetable.Timetable {
	filtered := make([]timetable.Event, 0, len(t))
	for _, event := range t {
		if slices.Contains(codes, event.CodModulo) {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

// This functions calls getSubjectsMapFromCourseAndCurricula for every course,
// so the cache is always full and users do not see a slow site
func fillSubjectsCache(courses unibo_integ.CoursesMap) {
	// This is to make sure everything is started
	time.Sleep(time.Second * 5)

	for _, course := range courses {
		log.Debug().Int("course-code", course.Codice).Str("course-name", course.Descrizione).Msg("queried subjects")

		curricula, err := course.GetAllCurricula()
		if err != nil {
			log.Err(err).Int("course-code", course.Codice).Str("course-name", course.Descrizione).Msg("Can't get curricula in workerfor course")
			continue
		}
		_, err = getSubjectsMapFromCourseAndCurricula(&course, curricula)
		if err != nil {
			log.Err(err).Msg("Can't subjects in worker")
			continue
		}

		time.Sleep(time.Second * 30)
	}

	time.Sleep(subjectsCacheExpirationTime)
}
