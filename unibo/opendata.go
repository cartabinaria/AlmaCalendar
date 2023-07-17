package unibo

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog/log"
)

const (
	rootUnibo = "https://dati.unibo.it"
)

var (
	reg = regexp.MustCompile(`<a title="Sito del corso" href="https://corsi\.unibo\.it/(.+?)"`)

	// Client is the http client used to make requests.
	// It is used to set a custom User-Agent.
	Client = http.Client{
		Transport: &transport{
			http.DefaultTransport,
		},
	}
)

type transport struct {
	http.RoundTripper
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", "CalendarBot")
	return t.RoundTripper.RoundTrip(req)
}

type Package struct {
	Success bool `json:"success"`
	Result  struct {
		Resources Resources
	}
}

type Resources []Resource

func (r Resources) GetByAlias(alias string) *Resource {
	for _, resource := range r {
		// Some resources have multiple aliases
		rAliases := strings.Split(resource.Alias, ", ")

		// Check if the alias is one of the aliases of the resource
		for _, rAlias := range rAliases {
			if rAlias == alias {
				return &resource
			}
		}
	}
	return nil
}

type Resource struct {
	Frequency string `json:"frequency"`
	Url       string `json:"url"`
	Id        string `json:"id"`
	PackageId string `json:"package_id"`
	LastMod   string `json:"last_modified"`
	Alias     string `json:"alias"`
}

func (r Resource) Download() ([]Course, error) {
	// Get the resource
	res, err := Client.Get(r.Url)
	if err != nil {
		return nil, err
	}

	// Parse the body
	var courses []Course
	if strings.HasSuffix(r.Url, ".csv") {
		courses, err = r.downloadCSV(res.Body)
	}
	if err != nil {
		return nil, err
	}

	// Close the body
	err = res.Body.Close()
	if err != nil {
		return nil, err
	}

	if courses == nil {
		return nil, fmt.Errorf("resource is not a csv file")
	}

	return courses, nil
}

func (r Resource) downloadCSV(body io.Reader) ([]Course, error) {
	courses := make([]Course, 0, 100)

	reader := csv.NewReader(body)

	// Skip first line
	_, err := reader.Read()
	if err != nil {
		return nil, err
	}

	for {
		row, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, err
			}
		}

		code, err := strconv.ParseInt(row[2], 10, 32)
		if err != nil {
			return nil, err
		}

		years, err := strconv.ParseInt(row[9], 10, 32)
		if err != nil {
			return nil, err
		}

		international, err := strconv.ParseBool(row[10])
		if err != nil {
			return nil, err
		}

		courses = append(courses, Course{
			AnnoAccademico:       row[0],
			Immatricolabile:      row[1],
			Codice:               int(code),
			Descrizione:          row[3],
			Url:                  row[4],
			Campus:               row[5],
			SedeDidattica:        row[6],
			Ambiti:               row[7],
			Tipologia:            row[8],
			DurataAnni:           int(years),
			Internazionale:       international,
			InternazionaleTitolo: row[11],
			InternazionaleLingua: row[12],
			Lingue:               row[13],
			Accesso:              row[14],
		})
	}
	return courses, nil
}

func GetPackageUrl(id string) string {
	return fmt.Sprintf("%s/api/3/action/package_show?id=%s", rootUnibo, id)
}

func GetPackage(id string) (*Package, error) {
	url := GetPackageUrl(id)

	html, err := Client.Get(url)
	if err != nil {
		return nil, err
	}

	body := html.Body
	pack := Package{}

	err = json.NewDecoder(body).Decode(&pack)
	if err != nil {
		return nil, err
	}

	err = body.Close()
	if err != nil {
		return nil, err
	}

	return &pack, nil
}

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

type CourseWebsiteId struct {
	Tipologia string
	Id        string
}

var websiteIdCache = cache.New(cache.NoExpiration, cache.NoExpiration)

// GetCourseWebsiteId returns the [CourseWebsiteId] of the course.
//
// If the course website id is already set, it returns it,
// otherwise it scrapes it from the course website.
func (c Course) GetCourseWebsiteId() (CourseWebsiteId, error) {
	codeStr := strconv.Itoa(c.Codice)

	// If the course website id is already in the cache, return it
	websiteIdAny, found := websiteIdCache.Get(codeStr)
	if found {
		return websiteIdAny.(CourseWebsiteId), nil
	}

	// Scrape the course website id and set it
	websiteId, err := c.scrapeCourseWebsiteId()
	if err != nil {
		return CourseWebsiteId{}, err
	}

	websiteIdCache.Set(codeStr, websiteId, cache.DefaultExpiration)
	return websiteId, nil
}

func (c Course) scrapeCourseWebsiteId() (CourseWebsiteId, error) {
	log.Debug().Str("course", c.Descrizione).Msg("scraping course website id")

	resp, err := Client.Get(c.Url)
	if err != nil {
		return CourseWebsiteId{}, err
	}

	buf := new(bytes.Buffer)

	// Read all body
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		return CourseWebsiteId{}, err
	}

	// Close body
	err = resp.Body.Close()
	if err != nil {
		return CourseWebsiteId{}, err
	}

	// Convert body to string
	found := reg.FindStringSubmatch(buf.String())
	if found == nil {
		return CourseWebsiteId{}, fmt.Errorf("unable to find course website")
	}

	// full url -> laurea/IngegneriaInformatica
	id := found[1]

	// laurea/IngegneriaInformatica -> IngegneriaInformatica
	split := strings.Split(id, "/")
	return CourseWebsiteId{split[0], split[1]}, nil
}

func (c Course) GetTimetable(anno int) (Timetable, error) {
	id, err := c.GetCourseWebsiteId()
	if err != nil {
		return nil, err
	}

	timetable, err := FetchTimetable(id, anno)
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
