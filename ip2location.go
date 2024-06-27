package traefik_ip2location

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/ip2location/ip2location-go"
)

const StrategyHeader = "header"
const StrategyPath = "path"
const StrategyQuery = "query"

// Config the plugin configuration.
type Config struct {
	DBPath                      string              `yaml:"dbPath"`
	Languages                   []string            `yaml:"languages"`
	DefaultLanguage             string              `yaml:"defaultLanguage"`
	DefaultLanguageHandling     bool                `yaml:"defaultLanguageHandling"`
	LanguageStrategy            string              `yaml:"languageStrategy"`
	LanguageParam               string              `yaml:"languageParam"`
	RedirectAfterHandling       bool                `yaml:"redirectAfterHandling"`
	LanguageToCountriesOverride map[string][]string `yaml:"languageToCountriesOverride"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		Languages:                   []string{},
		DefaultLanguage:             "",
		DefaultLanguageHandling:     false,
		LanguageStrategy:            "header",
		LanguageParam:               "lang",
		RedirectAfterHandling:       false,
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
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if config.DBPath == "" {
		return nil, fmt.Errorf("DBPath is required")
	}

	if len(config.Languages) == 0 {
		return nil, fmt.Errorf("languages are required")
	}

	if config.DefaultLanguage == "" {
		return nil, fmt.Errorf("DefaultLanguage is required")
	}

	if config.LanguageStrategy == StrategyQuery && config.LanguageParam == "" {
		return nil, fmt.Errorf("languageParam is required when LanguageStrategy is 'query'")
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

	language := g.config.DefaultLanguage

	if locale := normalizeLocale(locationData.Country_short); locale != "-" {
		if languageByLocale := g.getLanguageByLocale(locale); contains(g.config.Languages, languageByLocale) {
			language = languageByLocale
		}
	}

	if language != g.config.DefaultLanguage || g.config.DefaultLanguageHandling {
		if strategy, err := g.getStrategy(); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		} else {
			// Maybe lang already exist
			languageByRequest := strategy.GetLanguage(r)
			// Set lang
			if languageByRequest == "" || !g.isLanguage(languageByRequest) {
				// Executing
				strategy.SetLanguage(w, r, language)
				// Stop further execution if a redirect perform
				if strategy.HasRedirectAfterHandling() {
					http.Redirect(w, r, r.URL.String(), http.StatusFound)
					return
				}
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
	GetLanguage(r *http.Request) string
	SetLanguage(w http.ResponseWriter, r *http.Request, language string)
	HasRedirectAfterHandling() bool
}

type HeaderStrategy struct {
	redirectAfterHandling bool
}

type PathStrategy struct {
	redirectAfterHandling bool
}

type QueryStrategy struct {
	redirectAfterHandling bool
	languageParam         string
}

func (h *HeaderStrategy) GetLanguage(r *http.Request) string {
	return r.Header.Get("Accept-Language")
}

func (h *HeaderStrategy) SetLanguage(w http.ResponseWriter, r *http.Request, language string) {
	r.Header.Set("Accept-Language", language)
}

func (h *HeaderStrategy) HasRedirectAfterHandling() bool {
	return h.redirectAfterHandling
}

func (p *PathStrategy) GetLanguage(r *http.Request) string {
	segments := strings.Split(r.URL.Path, "/")
	if len(segments) > 1 && len(segments[1]) == 2 {
		return segments[1]
	}
	return ""
}

func (p *PathStrategy) SetLanguage(w http.ResponseWriter, r *http.Request, language string) {
	if r.URL.Path == "/" {
		r.URL.Path = "/" + language
	} else {
		r.URL.Path = "/" + language + r.URL.Path
	}
}

func (p *PathStrategy) HasRedirectAfterHandling() bool {
	return p.redirectAfterHandling
}

func (q *QueryStrategy) GetLanguage(r *http.Request) string {
	query := r.URL.Query()
	return query.Get(q.languageParam)
}

func (q *QueryStrategy) SetLanguage(w http.ResponseWriter, r *http.Request, language string) {
	query := r.URL.Query()
	query.Set(q.languageParam, language)
	r.URL.RawQuery = query.Encode()
}

func (q *QueryStrategy) HasRedirectAfterHandling() bool {
	return q.redirectAfterHandling
}

/* Helpers
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

func (g *GeoIP) getStrategy() (Strategy, error) {
	switch g.config.LanguageStrategy {
	case StrategyHeader:
		return &HeaderStrategy{redirectAfterHandling: g.config.RedirectAfterHandling}, nil
	case StrategyPath:
		return &PathStrategy{redirectAfterHandling: g.config.RedirectAfterHandling}, nil
	case StrategyQuery:
		return &QueryStrategy{languageParam: g.config.LanguageParam, redirectAfterHandling: g.config.RedirectAfterHandling}, nil
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

func (g *GeoIP) isLanguage(lang string) bool {
	// Check the override map first
	for language := range g.languageToCountriesOverride {
		if language == lang {
			return true
		}
	}

	// If not found in override, check the default map
	for language := range g.languageToCountriesDefault {
		if language == lang {
			return true
		}
	}

	return false
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

func normalizeLocales(locales []string) []string {
	normalizedLocales := make([]string, len(locales))
	for i, locale := range locales {
		normalizedLocales[i] = normalizeLocale(locale)
	}
	return normalizedLocales
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func normalizeLocale(locale string) string {
	return strings.ToUpper(locale)
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
