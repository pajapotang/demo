package util

import (
	"net/url"
	"testing"
)

func TestGetPageOK(t *testing.T) {
	cases := []struct {
		c      string
		expect int
	}{
		{
			c:      "https://localhost:9000/news?page=1",
			expect: 1,
		},
	}

	for _, c := range cases {
		requestURL, _ := url.Parse(c.c)
		page, _ := GetPage(requestURL)
		if page != c.expect {
			t.Errorf("Expecting %d but got %d", c.expect, page)
		}
	}
}

func TestGetPageFail(t *testing.T) {
	cases := []struct {
		c      string
		expect error
	}{
		{
			c:      "https://localhost:9000/news",
			expect: ErrParamPageEmpty,
		},
		{
			c:      "https://localhost:9000/news?page=0",
			expect: ErrParamPageValid,
		},
		{
			c:      "https://localhost:9000/news?page=",
			expect: ErrParamPageEmpty,
		},
	}

	for _, c := range cases {
		requestURL, _ := url.Parse(c.c)
		_, err := GetPage(requestURL)
		if err != c.expect {
			t.Errorf("Expecting %s but got %s", c.expect.Error(), err.Error())
		}
	}
}
