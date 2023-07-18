package unibo

import (
	"encoding/csv"
	"io"
	"strconv"
	"strings"
)

const (
	rootUnibo = "https://dati.unibo.it"
)

type Package struct {
	Success bool `json:"success"`
	Result  struct {
		Resources Resources
	}
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
