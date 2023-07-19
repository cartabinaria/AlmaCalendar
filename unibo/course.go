package unibo

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/patrickmn/go-cache"
)

type Course struct {
	AnnoAccademico       string
	Immatricolabile      string
	Codice               int
	Descrizione          string
	Url                  string
	Campus               string
	Ambiti               string
	Tipologia            string
	DurataAnni           int
	Internazionale       bool
	InternazionaleTitolo string
	InternazionaleLingua string
	Lingue               string
	Accesso              string
	SedeDidattica        string
}

type CourseId struct {
	Tipologia string
	Id        string
}

var websiteIdCache = cache.New(cache.NoExpiration, cache.NoExpiration)

// GetCourseWebsiteId returns the [CourseWebsiteId] of the course.
//
// If the course website id is already set, it returns it,
// otherwise it scrapes it from the course website.
func (c Course) GetCourseWebsiteId() (CourseId, error) {
	codeStr := strconv.Itoa(c.Codice)

	// If the course website id is already in the cache, return it
	websiteIdAny, found := websiteIdCache.Get(codeStr)
	if found {
		return websiteIdAny.(CourseId), nil
	}

	// Scrape the course website id and set it
	websiteId, err := c.scrapeCourseWebsiteId()
	if err != nil {
		return CourseId{}, err
	}

	websiteIdCache.Set(codeStr, websiteId, cache.DefaultExpiration)
	return websiteId, nil
}

var reg = regexp.MustCompile(`<a title="Sito del corso" href="https://corsi\.unibo\.it/(.+?)"`)

func (c Course) scrapeCourseWebsiteId() (CourseId, error) {

	resp, err := Client.Get(c.Url)
	if err != nil {
		return CourseId{}, err
	}

	buf := new(bytes.Buffer)

	// Read all body
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		return CourseId{}, err
	}

	// Close body
	err = resp.Body.Close()
	if err != nil {
		return CourseId{}, err
	}

	// Convert body to string
	found := reg.FindStringSubmatch(buf.String())
	if found == nil {
		return CourseId{}, fmt.Errorf("unable to find course website")
	}

	// full url -> laurea/IngegneriaInformatica
	id := found[1]

	// laurea/IngegneriaInformatica -> IngegneriaInformatica
	split := strings.Split(id, "/")
	return CourseId{split[0], split[1]}, nil
}

func (c Course) GetCurricula(year int) (Curricula, error) {
	id, err := c.GetCourseWebsiteId()
	if err != nil {
		return nil, err
	}

	curricula, err := FetchCurricula(id, year)
	if err != nil {
		return nil, err
	}

	return curricula, nil
}

func (c Course) GetAllCurricula() (map[int]Curricula, error) {
	id, err := c.GetCourseWebsiteId()
	if err != nil {
		return nil, fmt.Errorf("could not get course website id: %w", err)
	}

	currCh := make(chan Curricula)
	errCh := make(chan error)

	for year := 1; year <= c.DurataAnni; year++ {
		go func(year int) {
			curricula, err := FetchCurricula(id, year)
			if err != nil {
				errCh <- err
				return
			}
			currCh <- curricula
		}(year)
	}

	curriculaMap := make(map[int]Curricula, c.DurataAnni)
	for year := 1; year <= c.DurataAnni; year++ {
		select {
		case curricula := <-currCh:
			curriculaMap[year] = curricula
		case err := <-errCh:
			return nil, err
		}
	}

	return curriculaMap, nil
}

func (c Course) GetTimetable(year int, curriculum Curriculum, period *TimetablePeriod) (Timetable, error) {
	id, err := c.GetCourseWebsiteId()
	if err != nil {
		return nil, err
	}

	timetable, err := FetchTimetable(id, curriculum, year, period)
	if err != nil {
		return nil, err
	}

	return timetable, nil
}

type CoursesMap map[int]Course

func (c CoursesMap) ToList() Courses {
	courses := make([]Course, 0, len(c))
	for _, course := range c {
		courses = append(courses, course)
	}
	return courses
}

func (c CoursesMap) FindById(id int) (*Course, bool) {
	course, found := c[id]
	return &course, found
}

type Courses []Course

func (c Courses) Len() int {
	return len(c)
}

func (c Courses) Less(i, j int) bool {
	if c[i].AnnoAccademico != c[j].AnnoAccademico {
		return c[i].AnnoAccademico < c[j].AnnoAccademico
	}
	if c[i].Tipologia != c[j].Tipologia {
		return c[i].Tipologia < c[j].Tipologia
	}
	return c[i].Codice < c[j].Codice
}

func (c Courses) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}
