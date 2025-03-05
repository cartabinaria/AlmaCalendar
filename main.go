package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"github.com/cartabinaria/unibo-go/curriculum"

	"github.com/gin-contrib/multitemplate"
	limits "github.com/gin-contrib/size"
	"github.com/gin-gonic/gin"
	compress "github.com/lf4096/gin-compress"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/VaiTon/unibocalendar/unibo_integ"
)

//go:generate pnpm run css:build

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

	go fillSubjectsCache(courses)

	r := setupRouter(courses)

	err = r.Run()
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to start server")
	}
}

func setupRouter(courses unibo_integ.CoursesMap) *gin.Engine {
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
	slices.SortFunc(coursesList, func(a, b unibo_integ.Course) int {
		return b.Codice - a.Codice
	})
	r.GET("/courses", func(c *gin.Context) {
		c.HTML(http.StatusOK, "courses", gin.H{
			"courses": coursesList,
		})
	})

	r.GET("/courses/:id", coursePage(courses))

	r.GET("/cal/:id/:anno", getCoursesCal(&courses))
	return r
}

func coursePage(courses unibo_integ.CoursesMap) func(c *gin.Context) {
	return func(ctx *gin.Context) {
		courseId := ctx.Param("id")
		if courseId == "" {
			ctx.String(http.StatusBadRequest, "Invalid course id")
			return
		}

		courseIdInt, err := strconv.Atoi(courseId)
		if err != nil {
			ctx.String(http.StatusBadRequest, "Invalid course id")
			return
		}

		course, found := courses.FindById(courseIdInt)
		if !found {
			ctx.String(http.StatusNotFound, "Course not found")
			return
		}

		curricula, err := course.GetAllCurricula()
		if err != nil {
			_ = ctx.Error(fmt.Errorf("unable to retrieve curricula: %w", err))
			curricula = nil
		}

		m, err := getSubjectsMapFromCourseAndCurricula(course, curricula)
		if err != nil {
			_ = ctx.Error(fmt.Errorf("unable to retrieve subjects: %w", err))
		}

		ctx.HTML(http.StatusOK, "course", gin.H{
			"Course":    course,
			"Curricula": curricula,
			"Teachings": m,
		})
	}
}

func getCoursesCal(courses *unibo_integ.CoursesMap) func(c *gin.Context) {
	return func(ctx *gin.Context) {
		id := ctx.Param("id")
		anno := ctx.Param("anno")

		// Check if id is a number, otherwise return 400
		annoInt, err := strconv.Atoi(anno)
		if err != nil {
			ctx.String(http.StatusBadRequest, "Invalid year")
			return
		}

		// Check if id is a number, otherwise return 400
		idInt, err := strconv.Atoi(id)
		if err != nil {
			ctx.String(http.StatusBadRequest, "Invalid id")
			return
		}

		// Check if course exists, otherwise return 404
		course, found := courses.FindById(idInt)
		if !found {
			ctx.String(http.StatusNotFound, "Course not found")
			return
		}

		if annoInt <= 0 || annoInt > course.DurataAnni {
			ctx.String(http.StatusBadRequest, "Invalid year")
			return
		}

		curriculumId := ctx.Query("curr")
		curr := curriculum.Curriculum{}
		if curriculumId != "" {
			curr.Value = curriculumId
		}

		subjectIds := ctx.Query("subjects")
		var subjects []string
		if subjectIds != "" {
			tmp := strings.Split(subjectIds, ",")
			for i := range tmp {
				if len(tmp[i]) != 0 {
					subjects = append(subjects, tmp[i])
				}
			}
			log.Debug().Strs("subjects", subjects).Msg("queried subjects")
		}

		slices.Sort(subjects)

		cacheKey := fmt.Sprintf("%s-%s-%s-%s", id, anno, curr.Value, subjects)
		if cal, found := calcache.Get(cacheKey); found {
			successCalendar(ctx, cal.(*bytes.Buffer))
			return
		}

		// Try to retrieve timetable, otherwise return 500
		courseTimetable, err := course.GetTimetable(annoInt, curr, nil)
		if err != nil {
			_ = ctx.Error(err)
			ctx.String(http.StatusInternalServerError, "Unable to retrieve timetable")
			return
		}

		cal, err := createCal(courseTimetable, course, annoInt, subjects)
		if err != nil {
			_ = ctx.Error(err)
			ctx.String(http.StatusInternalServerError, "Unable to create calendar")
			return
		}

		buf := bytes.NewBuffer(nil)
		err = cal.SerializeTo(buf)
		if err != nil {
			_ = ctx.Error(err)
			ctx.String(http.StatusInternalServerError, "Unable to serialize calendar")
			return
		}

		calcache.Set(cacheKey, buf, cache.DefaultExpiration)

		successCalendar(ctx, buf)
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
