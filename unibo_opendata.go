package main

import (
	"encoding/json"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
	"unibocalendar/unibo"
)

const (
	coursesPathJson = "data/courses.json"
)

func downloadOpenDataIfNewer() {

	// Get package
	pack, err := unibo.GetPackage("degree-programmes")
	if err != nil {
		log.Panic().Err(err).Msg("Unable to get package")
	}

	// If no resources, return nil
	if len(pack.Result.Resources) == 0 {
		log.Panic().Msg("No resources found")
	}

	// Get wanted resource
	resource := pack.Result.Resources.GetByAlias("corsi_latest_it")
	if resource == nil {
		log.Panic().Msg("Unable to find resource")
	}

	// Get last modified resource
	lastMod := resource.LastMod

	// Parse last modified time
	lastModTime, err := time.Parse("2006-01-02T15:04:05.999999999", lastMod)
	if err != nil {
		log.Panic().Err(err).Msg("Unable to parse last modified time")
	}

	// Create data directory if not exists
	err = os.MkdirAll(path.Dir(coursesPathJson), os.ModePerm)
	if err != nil {
		log.Panic().Err(err).Msg("Unable to create data directory")
	}

	old := false
	// Get file last modified time, if file does not exist return lastMod.Url
	stat, err := os.Stat(coursesPathJson)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Panic().Err(err).Msg("Unable to get file stat")
		} else {
			old = true
		}
	}

	if !old && stat.ModTime().After(lastModTime) {
		log.Info().Msg("Opendata file is up to date")
		return
	}

	courses, err := resource.DownloadCourses()
	if err != nil {
		log.Panic().Err(err).Msg("Unable to download courses")
	}

	actualYear := time.Now().Year()

	// Filter courses by actual year
	courses = lo.Filter(courses, func(c unibo.Course, _ int) bool {
		return strings.Contains(c.AnnoAccademico, strconv.Itoa(actualYear))
	})

	jsonFile, err := os.Create(coursesPathJson)
	if err != nil {
		log.Panic().Err(err).Msg("Unable to create json file")
	}

	err = json.NewEncoder(jsonFile).Encode(courses)
	if err != nil {
		log.Panic().Err(err).Msg("Unable to encode json file")
	}

	log.Info().Msg("Opendata file downloaded")
}

func openOpenDataFile() (courses []unibo.Course, err error) {
	// Open file
	file, err := os.Open(coursesPathJson)
	if err != nil {
		return
	}

	// Decode json
	err = json.NewDecoder(file).Decode(&courses)
	if err != nil {
		return
	}

	// Close file
	err = file.Close()
	if err != nil {
		return
	}

	return
}
