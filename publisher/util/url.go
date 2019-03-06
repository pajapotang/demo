package util

import (
	"errors"
	"net/url"
	"strconv"
)

var (
	ErrParamPageEmpty = errors.New("parameter page cannot be empty")
	ErrParamPageValid = errors.New("page parameter have must greater than 1")
)

func GetPage(url *url.URL) (int, error) {
	param := url.Query().Get("page")
	page, err := strconv.Atoi(param)

	if err != nil {
		return page, ErrParamPageEmpty
	}

	if page < 1 {
		return 1, ErrParamPageValid
	}

	return page, nil
}
