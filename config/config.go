package config

import "os"

type LLMProvider int
type MusicSource int

type Config struct {
	LLMProvider LLMProvider
	ModelName   string

	MusicSource MusicSource
}

const (
	OLLAMA LLMProvider = iota
)

const (
	SPOTIFY MusicSource = iota
)

var (
	providers = map[string]LLMProvider{
		"ollama": OLLAMA,
	}
	sources = map[string]MusicSource{
		"spotify": SPOTIFY,
	}
)

func getLLMProvider(provider string) LLMProvider {
	enumVal, ok := providers[provider]
	if !ok {
		return OLLAMA
	}

	return enumVal
}

func getMusicSource(source string) MusicSource {
	enumVal, ok := sources[source]
	if !ok {
		return SPOTIFY
	}

	return enumVal
}

func getEnvOrDefault(key string, defaultValue string) string {
	value, ok := os.LookupEnv(key)
	if ok {
		return value
	}

	return defaultValue
}

func GetConfig() Config {
	return Config{
		LLMProvider: getLLMProvider(os.Getenv("LLM_PROVIDER")),
		ModelName:   getEnvOrDefault("MODEL_NAME", "gemma3n:e4b"),
		MusicSource: getMusicSource(os.Getenv("MUSIC_SOURCE")),
	}
}
