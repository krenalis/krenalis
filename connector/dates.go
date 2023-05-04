//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package connector

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"chichi/apis/types"
)

// DateTime represents a date and time.
type DateTime struct {
	time.Time
}

// String returns the datetime formatted as "yyyy-mm-dd hh:mm:ss.nnnnnnnnn".
func (dt DateTime) String() string {
	return dt.Format("2006-01-02 15:04:05.999999999")
}

// ParseDateTime parses a datetime string. A datetime string has form
// "yyyy-mm-dd hh:mm:ss.nnnnnnnnn". For accepted values see the ParseDate and
// ParseTime functions.
func ParseDateTime(s string) (DateTime, error) {
	t, err := time.Parse(time.DateTime, s)
	if err != nil {
		return DateTime{}, err
	}
	if t.Year() == 0 {
		return DateTime{}, &time.ParseError{Layout: "2006-01-02", Value: s, LayoutElem: "2006", ValueElem: "0000", Message: ": year out of range"}
	}
	if strings.Contains(s, ",") {
		p := strings.LastIndex(s, ":")
		return DateTime{}, &time.ParseError{Layout: "2006-01-02 15:04:05.999999999", Value: s, LayoutElem: "05", ValueElem: s[p+1:]}
	}
	if len(s) > 29 {
		p := strings.LastIndex(s, ":")
		return DateTime{}, &time.ParseError{Layout: "2006-01-02 15:04:05.999999999", Value: s, LayoutElem: "05", ValueElem: s[p+1:]}
	}
	return DateTime{t}, nil
}

// AsDateTime returns a DateTime value corresponding to t.UTC(). It returns an
// error if the year is not in range [1, 9999].
func AsDateTime(t time.Time) (DateTime, error) {
	t = t.UTC()
	if y := t.Year(); y < 1 || y > 9999 {
		return DateTime{}, fmt.Errorf("year %d is out of range", y)
	}
	return DateTime{t}, nil
}

// MustDateTime is like AsDateTime but panics on error.
func MustDateTime(t time.Time) DateTime {
	dt, err := AsDateTime(t)
	if err != nil {
		panic(err)
	}
	return dt
}

// Date represents a date.
type Date struct {
	time.Time
}

// MarshalJSON implements the json.Marshaler interface.
func (d Date) MarshalJSON() ([]byte, error) {
	b := make([]byte, 1, 12)
	b[0] = '"'
	b = d.Time.AppendFormat(b, time.DateOnly)
	b = append(b, '"')
	return b, nil
}

// String returns the date formatted as "yyyy-mm-dd".
func (d Date) String() string {
	return d.Format("2006-01-02")
}

// ParseDate parses a date string. A date string has form "yyyy-mm-dd" with year
// in range [1, 9999], month in range [1, 12] and day in range [1, 31]. The date
// should also be an existing date.
func ParseDate(s string) (Date, error) {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return Date{}, err
	}
	if t.Year() == 0 {
		return Date{}, &time.ParseError{Layout: "2006-01-02", Value: s, LayoutElem: "2006", ValueElem: "0000", Message: ": year out of range"}
	}
	return Date{t}, nil
}

// AsDate returns a Date value corresponding to t with the location set UTC and
// with hours, minutes and seconds set to 0. It returns an error if the year is
// not in range [1, 9999].
func AsDate(t time.Time) (Date, error) {
	t = t.UTC()
	y := t.Year()
	if y < 1 || y > 9999 {
		return Date{}, fmt.Errorf("year %d is out of range", y)
	}
	return Date{time.Date(y, t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)}, nil
}

// MustDate is like AsDate but panics on error.
func MustDate(t time.Time) Date {
	d, err := AsDate(t)
	if err != nil {
		panic(err)
	}
	return d
}

// Time represents a time.
type Time int64

// Format returns a textual representation of the time value formatted according
// to the given layout. layout can be any layout supported by the types package.
func (t Time) Format(layout string) string {
	switch layout {
	case types.Seconds:
		return strconv.FormatInt(int64(t)/int64(time.Second), 10)
	case types.Milliseconds:
		return strconv.FormatInt(int64(t)/int64(time.Millisecond), 10)
	case types.Microseconds:
		return strconv.FormatInt(int64(t)/int64(time.Microsecond), 10)
	case types.Nanoseconds:
		return strconv.FormatInt(int64(t), 10)
	default:
		return time.Unix(0, int64(t)).Format(layout)
	}
}

// MarshalJSON implements the json.Marshaler interface.
func (t Time) MarshalJSON() ([]byte, error) {
	b := make([]byte, 1, 20)
	b[0] = '"'
	b = time.Unix(0, int64(t)).AppendFormat(b, "15:04:05.999999999")
	b = append(b, '"')
	return b, nil
}

// String returns the time formatted as "hh:mm:ss.nnnnnnnnn". Trailing zeros are
// omitted and, if all the decimals are zero, the decimal part is omitted.
func (t Time) String() string {
	return time.Unix(0, int64(t)).Format("15:04:05.999999999")
}

// refTime is the time "0000-01-01 00:00:00" in UTC.
var refTime = time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC)

// ParseTime parses a time string. A time string has form "hh:mm:ss.nnnnnnnnn"
// when the sub-second can contain from 1 to 9 digits or can be missing.
// hours must be in range [0, 23], minutes in range [0, 59] and seconds in range
// [0, 60).
func ParseTime(s string) (Time, error) {
	t, err := time.Parse(time.TimeOnly, s)
	if err != nil {
		return 0, err
	}
	if strings.Contains(s, ",") {
		p := strings.LastIndex(s, ":")
		return 0, &time.ParseError{Layout: "15:04:05.999999999", Value: s, LayoutElem: "05", ValueElem: s[p+1:]}
	}
	if len(s) > 18 {
		p := strings.LastIndex(s, ":")
		return 0, &time.ParseError{Layout: "15:04:05.999999999", Value: s, LayoutElem: "05", ValueElem: s[p+1:]}
	}
	return Time(t.Sub(refTime)), nil
}
