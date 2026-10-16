package config

import (
	"strconv"
	"strings"
	"time"
)

type DjangoSeriesRule struct {
	Series   string
	MinMinor int
	MaxMinor int
	LTS      bool

	SecurityUntil time.Time
}

func supportEnds(year int, month time.Month) time.Time {
	return time.Date(year, month, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0)
}

var DjangoCompatibilityRules = []DjangoSeriesRule{
	{Series: "6.1", MinMinor: 12, MaxMinor: 14, SecurityUntil: supportEnds(2027, time.December)},
	{Series: "6.0", MinMinor: 12, MaxMinor: 14, SecurityUntil: supportEnds(2027, time.April)},
	{Series: "5.2", MinMinor: 10, MaxMinor: 14, LTS: true, SecurityUntil: supportEnds(2028, time.April)},
}

func SupportedDjangoSeries(now time.Time) []DjangoSeriesRule {
	supported := make([]DjangoSeriesRule, 0, len(DjangoCompatibilityRules))
	for _, rule := range DjangoCompatibilityRules {
		if now.Before(rule.SecurityUntil) {
			supported = append(supported, rule)
		}
	}
	return supported
}

func NewestSupportedPythonMinor(now time.Time) int {
	newest := 0
	for _, rule := range SupportedDjangoSeries(now) {
		newest = max(newest, rule.MaxMinor)
	}
	return newest
}

func DjangoSeries(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return version
	}
	return parts[0] + "." + parts[1]
}

func DjangoSeriesAtLeast(djangoVersion string, major, minor int) bool {
	gotMajor, gotMinor, ok := parseDjangoSeries(djangoVersion)
	if !ok {
		return false
	}
	if gotMajor != major {
		return gotMajor > major
	}
	return gotMinor >= minor
}

func parseDjangoSeries(djangoVersion string) (int, int, bool) {
	majorText, minorText, ok := strings.Cut(DjangoSeries(djangoVersion), ".")
	if !ok {
		return 0, 0, false
	}
	major, err := strconv.Atoi(majorText)
	if err != nil {
		return 0, 0, false
	}
	minor, err := strconv.Atoi(minorText)
	if err != nil {
		return 0, 0, false
	}
	return major, minor, true
}

func DjangoSupportsNativeCSP(djangoVersion string) bool {
	return DjangoSeriesAtLeast(djangoVersion, 6, 0)
}

func DjangoSupportsMailers(djangoVersion string) bool {
	return DjangoSeriesAtLeast(djangoVersion, 6, 1)
}

func DjangoSupportsNinja(djangoVersion string) bool {
	return !DjangoSeriesAtLeast(djangoVersion, 6, 1)
}

func DjangoConstraintConditionKeyword(djangoVersion string) string {
	if DjangoSeriesAtLeast(djangoVersion, 5, 1) {
		return "condition"
	}
	return "check"
}

func PythonFloorForDjango(djangoVersion string) (int, bool) {
	series := DjangoSeries(djangoVersion)
	for _, rule := range DjangoCompatibilityRules {
		if rule.Series == series {
			return rule.MinMinor, true
		}
	}
	return 0, false
}
