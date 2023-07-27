//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2018 Open2b
//

package culture

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var errInvalidTimeFormat = errors.New("culture: invalid time format")

var enUS *LocaleInfo

func init() {
	enUS = Locale("en-US")
}

// FormatTime formatta t secondo il formato format e nel locale indicato.
func FormatTime(t time.Time, format string, locale *LocaleInfo) string {
	return NewTimeFormatter(format, locale).Format(t)
}

// TimeFormatter rappresenta un formatter per i valori time.Time.
type TimeFormatter struct {
	format string
	locale *LocaleInfo
	toUTC  bool
}

// NewTimeFormatter ritorna un formatter che formatta un valore time.Time
// in formato format con il locale indicato.
func NewTimeFormatter(format string, locale *LocaleInfo) *TimeFormatter {
	toUTC := false
	if format == "" {
		format, locale, toUTC = standardformatTime("G", locale)
	} else if len(format) == 1 {
		format, locale, toUTC = standardformatTime(format, locale)
	}
	return &TimeFormatter{format, locale, toUTC}
}

var timeFormatReg = regexp.MustCompile(`%.|d{1,4}|f{1,9}|F{1,9}|h{1,2}|H{1,2}|K|m{1,2}|M{1,4}|s{1,2}|t{1,2}|y{1,5}|z{1,3}|:|/|\\.?|"[^"]*"|'[^']*'`)

// Format formatta t secondo il formato e nel locale di tf.
func (tf TimeFormatter) Format(t time.Time) string {
	if tf.toUTC {
		t = t.UTC()
	}
	return timeFormatReg.ReplaceAllStringFunc(tf.format, func(s string) string {
		return formatTime(t, s, tf.locale)
	})
}

// formatTime formatta t secondo il formato format nel locale indicato.
// Vedere: https://docs.microsoft.com/en-us/dotnet/standard/base-types/custom-date-and-time-format-strings
func formatTime(t time.Time, format string, locale *LocaleInfo) string {

	hasPercent := format[0] == '%'
	if hasPercent {
		format = format[1:]
		if format == "%" {
			panic(errInvalidTimeFormat)
		}
	}

	switch format {
	case "d", "dd":
		day := strconv.Itoa(t.Day())
		if len(format) == 2 && len(day) == 1 {
			return "0" + day
		}
		return day
	case "ddd":
		return locale.AbbreviatedDayName(t.Weekday())
	case "dddd":
		return locale.DayName(t.Weekday())
	case "f", "ff", "fff", "ffff", "fffff", "ffffff", "fffffff", "ffffffff", "fffffffff":
		ns := fmt.Sprintf("%09d", t.Nanosecond())
		return ns[:len(format)]
	case "F", "FF", "FFF", "FFFF", "FFFFF", "FFFFFF", "FFFFFFF", "FFFFFFFF", "FFFFFFFFF":
		ns := fmt.Sprintf("%09d", t.Nanosecond())
		s := ns[:len(format)]
		for i := 0; i < len(s); i++ {
			if s[i] != '0' {
				return s
			}
		}
		return ""
	case "h", "hh":
		h := strconv.Itoa(t.Hour() % 12)
		if h == "0" {
			h = "12"
		}
		if len(format) == 2 && len(h) == 1 {
			return "0" + h
		}
		return h
	case "H", "HH":
		h := strconv.Itoa(t.Hour())
		if len(format) == 2 && len(h) == 1 {
			return "0" + h
		}
		return h
	case "K":
		if t.Location() == time.UTC {
			return "Z"
		}
		return formatTime(t, "zzz", locale)
	case "m", "mm":
		minute := strconv.Itoa(t.Minute())
		if format == "mm" && len(minute) == 1 {
			return "0" + minute
		}
		return minute
	case "M", "MM":
		month := strconv.Itoa(int(t.Month()))
		if format == "MM" && len(month) == 1 {
			return "0" + month
		}
		return month
	case "MMM":
		return locale.AbbreviatedMonthName(t.Month())
	case "MMMM":
		return locale.MonthName(t.Month())
	case "s", "ss":
		second := strconv.Itoa(t.Second())
		if format == "ss" && len(second) == 1 {
			return "0" + second
		}
		return second
	case "t", "tt":
		var d string
		if t.Hour() < 12 {
			d = locale.AMDesignator()
		} else {
			d = locale.PMDesignator()
		}
		if len(format) == 1 {
			// ritorna solo il primo carattere
			for _, c := range d {
				return string(c)
			}
		}
		return d
	case "y", "yy":
		year := strconv.Itoa(t.Year() % 100)
		if format == "yy" && len(year) == 1 {
			return "0" + year
		}
		return year
	case "yyy", "yyyy", "yyyyy":
		year := strconv.Itoa(t.Year())
		if len(format) <= len(year) {
			return year
		}
		return strings.Repeat("0", len(format)-len(year)) + year
	case "z", "zz":
		_, offset := t.Zone()
		s := "+"
		if offset < 0 {
			s = "-"
			offset = -offset
		}
		zone := strconv.Itoa(offset / 3600) // convert to hours
		if len(format) == 2 && len(zone) == 1 {
			return s + "0" + zone
		}
		return s + zone
	case "zzz":
		_, offset := t.Zone()
		s := "+"
		if offset < 0 {
			s = "-"
			offset = -offset
		}
		return fmt.Sprintf("%s%02d:%02d", s, offset/3600, offset%60)
	case ":":
		return locale.TimeSeparator()
	case "/":
		return locale.DateSeparator()
	default:
		if format[0] == '\\' {
			if len(format) == 1 {
				panic(errInvalidTimeFormat)
			}
			return format[1:2]
		}
		if len(format) >= 2 {
			if format[0] == '"' || format[0] == '\'' {
				if format[len(format)-1] != format[0] {
					panic(errInvalidTimeFormat)
				}
				return format[1 : len(format)-1]
			}
		}
		return format
	}

}

// standardformatTime converte un formato standard in uno custom.
func standardformatTime(format string, locale *LocaleInfo) (string, *LocaleInfo, bool) {

	switch format {
	case "d":
		return locale.ShortDatePattern(), locale, false
	case "D":
		return locale.LongDatePattern(), locale, false
	case "f":
		return locale.LongDatePattern() + " " + locale.ShortTimePattern(), locale, false
	case "F":
		return locale.FullDateTimePattern(), locale, false
	case "g":
		return locale.ShortDatePattern() + " " + locale.ShortTimePattern(), locale, false
	case "G":
		return locale.ShortDatePattern() + " " + locale.LongTimePattern(), locale, false
	case "m", "M":
		return locale.MonthDayPattern(), locale, false
	case "o":
		return "yyyy'-'MM'-'dd'T'HH':'mm':'ss.fffffffK", locale, false
	case "r", "R":
		return "ddd, dd MMM yyyy HH':'mm':'ss 'GMT'", enUS, false
	case "s":
		return "yyyy'-'MM'-'dd'T'HH':'mm':'ss", locale, false
	case "t":
		return locale.ShortTimePattern(), locale, false
	case "T":
		return locale.LongTimePattern(), locale, false
	case "u":
		return "yyyy'-'MM'-'dd HH':'mm':'ss'Z'", locale, false
	case "U":
		return locale.FullDateTimePattern(), locale, true
	case "y", "Y":
		return locale.YearMonthPattern(), locale, false
	}

	panic(errInvalidTimeFormat)

}
