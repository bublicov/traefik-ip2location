package traefik_ip2location

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/ip2location/ip2location-go"
)

// RedirectPerform is a special value indicating that need a redirect.
const RedirectPerform = "redirect_perform"

// Config the plugin configuration.
type Config struct {
	DBPath                      string              `json:"dbPath"`
	Locales                     []string            `json:"locales"`
	DefaultLocale               string              `json:"defaultLocale"`
	LanguageStrategy            string              `json:"languageStrategy"`
	LanguageParam               string              `json:"languageParam"`
	Redirect                    bool                `json:"redirect"`
	LanguageToCountriesOverride map[string][]string `json:"languageToCountriesOverride"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		Locales:                     []string{},
		DefaultLocale:               "",
		LanguageStrategy:            "header",
		LanguageParam:               "",
		Redirect:                    false,
		LanguageToCountriesOverride: make(map[string][]string),
	}
}

// GeoIP a plugin.
type GeoIP struct {
	db                          *ip2location.DB
	next                        http.Handler
	config                      *Config
	languageToCountriesOverride map[string][]string
	languageToCountriesDefault  map[string][]string
}

// New creates a new plugin.
func New(ctx context.Context, next http.Handler, config *Config) (http.Handler, error) {
	if config.DBPath == "" {
		return nil, fmt.Errorf("DBPath is required")
	}

	if config.DefaultLocale == "" {
		return nil, fmt.Errorf("DefaultLocale is required")
	}

	if len(config.Locales) == 0 {
		return nil, fmt.Errorf("locales are required")
	}

	db, err := ip2location.OpenDB(config.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open IP2Location database: %w", err)
	}

	return &GeoIP{
		db:                          db,
		next:                        next,
		config:                      config,
		languageToCountriesOverride: config.LanguageToCountriesOverride,
		languageToCountriesDefault:  createLanguageToCountriesMap(),
	}, nil
}

// ServeHTTP implements the http.Handler interface.
func (g *GeoIP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	locationData, err := g.getLocationData(r.RemoteAddr)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	locale := locationData.Country_short
	if locale == "-" || !contains(g.config.Locales, normalizeLocale(locale)) {
		locale = g.config.DefaultLocale
	}

	language := g.getLanguageByLocale(locale)
	if language != "-" {
		if strategy, err := g.getStrategy(); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		} else {
			if result := strategy.SetLanguage(w, r, language, g.config.Redirect); result == RedirectPerform {
				http.Redirect(w, r, r.URL.String(), http.StatusFound)
				return // Stop further execution if a redirect was performed
			}
		}
	}

	g.next.ServeHTTP(w, r)
}

// Close closes the IP2Location database.
func (g *GeoIP) Close() error {
	if g.db != nil {
		g.db.Close()
	}
	return nil
}

/* Handlers
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

type Strategy interface {
	SetLanguage(w http.ResponseWriter, r *http.Request, language string, redirect bool) interface{}
}

type HeaderStrategy struct {
}
type PathStrategy struct {
}
type QueryStrategy struct {
	LanguageParam string
}

func (h *HeaderStrategy) SetLanguage(w http.ResponseWriter, r *http.Request, language string, redirect bool) interface{} {
	if r.Header.Get("Accept-Language") == "" {
		r.Header.Set("Accept-Language", language)
	}
	return nil
}

func (p *PathStrategy) SetLanguage(w http.ResponseWriter, r *http.Request, language string, redirect bool) interface{} {
	if !strings.HasPrefix(r.URL.Path, "/"+language) {
		// Add the language prefix to the URL path
		r.URL.Path = "/" + language + strings.TrimPrefix(r.URL.Path, "/")
		// If redirect is enabled, perform a redirect to the updated URL
		if redirect {
			return RedirectPerform
		}
	}
	return nil
}

func (q *QueryStrategy) SetLanguage(w http.ResponseWriter, r *http.Request, language string, redirect bool) interface{} {
	if query := r.URL.Query(); query.Get(q.LanguageParam) == "" {
		// Set the language parameter in the query string
		query.Set(q.LanguageParam, language)
		r.URL.RawQuery = query.Encode()
		// If redirect is enabled, perform a redirect to the updated URL
		if redirect {
			return RedirectPerform
		}
	}
	return nil
}

/* Helpers
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func normalizeLocales(locales []string) []string {
	normalizedLocales := make([]string, len(locales))
	for i, locale := range locales {
		normalizedLocales[i] = normalizeLocale(locale)
	}
	return normalizedLocales
}

func normalizeLocale(locale string) string {
	return strings.ToUpper(locale)
}

func (g *GeoIP) getStrategy() (Strategy, error) {
	switch g.config.LanguageStrategy {
	case "header":
		return &HeaderStrategy{}, nil
	case "path":
		return &PathStrategy{}, nil
	case "query":
		if g.config.LanguageParam == "" {
			return nil, fmt.Errorf("LanguageParam is required when LanguageStrategy is 'query'")
		}
		return &QueryStrategy{LanguageParam: g.config.LanguageParam}, nil
	default:
		return nil, fmt.Errorf("invalid LanguageStrategy: %s", g.config.LanguageStrategy)
	}
}

func (g *GeoIP) getLocationData(remoteAddr string) (*ip2location.IP2Locationrecord, error) {
	ip, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return nil, fmt.Errorf("error parsing IP: %w", err)
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return nil, fmt.Errorf("invalid IP address")
	}

	results, err := g.db.Get_all(parsedIP.String())
	if err != nil {
		return nil, fmt.Errorf("error getting location data: %w", err)
	}

	return &results, nil
}

func (g *GeoIP) getLanguageByLocale(locale string) string {
	// Check the override map first
	for language, countries := range g.languageToCountriesOverride {
		for _, country := range countries {
			if country == locale {
				return language
			}
		}
	}

	// If not found in override, check the default map
	for language, countries := range g.languageToCountriesDefault {
		for _, country := range countries {
			if country == locale {
				return language
			}
		}
	}

	return "-"
}

func createLanguageToCountriesMap() map[string][]string {
	return map[string][]string{
		"en": {"US", "GB", "CA", "AU", "NZ", "IE", "ZA", "JM", "BS", "BZ", "BB", "TT", "GY", "SR", "VC", "AG", "KN", "LC", "GD", "TC", "VG", "KY", "BM", "VI", "PR", "GU", "AS", "MP", "UM", "IN", "PK", "SG", "MY", "NG", "PH"},
		"fr": {"FR", "BE", "CH", "DJ", "GQ", "CA", "CD", "CF", "CG", "CI", "CM", "KM", "GA", "GN", "HT", "LU", "MC", "MG", "ML", "MQ", "NC", "NE", "PF", "RE", "RW", "SC", "SN", "TD", "TG"},
		"es": {"ES", "MX", "AR", "CO", "PE", "CL", "VE", "GT", "CU", "BO", "DO", "EC", "HN", "NI", "PA", "PY", "SV", "UY", "CR", "PR", "GQ", "PH"},
		"de": {"DE", "AT", "CH", "LI", "LU"},
		"ru": {"RU", "BY", "KZ", "KG", "MD", "TJ", "TM", "UA", "UZ"},
		"zh": {"CN", "TW", "HK", "MO", "SG", "MY"},
		"ja": {"JP"},
		"it": {"IT", "SM", "VA", "CH"},
		"pt": {"PT", "BR", "AO", "CV", "GW", "MZ", "ST", "TL"},
		"nl": {"NL"},
		"pl": {"PL"},
		"tr": {"TR"},
		"ko": {"KR"},
		"sv": {"SE"},
		"no": {"NO"},
		"da": {"DK"},
		"fi": {"FI"},
		"el": {"GR"},
		"hu": {"HU"},
		"cs": {"CZ"},
		"sk": {"SK"},
		"ro": {"RO"},
		"bg": {"BG"},
		"sl": {"SI"},
		"lt": {"LT"},
		"lv": {"LV"},
		"et": {"EE"},
		"is": {"IS"},
		"he": {"IL"},
		"ar": {"DZ", "BH", "TD", "DJ", "EG", "IQ", "JO", "KW", "LB", "LY", "MR", "MA", "OM", "PS", "QA", "SA", "SO", "SD", "SS", "SY", "TN", "AE", "YE", "KM"},
	}
}
