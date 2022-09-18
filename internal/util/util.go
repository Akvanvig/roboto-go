package util

import (
	"net/url"
)

func ValidateUrl(rawUrl string) (string, error) {
	parsedUrl, err := url.Parse(rawUrl)

	if err != nil {
		return "", err
	}

	var parsedUrlStr string

	if parsedUrl.Scheme == "" {
		parsedUrlStr = parsedUrl.RequestURI()
	} else {
		parsedUrlStr = parsedUrl.Host + parsedUrl.RequestURI()
	}

	return parsedUrlStr, nil
}
