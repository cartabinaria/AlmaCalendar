package main

import (
	"embed"
	_ "embed"
	"fmt"
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
	courses, err := openOpenDataFile()
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

	r.GET("/cal/:id/:anno", getCoursesCal(courses))

	err = r.Run()
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to start server")
	}
}

func getCoursesCal(courses unibo.Courses) func(c *gin.Context) {
	return func(c *gin.Context) {
		id := c.Param("id")
		anno := c.Param("anno")

		annoInt, err := strconv.Atoi(anno)
		if err != nil {
			_ = c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		var course *unibo.Course
		for _, c := range courses {
			if strconv.FormatInt(int64(c.Codice), 10) == id {
				course = &c
				break
			}
		}

		if course == nil {
			c.Status(http.StatusNotFound)
			return
		}

		timetable, err := course.RetrieveTimetable(annoInt)
		if err != nil {
			_ = c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		ics := unibo.Timetable(timetable).ToICS()

		calName := fmt.Sprintf("%s - %d anno", course.Descrizione, annoInt)
		ics.SetName(calName)

		calDesc := fmt.Sprintf("Orario delle lezioni del %d anno del corso di %s", annoInt, course.Descrizione)
		ics.SetDescription(calDesc)

		err = ics.SerializeTo(c.Writer)
		if err != nil {
			_ = c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.Status(http.StatusOK)

	}
}
