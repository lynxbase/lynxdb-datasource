package models

import (
	"encoding/json"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// Default field names and limits applied when the user leaves config blank.
const (
	DefaultTimeField    = "_time"
	DefaultMessageField = "_raw"
	DefaultLevelField   = "level"
	DefaultMaxLines     = 1000
)

// PluginSettings holds the datasource configuration sent from the config editor.
// The LynxDB base URL is taken from DataSourceInstanceSettings.URL, not from here.
type PluginSettings struct {
	DefaultIndex string                `json:"defaultIndex"`
	TimeField    string                `json:"timeField"`
	MessageField string                `json:"messageField"`
	LevelField   string                `json:"levelField"`
	MaxLines     int                   `json:"maxLines"`
	Secrets      *SecretPluginSettings `json:"-"`
}

// SecretPluginSettings holds encrypted fields decrypted on the backend only.
type SecretPluginSettings struct {
	Token string `json:"token"`
}

// LoadPluginSettings parses datasource settings and applies field defaults.
func LoadPluginSettings(source backend.DataSourceInstanceSettings) (*PluginSettings, error) {
	settings := PluginSettings{}
	if len(source.JSONData) > 0 {
		if err := json.Unmarshal(source.JSONData, &settings); err != nil {
			return nil, fmt.Errorf("could not unmarshal PluginSettings json: %w", err)
		}
	}

	if settings.TimeField == "" {
		settings.TimeField = DefaultTimeField
	}
	if settings.MessageField == "" {
		settings.MessageField = DefaultMessageField
	}
	if settings.LevelField == "" {
		settings.LevelField = DefaultLevelField
	}
	if settings.MaxLines <= 0 {
		settings.MaxLines = DefaultMaxLines
	}

	settings.Secrets = loadSecretPluginSettings(source.DecryptedSecureJSONData)

	return &settings, nil
}

func loadSecretPluginSettings(source map[string]string) *SecretPluginSettings {
	return &SecretPluginSettings{
		Token: source["token"],
	}
}
