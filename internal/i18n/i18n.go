package i18n

import (
	"strings"
)

const ContextLocale = "locale"

// LocaleInfo describes a supported language.
type LocaleInfo struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	NativeName string `json:"native_name"`
	RTL        bool   `json:"rtl"`
}

var Supported = []LocaleInfo{
	{Code: "en", Name: "English", NativeName: "English", RTL: false},
	{Code: "am", Name: "Amharic", NativeName: "አማርኛ", RTL: false},
}

// Resolve picks the best locale from Accept-Language or explicit lang query.
func Resolve(explicit, acceptLanguage string) string {
	if code := normalize(explicit); code != "" {
		return code
	}
	for _, part := range strings.Split(acceptLanguage, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if i := strings.Index(part, ";"); i >= 0 {
			part = part[:i]
		}
		if code := normalize(part); code != "" {
			return code
		}
	}
	return "en"
}

func normalize(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	if code == "" {
		return ""
	}
	if i := strings.Index(code, "-"); i >= 0 {
		code = code[:i]
	}
	for _, loc := range Supported {
		if loc.Code == code {
			return code
		}
	}
	return ""
}

// Messages returns UI strings for a locale (fallback to English).
func Messages(locale string) map[string]string {
	if m, ok := catalogs[normalizeOrEn(locale)]; ok {
		return m
	}
	return catalogs["en"]
}

func normalizeOrEn(code string) string {
	if c := normalize(code); c != "" {
		return c
	}
	return "en"
}

var catalogs = map[string]map[string]string{
	"en": {
		"app.name":              "Asnakech School",
		"app.tagline":           "Online learning for everyone",
		"nav.courses":           "Courses",
		"nav.dashboard":         "Dashboard",
		"auth.login":            "Log in",
		"auth.register":         "Register",
		"course.enroll":         "Enroll",
		"course.checkout":       "Buy course",
		"search.placeholder":    "Search courses, teachers…",
		"recommendations.title": "Recommended for you",
		"parent.children.title": "Linked students",
		"error.not_found":       "Not found",
		"error.forbidden":       "Forbidden",
		"error.validation":      "Validation error",
	},
	"am": {
		"app.name":              "አስናከች ትምህርት",
		"app.tagline":           "ለሁሉም የመስመር ላይ ትምህርት",
		"nav.courses":           "ኮርሶች",
		"nav.dashboard":         "ዳሽቦርድ",
		"auth.login":            "ግባ",
		"auth.register":         "ተመዝገብ",
		"course.enroll":         "ተመዝገብ",
		"course.checkout":       "ኮርስ ግዛ",
		"search.placeholder":    "ኮርሶችን፣ መምህሮችን ፈልግ…",
		"recommendations.title": "ለእርስዎ የተመከሩ",
		"parent.children.title": "ተገናኝተው ተማሪዎች",
		"error.not_found":       "አልተገኘም",
		"error.forbidden":       "አልተፈቀደም",
		"error.validation":      "የማረጋገጫ ስህተት",
	},
}
