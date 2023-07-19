package main

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"net/http"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	ics "github.com/arran4/golang-ical"
	"github.com/gin-contrib/multitemplate"
	limits "github.com/gin-contrib/size"
	"github.com/gin-gonic/gin"
	"github.com/lf4096/gin-compress"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/VaiTon/unibocalendar/unibo"
)

const templateDir = "./templates"

func createMyRender() multitemplate.Renderer {
	funcMap := template.FuncMap{"anniRange": func(end int) []int {
		r := make([]int, 0, end)
		for i := 1; i <= end; i++ {
			r = append(r, i)
		}
		return r
	}}

	r := multitemplate.NewRenderer()

	r.AddFromFiles("base", path.Join(templateDir, "base.gohtml"))
	r.AddFromFilesFuncs("index", funcMap,
		path.Join(templateDir, "index.gohtml"), path.Join(templateDir, "base.gohtml"),
	)
	r.AddFromFilesFuncs("courses", funcMap,
		path.Join(templateDir, "courses.gohtml"), path.Join(templateDir, "base.gohtml"),
	)
	r.AddFromFilesFuncs("course", funcMap,
		path.Join(templateDir, "course.gohtml"), path.Join(templateDir, "base.gohtml"),
	)
	return r
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	downloadOpenDataIfNewer()

	courses, err := openData()
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to open open data file")
	}

	r := setupRouter(courses)

	err = r.Run()
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to start server")
	}
}

func setupRouter(courses unibo.CoursesMap) *gin.Engine {
	r := gin.Default()
	r.Use(compress.Compress())
	// Limit payload to 10 MB. This fixes zip bombs.
	r.Use(limits.RequestSizeLimiter(10 * 1024 * 1024))
	r.HTMLRender = createMyRender()

	r.Static("/static", "./static")

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index", gin.H{})
	})

	coursesList := courses.ToList()
	sort.Sort(coursesList)
	r.GET("/courses", func(c *gin.Context) {
		c.HTML(http.StatusOK, "courses", gin.H{
			"courses": coursesList,
		})
	})

	r.GET("/courses/:id", coursePage(courses))

	r.GET("/cal/:id/:anno", getCoursesCal(&courses))
	return r
}

func coursePage(courses unibo.CoursesMap) func(c *gin.Context) {
	return func(c *gin.Context) {
		courseId := c.Param("id")
		if courseId == "" {
			c.String(http.StatusBadRequest, "Invalid course id")
			return
		}

		courseIdInt, err := strconv.Atoi(courseId)
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid course id")
			return
		}

		course, found := courses.FindById(courseIdInt)
		if !found {
			c.String(http.StatusNotFound, "Course not found")
			return
		}

		curricula, err := course.GetAllCurricula()
		if err != nil {
			_ = c.Error(fmt.Errorf("unable to retrieve curricula: %w", err))
			curricula = map[int]unibo.Curricula{}
		}

		c.HTML(http.StatusOK, "course", gin.H{
			"Course":    course,
			"Curricula": curricula,
		})
	}
}

var calcache = cache.New(time.Minute*10, time.Minute*30)

func getCoursesCal(courses *unibo.CoursesMap) func(c *gin.Context) {
	return func(c *gin.Context) {
		id := c.Param("id")
		anno := c.Param("anno")

		cacheKey := fmt.Sprintf("%s-%s", id, anno)
		if cal, found := calcache.Get(cacheKey); found {
			successCalendar(c, cal.(*bytes.Buffer))
			return
		}

		// Check if id is a number, otherwise return 400
		annoInt, err := strconv.Atoi(anno)
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid year")
			return
		}

		// Check if id is a number, otherwise return 400
		idInt, err := strconv.Atoi(id)
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid id")
			return
		}

		// Check if course exists, otherwise return 404
		course, found := courses.FindById(idInt)
		if !found {
			c.String(http.StatusNotFound, "Course not found")
			return
		}

		if annoInt <= 0 || annoInt > course.DurataAnni {
			c.String(http.StatusBadRequest, "Invalid year")
			return
		}

		curriculumId := c.Query("curriculum")
		curriculum := unibo.Curriculum{}
		if curriculumId != "" {
			curriculum.Value = curriculumId
		}

		// Try to retrieve timetable, otherwise return 500
		timetable, err := course.GetTimetable(annoInt, curriculum, nil)
		if err != nil {
			_ = c.Error(err)
			c.String(http.StatusInternalServerError, "Unable to retrieve timetable")
			return
		}

		cal := createCal(timetable, course, annoInt)
		buf := bytes.NewBuffer(nil)
		err = cal.SerializeTo(buf)
		if err != nil {
			_ = c.Error(err)
			c.String(http.StatusInternalServerError, "Unable to serialize calendar")
			return
		}
		calcache.Set(cacheKey, buf, cache.DefaultExpiration)

		successCalendar(c, buf)
	}
}

func successCalendar(c *gin.Context, cal *bytes.Buffer) {
	c.Header("Content-Type", "text/calendar; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=lezioni.ics")
	// Allow CORS
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, Authorization")
	c.Header("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")

	c.String(http.StatusOK, cal.String())
}

func createCal(timetable unibo.Timetable, course *unibo.Course, year int) (cal *ics.Calendar) {
	cal = toICS(timetable)
	cal.SetName(fmt.Sprintf("%s - %d year", course.Descrizione, year))
	cal.SetDescription(
		fmt.Sprintf("Orario delle lezioni del %d anno del corso di %s", year, course.Descrizione),
	)
	return
}

func toICS(t unibo.Timetable) *ics.Calendar {
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
