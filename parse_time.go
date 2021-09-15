package main

import (
	"fmt"
	"strconv"
	"time"
)

func GetTimeLayouts() []string {
	return []string{
		"2006-01-02 15:04:05",
		"1/2/2006 3:04:05 PM",
		"1/2/2006 15:04:05",
		"1/2/2006 03:04:05",
		"1/2/2006 3:04:05",
		`2006-01-02 15:04:05.999999999 -0700 MST`, //time.String
		time.Layout,                               //  = "01/02 03:04:05PM '06 -0700" // The reference time, in numerical order.
		time.ANSIC,                                //= "Mon Jan _2 15:04:05 2006"
		time.UnixDate,                             //= "Mon Jan _2 15:04:05 MST 2006"
		time.RubyDate,                             //= "Mon Jan 02 15:04:05 -0700 2006"
		time.RFC822,                               //= "02 Jan 06 15:04 MST"
		time.RFC822Z,                              //= "02 Jan 06 15:04 -0700" // RFC822 with numeric zone
		time.RFC850,                               //= "Monday, 02-Jan-06 15:04:05 MST"
		time.RFC1123,                              //= "Mon, 02 Jan 2006 15:04:05 MST"
		time.RFC1123Z,                             //= "Mon, 02 Jan 2006 15:04:05 -0700" // RFC1123 with numeric zone
		time.RFC3339,                              //= "2006-01-02T15:04:05Z07:00"
		time.RFC3339Nano,                          //= "2006-01-02T15:04:05.999999999Z07:00"
		time.Kitchen,                              //= "3:04PM"
		// Handy time stamps.
		time.Stamp,      //= "Jan _2 15:04:05"
		time.StampMilli, //= "Jan _2 15:04:05.000"
		time.StampMicro, //= "Jan _2 15:04:05.000000"
		time.StampNano,  //= "Jan _2 15:04:05.000000000"
	}
}

func ParseFromUnix(s string) (*time.Time, error) {
	releaseAtUnix, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil, fmt.Errorf(`strconv:%s`, err.Error())
	}
	t := time.Unix(releaseAtUnix, 0)
	return &t, nil
}

func ParseOverLayouts(s string) (*time.Time, error) {

	for _, layout := range GetTimeLayouts() {

		// ret, err := time.Parse(layout, s)
		ret, err := time.ParseInLocation(layout, s, time.Local)
		if err == nil {
			return &ret, nil
		}

	}

	return nil, nil
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
	return ret
}

func tryParseTime(s string) time.Time {
	if ret, err := ParseFromUnix(s); err == nil {
		return *ret
	}

	if ret, err := ParseOverLayouts(s); err == nil {
		return *ret
	}

	return time.Now()
}
