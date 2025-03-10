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
	"github.com/cartabinaria/unibo-go/exams"

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

	coursesList := courses.ToList()
	slices.SortFunc(coursesList, func(a, b unibo_integ.Course) int {
		return b.Codice - a.Codice
	})
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index", gin.H{
			"courses": coursesList,
		})
	})

	r.GET("/courses/:id", coursePage(courses))

	r.GET("/cal/:id/:anno", getCoursesCal(&courses))

	r.GET("/exams/:id/:anno", getExams(&courses))
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

		cal, err := createCourseCal(courseTimetable, course, annoInt, subjects)
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

func getExams(courses *unibo_integ.CoursesMap) func(c *gin.Context) {
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
		isCurrValid := false
		if curriculumId != "" {
			curr.Value = curriculumId
			isCurrValid = true
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

		courseID, err := course.GetCourseWebsiteId()
		if err != nil {
			ctx.String(http.StatusInternalServerError, "Unable to get course website id")
			return
		}

		curricula, err := course.GetAllCurricula()
		if err != nil {
			_ = ctx.Error(fmt.Errorf("unable to retrieve curricula: %w", err))
			curricula = nil
		}

		subjectsMap, err := getSubjectsMapFromCourseAndCurricula(course, curricula)
		if err != nil {
			ctx.String(http.StatusInternalServerError, "Unable to get subjects for course and curricula")
			return
		}

		if isCurrValid {
			log.Printf("%v", curr)
			index := slices.IndexFunc([]curriculum.Curriculum(curricula[annoInt]), func(c curriculum.Curriculum) bool { return c.Value == curr.Value })
			if index == -1 {
				ctx.String(http.StatusBadRequest, "Invalid curriculum")
				return
			}

			curr = curricula[annoInt][index]
		} else {
			curr = curricula[annoInt][0]
		}
		validSubjects := subjectsMap[annoInt][curr]

		filteredValidSubjectsCodes := make([]string, 0)
		for _, s := range validSubjects {
			if slices.Contains(subjects, s.Code) || len(subjects) == 0 {
				// Some subject codes are not valid, because have the module number in the code.
				// Something like "04642_1". We need to extract only the first part.
				// TODO: Some codes are like "SPOT_79006" for the 8005 course. I've no idea what that means. We should check if they are valid on the exams period.
				filteredValidSubjectsCodes = append(filteredValidSubjectsCodes, strings.Split(s.Code, "_")[0])
			}
		}

		log.Debug().Any("validSubjects", validSubjects).Msg("validSubjects")

		allExams, err := exams.GetExams(courseID.Tipologia, courseID.Id)
		if err != nil {
			ctx.String(http.StatusInternalServerError, "Unable to get exams")
			return
		}

		log.Debug().Any("allExams", allExams).Msg("allExams")
		log.Debug().Any("filteredValidSubjectsCodes", filteredValidSubjectsCodes).Msg("filteredValidSubjectsCodes")

		filteredExams := make([]exams.Exam, 0)
		for _, exam := range allExams {
			if slices.Contains(filteredValidSubjectsCodes, exam.SubjectCode) {
				filteredExams = append(filteredExams, exam)
			}
		}

		log.Debug().Any("filteredExams", filteredExams).Msg("filteredExams")

		calName := fmt.Sprintf("Esami %d anno %s", annoInt, course.Descrizione)
		description := fmt.Sprintf("Esami del %d anno del corso di %s", annoInt, course.Descrizione)

		cal, err := createExamsCal(filteredExams, calName, description)
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
