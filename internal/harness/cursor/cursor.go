package cursor

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/katasyst/fireconnect/internal/cli"
	"github.com/katasyst/fireconnect/internal/config"
	"github.com/katasyst/fireconnect/internal/fileutil"
	"github.com/katasyst/fireconnect/internal/fireworks"
	"github.com/katasyst/fireconnect/internal/harness"
	"github.com/katasyst/fireconnect/internal/ui"
)

const (
	blobKey   = "storage.blob.openai"
	secretKey = "storage.secret.openai"
	backupFileName = "provider-backup.json"
)

type openAIBlob struct {
	UseOpenAIKey  bool   `json:"useOpenAIKey"`
	OpenAIBaseURL string `json:"openAIBaseUrl"`
}

type cursorHarness struct{}

func init() {
	harness.Register(&cursorHarness{})
}

func (h *cursorHarness) ID() harness.HarnessID { return harness.Cursor }
func (h *cursorHarness) Label() string         { return "Cursor" }

func (h *cursorHarness) On(ctx *cli.HarnessContext) error {
	if err := harness.EngineOn(h, ctx); err != nil {
		return err
	}

	apiKey, err := harness.ResolveFireworksAPIKey(ctx)
	if err != nil {
		return err
	}

	if harness.IsCursorRunning() && !ctx.Force {
		return fmt.Errorf("Cursor IDE is running. Quit Cursor first.")
	}

	dbPath := stateDbPath(ctx)
	dataDir := dataDir(ctx)
	backupPath := filepath.Join(dataDir, backupFileName)

	blobRaw, blobErr := readSQLiteValue(dbPath, blobKey)
	secretRaw, secretErr := readSQLiteValue(dbPath, secretKey)
	if blobErr != nil {
		return blobErr
	}
	if secretErr != nil {
		return secretErr
	}

	if err := maybeWriteBackup(backupPath, blobRaw, secretRaw); err != nil {
		return err
	}

	newBlob, err := json.Marshal(openAIBlob{
		UseOpenAIKey:  true,
		OpenAIBaseURL: fireworks.FireworksBaseURL,
	})
	if err != nil {
		return err
	}

	if err := writeSQLiteValue(dbPath, blobKey, string(newBlob)); err != nil {
		return err
	}
	if err := writeSQLiteValue(dbPath, secretKey, apiKey); err != nil {
		return err
	}

	if err := config.SetHarnessEnabled(ctx.Home, string(h.ID()), true, "fireworks"); err != nil {
		return err
	}

	model := ctx.Main
	if model == "" {
		model = fireworks.DefaultMainModel
	}
	ui.PrintHarnessConnected(h.Label(), model)
	ui.PrintRestartHint("Quit & reopen Cursor IDE for the change to take effect.")
	return nil
}

func (h *cursorHarness) Off(ctx *cli.HarnessContext) error {
	if harness.IsCursorRunning() && !ctx.Force {
		return fmt.Errorf("Cursor IDE is running. Quit Cursor first.")
	}

	dbPath := stateDbPath(ctx)
	dataDir := dataDir(ctx)
	backupPath := filepath.Join(dataDir, backupFileName)

	backup, err := fileutil.ReadJSONMap(backupPath)
	if err != nil {
		return err
	}

	if snapshot, ok := backup["snapshot"].(map[string]interface{}); ok {
		blobVal, _ := snapshot["blob"].(string)
		keyVal, _ := snapshot["openAIKey"].(string)

		if blobVal != "" {
			if err := writeSQLiteValue(dbPath, blobKey, blobVal); err != nil {
				return err
			}
		} else {
			if err := deleteSQLiteValue(dbPath, blobKey); err != nil {
				return err
			}
		}

		if keyVal != "" {
			if err := writeSQLiteValue(dbPath, secretKey, keyVal); err != nil {
				return err
			}
		} else {
			if err := deleteSQLiteValue(dbPath, secretKey); err != nil {
				return err
			}
		}

		_ = os.Remove(backupPath)
	}

	if err := config.SetHarnessEnabled(ctx.Home, string(h.ID()), false, ""); err != nil {
		return err
	}

	ui.PrintHarnessRestored(h.Label())
	ui.PrintRestartHint("Quit & reopen Cursor IDE for full effect.")
	return nil
}

func (h *cursorHarness) Status(ctx *cli.HarnessContext) error {
	dbPath := stateDbPath(ctx)

	blobRaw, err := readSQLiteValue(dbPath, blobKey)
	if err != nil {
		return err
	}
	secretRaw, err := readSQLiteValue(dbPath, secretKey)
	if err != nil {
		return err
	}

	provider := "default"
	var baseURL string
	useKey := false

	if blobRaw != "" {
		var blob openAIBlob
		if json.Unmarshal([]byte(blobRaw), &blob) == nil {
			useKey = blob.UseOpenAIKey
			baseURL = blob.OpenAIBaseURL
			if useKey && baseURL == fireworks.FireworksBaseURL {
				provider = "fireworks"
			} else if useKey && baseURL != "" {
				provider = "custom"
			}
		}
	}

	enabled, err := config.IsHarnessEnabled(ctx.Home, string(h.ID()))
	if err != nil {
		return err
	}

	hasKey := secretRaw != "" && fireworks.IsFireworksKey(secretRaw)

	if ctx.JSON {
		payload := map[string]interface{}{
			"harness":      h.ID(),
			"enabled":      enabled,
			"provider":     provider,
			"useOpenAIKey": useKey,
			"baseUrl":      baseURL,
			"hasKey":       hasKey,
		}
		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	state := "off"
	if enabled {
		state = "on"
	}
	ui.Info(fmt.Sprintf("%s: %s · provider=%s", h.Label(), state, provider))
	if hasKey {
		ui.Info("  API key: configured")
	}
	return nil
}

func stateDbPath(ctx *cli.HarnessContext) string {
	if ctx.DBPath != "" {
		return ctx.DBPath
	}
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(ctx.Home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "Cursor", "User", "globalStorage", "state.vscdb")
	case "darwin":
		return filepath.Join(ctx.Home, "Library", "Application Support", "Cursor", "User", "globalStorage", "state.vscdb")
	default:
		configHome := os.Getenv("XDG_CONFIG_HOME")
		if configHome == "" {
			configHome = filepath.Join(ctx.Home, ".config")
		}
		return filepath.Join(configHome, "Cursor", "User", "globalStorage", "state.vscdb")
	}
}

func dataDir(ctx *cli.HarnessContext) string {
	if ctx.DataDir != "" {
		return ctx.DataDir
	}
	return filepath.Join(ctx.Home, ".fireconnect", "cursor")
}

func maybeWriteBackup(backupPath, blobRaw, openAIKey string) error {
	existing, err := fileutil.ReadJSONMap(backupPath)
	if err != nil {
		return err
	}
	if existing["snapshot"] != nil {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(backupPath), 0o700); err != nil {
		return err
	}
	return fileutil.WriteJSONFile(backupPath, map[string]interface{}{
		"snapshot": map[string]interface{}{
			"blob":      blobRaw,
			"openAIKey": openAIKey,
		},
	}, 0o600)
}

func openSQLite(dbPath string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func readSQLiteValue(dbPath, key string) (string, error) {
	if _, err := os.Stat(dbPath); errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	db, err := openSQLite(dbPath)
	if err != nil {
		return "", err
	}
	defer db.Close()

	var value string
	err = db.QueryRow(`SELECT value FROM ItemTable WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return "", nil
		}
		return "", err
	}
	return value, nil
}

func writeSQLiteValue(dbPath, key, value string) error {
	db, err := openSQLite(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := ensureItemTable(db); err != nil {
		return err
	}

	_, err = db.Exec(`INSERT OR REPLACE INTO ItemTable (key, value) VALUES (?, ?)`, key, value)
	return err
}

func deleteSQLiteValue(dbPath, key string) error {
	if _, err := os.Stat(dbPath); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	db, err := openSQLite(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(`DELETE FROM ItemTable WHERE key = ?`, key)
	if err != nil && strings.Contains(err.Error(), "no such table") {
		return nil
	}
	return err
}

func ensureItemTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS ItemTable (key TEXT PRIMARY KEY, value TEXT)`)
	return err
}
