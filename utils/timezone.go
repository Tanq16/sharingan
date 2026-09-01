package utils

import (
	"os"
	"strings"
	"time"
)

const (
	defaultTimezone = "Etc/UTC"
	localtimePath   = "/etc/localtime"
)

// time.Local renames whatever zone it loaded to "Local", so the IANA name comes from the file instead.
func LocalTimezone() string {
	return timezoneFrom(localtimePath)
}

func LocalLocation() *time.Location {
	return locationFor(LocalTimezone())
}

func timezoneFrom(path string) string {
	link, err := os.Readlink(path)
	if err != nil {
		return defaultTimezone
	}
	if _, zone, found := strings.Cut(link, "/zoneinfo/"); found && zone != "" {
		return zone
	}
	return defaultTimezone
}

func locationFor(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.Local
	}
	return loc
}
