package unibo

import (
	"encoding/json"
	"fmt"
)

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
