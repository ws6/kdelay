package main

import (
	"time"

	"github.com/araddon/dateparse"
)

func praseLocalTimePtr(s string) (*time.Time, error) {
	ret, err := dateparse.ParseLocal(s)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func TryParseTime(s string) time.Time {
	ret := tryParseTime(s)
	now := time.Now()
	if ret.Year() == 0 {
		// hour, min, sec
		ret = time.Date(now.Year(), now.Month(), now.Day(),
			ret.Hour(), ret.Minute(), ret.Second(), ret.Nanosecond(), time.Local)
	}
	return ret.Local()

}

func tryParseTime(s string) time.Time {

	if ret, err := praseLocalTimePtr(s); err == nil && ret != nil {
		return *ret
	}

	return time.Now()
}
