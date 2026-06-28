package service

import "strings"

func MaskName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	runes := []rune(value)
	if len(runes) <= 1 {
		return "*"
	}

	return string(runes[0]) + strings.Repeat("*", len(runes)-1)
}

func MaskEmail(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	parts := strings.SplitN(value, "@", 2)
	if len(parts) != 2 {
		return "***"
	}

	local := []rune(parts[0])
	if len(local) == 0 {
		return "***@" + parts[1]
	}

	return string(local[0]) + "***@" + parts[1]
}

func MaskIP(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	if strings.Contains(value, ".") {
		parts := strings.Split(value, ".")
		if len(parts) == 4 {
			return parts[0] + "." + parts[1] + ".*.*"
		}
	}

	if strings.Contains(value, ":") {
		parts := strings.Split(value, ":")
		if len(parts) > 2 {
			return parts[0] + ":" + parts[1] + ":****"
		}
	}

	return "***"
}
