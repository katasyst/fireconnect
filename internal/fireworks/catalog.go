package fireworks

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	gatewayURL     = "https://api.fireworks.ai"
	catalogPath    = "/v1/serverless/models"
	catalogUseCase = "coding"
	catalogTTL     = 1 * time.Hour
	cacheFileName  = "catalog-cache.json"
)

// CatalogEntry represents one model/router from the Fireworks serverless catalog.
type CatalogEntry struct {
	ID          string `json:"id"`
	ShortID     string `json:"shortId"`
	DisplayName string `json:"displayName"`
	Kind        string `json:"kind"`
}

type catalogCache struct {
	CachedAt int64          `json:"cachedAt"`
	Entries  []CatalogEntry `json:"entries"`
}

type apiPage struct {
	Models        []apiModel `json:"models"`
	NextPageToken string     `json:"nextPageToken"`
}

type apiModel struct {
	Name             string           `json:"name"`
	DisplayName      string           `json:"display_name"`
	DisplayNameCamel string           `json:"displayName"`
	ServerlessModes  []serverlessMode `json:"serverlessModes"`
	ServerlessModes2 []serverlessMode `json:"serverless_modes"`
}

func (m *apiModel) displayName() string {
	if m.DisplayNameCamel != "" {
		return m.DisplayNameCamel
	}
	return m.DisplayName
}

func (m *apiModel) modes() []serverlessMode {
	if len(m.ServerlessModes) > 0 {
		return m.ServerlessModes
	}
	return m.ServerlessModes2
}

type serverlessMode struct {
	UsageIdentifier  string `json:"usageIdentifier"`
	UsageIdentifier2 string `json:"usage_identifier"`
}

func (s *serverlessMode) usageID() string {
	if s.UsageIdentifier != "" {
		return s.UsageIdentifier
	}
	return s.UsageIdentifier2
}

func cacheFilePath(home string) string {
	return filepath.Join(home, ".fireconnect", cacheFileName)
}

func shortIDFromResource(name string) string {
	parts := strings.Split(name, "/")
	return parts[len(parts)-1]
}

// FetchCatalog fetches the model catalog from the Fireworks API, using a local cache with TTL.
func FetchCatalog(apiKey, home string, forceRefresh bool) ([]CatalogEntry, error) {
	if !forceRefresh {
		if entries, ok := readCatalogCache(home); ok {
			return entries, nil
		}
	}

	entries, err := fetchCatalogFromAPI(apiKey)
	if err != nil {
		if cached, ok := readCatalogCacheStale(home); ok {
			return cached, nil
		}
		return nil, err
	}

	writeCatalogCache(home, entries)
	return entries, nil
}

func fetchCatalogFromAPI(apiKey string) ([]CatalogEntry, error) {
	var allModels []apiModel
	pageToken := ""

	client := &http.Client{Timeout: 30 * time.Second}

	for {
		url := fmt.Sprintf("%s%s?format=nested&use_cases=%s", gatewayURL, catalogPath, catalogUseCase)
		if pageToken != "" {
			url += "&pageToken=" + pageToken
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetch Fireworks catalog: %w", err)
		}

		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			resp.Body.Close()
			return nil, fmt.Errorf("Fireworks API rejected the API key (%d). Check your key.", resp.StatusCode)
		}
		if resp.StatusCode != 200 {
			resp.Body.Close()
			return nil, fmt.Errorf("Fireworks API %d %s", resp.StatusCode, resp.Status)
		}

		var page apiPage
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decode catalog response: %w", err)
		}
		resp.Body.Close()

		allModels = append(allModels, page.Models...)
		if page.NextPageToken == "" {
			break
		}
		pageToken = page.NextPageToken
	}

	return buildCatalogEntries(allModels), nil
}

func buildCatalogEntries(models []apiModel) []CatalogEntry {
	seen := map[string]struct{}{}
	var entries []CatalogEntry

	for i := range models {
		m := &models[i]
		if m.Name == "" || !strings.Contains(m.Name, "/models/") {
			continue
		}

		shortID := shortIDFromResource(m.Name)
		dn := m.displayName()
		if dn == "" {
			dn = prettyModelName(shortID)
		}
		dn = strings.TrimSuffix(dn, " via Fireworks")

		if _, ok := seen[m.Name]; !ok {
			entries = append(entries, CatalogEntry{
				ID:          m.Name,
				ShortID:     shortID,
				DisplayName: dn,
				Kind:        "serverless",
			})
			seen[m.Name] = struct{}{}
		}

		for j := range m.modes() {
			uid := m.modes()[j].usageID()
			if uid == "" || !strings.Contains(uid, "/routers/") {
				continue
			}
			if _, ok := seen[uid]; ok {
				continue
			}
			rShortID := shortIDFromResource(uid)
			entries = append(entries, CatalogEntry{
				ID:          uid,
				ShortID:     rShortID,
				DisplayName: prettyModelName(rShortID),
				Kind:        "serverless",
			})
			seen[uid] = struct{}{}
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ShortID < entries[j].ShortID
	})
	return entries
}

func prettyModelName(shortID string) string {
	tokens := strings.FieldsFunc(shortID, func(r rune) bool { return r == '-' || r == '_' })
	out := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		if len(tok) <= 3 && isAllAlpha(tok) {
			out = append(out, strings.ToUpper(tok))
		} else if strings.Contains(tok, "p") && len(tok) <= 4 {
			out = append(out, strings.Replace(tok, "p", ".", 1))
		} else {
			out = append(out, strings.ToUpper(tok[:1])+tok[1:])
		}
	}
	return strings.Join(out, " ")
}

func isAllAlpha(s string) bool {
	for _, r := range s {
		if r < 'A' || (r > 'Z' && r < 'a') || r > 'z' {
			return false
		}
	}
	return true
}

// FilterCatalog filters catalog entries by a case-insensitive search term.
func FilterCatalog(entries []CatalogEntry, search string) []CatalogEntry {
	query := strings.ToLower(strings.TrimSpace(search))
	if query == "" {
		return entries
	}
	var out []CatalogEntry
	for _, e := range entries {
		if strings.Contains(strings.ToLower(e.ShortID), query) ||
			strings.Contains(strings.ToLower(e.DisplayName), query) ||
			strings.Contains(strings.ToLower(e.ID), query) {
			out = append(out, e)
		}
	}
	return out
}

func readCatalogCache(home string) ([]CatalogEntry, bool) {
	path := cacheFilePath(home)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var cache catalogCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, false
	}
	age := time.Since(time.UnixMilli(cache.CachedAt))
	if age > catalogTTL || len(cache.Entries) == 0 {
		return nil, false
	}
	return cache.Entries, true
}

func readCatalogCacheStale(home string) ([]CatalogEntry, bool) {
	path := cacheFilePath(home)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var cache catalogCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, false
	}
	if len(cache.Entries) == 0 {
		return nil, false
	}
	return cache.Entries, true
}

func writeCatalogCache(home string, entries []CatalogEntry) {
	cache := catalogCache{
		CachedAt: time.Now().UnixMilli(),
		Entries:  entries,
	}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return
	}
	dir := filepath.Dir(cacheFilePath(home))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	_ = os.WriteFile(cacheFilePath(home), data, 0o644)
}

// FormatCatalogTable formats catalog entries as an aligned table for terminal output.
func FormatCatalogTable(entries []CatalogEntry) string {
	if len(entries) == 0 {
		return "No models found."
	}

	idWidth := 8
	nameWidth := 12
	for _, e := range entries {
		if len(e.ShortID) > idWidth {
			idWidth = len(e.ShortID)
		}
		if len(e.DisplayName) > nameWidth {
			nameWidth = len(e.DisplayName)
		}
	}

	var b strings.Builder
	header := fmt.Sprintf("%-*s  %-*s", idWidth, "ID", nameWidth, "NAME")
	b.WriteString(header + "\n")

	routers := filterByKind(entries, "/routers/")
	models := filterByKind(entries, "/models/")

	if len(routers) > 0 {
		b.WriteString("\nROUTERS\n")
		for _, e := range routers {
			b.WriteString(fmt.Sprintf("%-*s  %s\n", idWidth, e.ShortID, e.DisplayName))
		}
	}
	if len(models) > 0 {
		b.WriteString("\nMODELS\n")
		for _, e := range models {
			b.WriteString(fmt.Sprintf("%-*s  %s\n", idWidth, e.ShortID, e.DisplayName))
		}
	}

	return b.String()
}

func filterByKind(entries []CatalogEntry, pathSegment string) []CatalogEntry {
	var out []CatalogEntry
	for _, e := range entries {
		if strings.Contains(e.ID, pathSegment) {
			out = append(out, e)
		}
	}
	return out
}

// ValidateCatalogAPIKey checks that we have a usable fw_ key for catalog fetch.
func ValidateCatalogAPIKey(key string) error {
	if key == "" {
		return errors.New("no API key. Run: fireconnect login --api-key <key>")
	}
	if !strings.HasPrefix(key, "fw_") {
		return errors.New("model catalog requires a standard fw_ key (Fire Pass fpk_ keys cannot list the catalog)")
	}
	return nil
}
