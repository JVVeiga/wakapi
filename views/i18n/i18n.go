package i18n

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
	"sync"
)

var (
	translations map[string]map[string]string
	defaultLang  string
	mu           sync.RWMutex
)

var languageLabels = map[string]string{
	"pt-BR": "Português (Brasil)",
	"en":    "English",
}

func Init(translationFS fs.FS, defaultLanguage string) error {
	mu.Lock()
	defer mu.Unlock()

	defaultLang = defaultLanguage
	translations = make(map[string]map[string]string)

	entries, err := fs.ReadDir(translationFS, ".")
	if err != nil {
		return fmt.Errorf("failed to read translation directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		lang := strings.TrimSuffix(entry.Name(), ".json")
		data, err := fs.ReadFile(translationFS, entry.Name())
		if err != nil {
			return fmt.Errorf("failed to read translation file %s: %w", entry.Name(), err)
		}

		var t map[string]string
		if err := json.Unmarshal(data, &t); err != nil {
			return fmt.Errorf("failed to parse translation file %s: %w", entry.Name(), err)
		}

		translations[lang] = t
	}

	return nil
}

func Translate(lang, key string) string {
	mu.RLock()
	defer mu.RUnlock()

	if t, ok := translations[lang]; ok {
		if v, ok := t[key]; ok {
			return v
		}
	}
	if t, ok := translations[defaultLang]; ok {
		if v, ok := t[key]; ok {
			return v
		}
	}
	return key
}

func SupportedLanguages() []string {
	return []string{"pt-BR", "en"}
}

func LanguageLabel(lang string) string {
	if label, ok := languageLabels[lang]; ok {
		return label
	}
	return lang
}

func IsSupported(lang string) bool {
	for _, l := range SupportedLanguages() {
		if l == lang {
			return true
		}
	}
	return false
}

func DefaultLanguage() string {
	mu.RLock()
	defer mu.RUnlock()
	return defaultLang
}
