package unibo

import (
	"fmt"
	"strings"
)

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
