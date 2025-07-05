package connection

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

type Config struct {
	URL          string `json:"url"`
	Region       string `json:"region"`
	Bucket       string `json:"bucket"`
	StorageClass string `json:"storageClass"`
	KeyID        string `json:"keyID"`
	KeySecret    string `json:"keySecret"`

	AgePrivateKey string `json:"ageKey"`
}

type configV1 struct {
	URL          string `json:"u"`
	Region       string `json:"r"`
	Bucket       string `json:"b"`
	StorageClass string `json:"c"`
	KeyID        string `json:"k"`
	KeySecret    string `json:"ks"`

	AgePrivateKey string `json:"a"`
}

type versionedConfig struct {
	Version int             `json:"v"`
	Config  json.RawMessage `json:"d"`
}

type connectionError struct {
	in error
}

func (e connectionError) Error() string {
	return "Invalid connection string. Please make sure you copied it correctly."
}

func (e connectionError) Is(err error) bool {
	return e.in == err
}

func (e connectionError) Unwrap() error {
	return e.in
}

func ToString(config Config) (string, error) {
	data, err := json.Marshal(configV1(config))
	if err != nil {
		return "", fmt.Errorf("encode: %w", err)
	}

	versioned, err := json.Marshal(versionedConfig{Version: 1, Config: data})
	if err != nil {
		return "", fmt.Errorf("encode: %w", err)
	}

	return base64.RawStdEncoding.EncodeToString(versioned), nil
}

func FromString(encoded string) (Config, error) {
	jsonBytes, err := decodeBase64(encoded)
	if err != nil {
		return Config{}, connectionError{in: fmt.Errorf("decode connection string: %w", err)}
	}

	var versionedConfig versionedConfig
	err = json.Unmarshal(jsonBytes, &versionedConfig)
	if err != nil {
		return Config{}, connectionError{in: fmt.Errorf("parse connection string: %w", err)}
	}

	if versionedConfig.Version == 1 {
		var config configV1
		err = json.Unmarshal(versionedConfig.Config, &config)
		if err != nil {
			return Config{}, connectionError{in: fmt.Errorf("parse connection string: %w", err)}
		}
		return Config(config), nil
	} else {
		return Config{}, connectionError{in: fmt.Errorf("invalid version: %d", versionedConfig.Version)}
	}
}

func decodeBase64(encoded string) ([]byte, error) {
	r, err := base64.RawStdEncoding.DecodeString(encoded)
	if err == nil {
		return r, nil
	}

	r, err = base64.StdEncoding.DecodeString(encoded)
	if err == nil {
		return r, nil
	}

	r, err = base64.RawURLEncoding.DecodeString(encoded)
	if err == nil {
		return r, nil
	}

	r, err = base64.URLEncoding.DecodeString(encoded)
	if err == nil {
		return r, nil
	}

	return nil, err
}
