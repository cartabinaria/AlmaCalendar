package main

import (
	"embed"
	_ "embed"
	fmt "fmt"
	ics "github.com/arran4/golang-ical"
	"io/fs"
	"net/http"
	"os"
	"sort"
	"strconv"
	"text/template"
	"unibocalendar/unibo"

	"github.com/gin-contrib/multitemplate"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	//go:embed templates/base.gohtml
	baseTemplate string
	//go:embed templates/index.gohtml
	indexTemplate string
	//go:embed templates/courses.gohtml
	coursesTemplate string

	//go:embed static/*
	staticFS embed.FS
)

func createMyRender() multitemplate.Renderer {
	funcMap := template.FuncMap{"anniRange": func(end int) []int {
		r := make([]int, 0, end)
		for i := 1; i <= end; i++ {
			r = append(r, i)
		}
		return r
	}}

	r := multitemplate.NewRenderer()

	r.AddFromString("base", baseTemplate)
	r.AddFromStringsFuncs("index", funcMap, baseTemplate, indexTemplate)
	r.AddFromStringsFuncs("courses", funcMap, baseTemplate, coursesTemplate)
	return r
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	downloadOpenDataIfNewer()

	var courses unibo.Courses
	courses, err := openData()
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to open open data file")
	}

	sort.Sort(courses)

	r := gin.Default()
	r.HTMLRender = createMyRender()

	staticFSSub, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to get sub staticFS")
	}
	r.StaticFS("/static", http.FS(staticFSSub))

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index", gin.H{})
	})
	r.GET("/courses", func(c *gin.Context) {
		c.HTML(http.StatusOK, "courses", gin.H{
			"courses": courses,
		})
	})

	r.GET("/cal/:id/:anno", getCoursesCal(&courses))

	err = r.Run()
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to start server")
	}
}

func getCoursesCal(courses *unibo.Courses) func(c *gin.Context) {
	return func(c *gin.Context) {
		id := c.Param("id")
		anno := c.Param("anno")

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

		// Try to retrieve timetable, otherwise return 500
		timetable, err := course.RetrieveTimetable(annoInt)
		if err != nil {
			_ = c.Error(err)
			c.String(http.StatusInternalServerError, "Unable to retrieve timetable")
			return
		}

		cal := createCal(timetable, course, annoInt)

		err = cal.SerializeTo(c.Writer)
		if err != nil {
			_ = c.Error(err)
			c.String(http.StatusInternalServerError, "Unable to serialize calendar")
			return
		}
		c.Status(http.StatusOK)

	}
}

func createCal(timetable unibo.Timetable, course *unibo.Course, year int) (cal *ics.Calendar) {
	cal = timetable.ToICS()
	cal.SetName(fmt.Sprintf("%s - %d year", course.Descrizione, year))
	cal.SetDescription(
		fmt.Sprintf("Orario delle lezioni del %d anno del corso di %s", year, course.Descrizione),
	)
	return
}
