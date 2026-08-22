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

// LocalTimezone reports the host zone as an IANA name, which time.Local cannot
// supply because it renames whatever zone it loaded to "Local".
func LocalTimezone() string {
	return timezoneFrom(localtimePath)
}

// LocalLocation resolves that same name, so a rendered timestamp and the clock
// on a machine provisioned from this host never disagree.
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
