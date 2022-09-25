package util

import (
	"net/url"
	"unsafe"
)

func GetInt16Representation(bytes []byte) []int16 {
	return unsafe.Slice((*int16)(unsafe.Pointer(&bytes[0])), len(bytes)/2)
}

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
