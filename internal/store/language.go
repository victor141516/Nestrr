package store

import "strings"

// NormalizeLanguage returns one stable BCP 47 representation for database
// keys and upstream TMDB requests. It intentionally covers the language,
// script, and region forms used by Nestrr without accepting duplicate casing.
func NormalizeLanguage(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "_", "-"))
	if value == "" {
		return "en-US"
	}
	parts := strings.Split(value, "-")
	parts[0] = strings.ToLower(parts[0])
	for index := 1; index < len(parts); index++ {
		part := strings.ToLower(parts[index])
		switch len(part) {
		case 2:
			part = strings.ToUpper(part)
		case 4:
			part = strings.ToUpper(part[:1]) + part[1:]
		}
		parts[index] = part
	}
	return strings.Join(parts, "-")
}
