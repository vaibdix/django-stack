package config

import (
	"strconv"
	"strings"
	"time"

	"github.com/vaibdix/django-stack/internal/catalog"
)

type DjangoSeriesRule struct {
	Series   string
	MinMinor int
	MaxMinor int
	LTS      bool

	SecurityUntil time.Time
}

var DjangoCompatibilityRules = djangoRulesFrom(catalog.Load())

func djangoRulesFrom(c catalog.Catalog) []DjangoSeriesRule {
	rules := make([]DjangoSeriesRule, len(c.Django))
	for i, row := range c.Django {
		rules[i] = DjangoSeriesRule{Series: row.Series, MinMinor: row.Python[0], MaxMinor: row.Python[1], LTS: row.LTS, SecurityUntil: row.SecurityUntil()}
	}
	return rules
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

func djangoSeriesAtLeast(djangoVersion string, major, minor int) bool {
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
	return djangoSeriesAtLeast(djangoVersion, 6, 0)
}

func DjangoSupportsMailers(djangoVersion string) bool {
	return djangoSeriesAtLeast(djangoVersion, 6, 1)
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
