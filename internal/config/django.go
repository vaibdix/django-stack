package config

import "strings"

type DjangoSeriesRule struct {
	Series   string
	MinMinor int
	MaxMinor int
}

var DjangoCompatibilityRules = []DjangoSeriesRule{
	{Series: "6.1", MinMinor: 12, MaxMinor: 14},
	{Series: "6.0", MinMinor: 12, MaxMinor: 14},
	{Series: "5.2", MinMinor: 10, MaxMinor: 14},
	{Series: "5.1", MinMinor: 10, MaxMinor: 13},
	{Series: "5.0", MinMinor: 10, MaxMinor: 12},
	{Series: "4.2", MinMinor: 8, MaxMinor: 12},
}

func DjangoSeries(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return version
	}
	return parts[0] + "." + parts[1]
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
