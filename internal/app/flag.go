package app

import (
	"net/url"
	"strings"
)

type FlagError struct {
	buf strings.Builder
}

func (err *FlagError) Error() string {
	return err.buf.String()
}

type FlagURLValue struct {
	URL *url.URL
}

func (v FlagURLValue) String() string {
	if v.URL != nil {
		return v.URL.String()
	}
	return ""
}

func (v FlagURLValue) Set(s string) error {
	if u, err := url.Parse(s); err != nil {
		return err
	} else {
		*v.URL = *u
	}
	return nil
}
