package common

import (
	"os"
	"strings"
)

const (
	EnvBaseURL      = "LAKEKEEPER_BASE_URL"
	EnvAuthMode     = "LAKEKEEPER_AUTH_MODE"
	EnvTokenURL     = "LAKEKEEPER_TOKEN_URL"
	EnvClientID     = "LAKEKEEPER_CLIENT_ID"
	EnvClientSecret = "LAKEKEEPER_CLIENT_SECRET"
	EnvScope        = "LAKEKEEPER_SCOPE"
	EnvAccessToken  = "LAKEKEEPER_ACCESS_TOKEN"
	EnvK8sTokenPath = "LAKEKEEPER_K8S_TOKEN_PATH"
	EnvBootstrap    = "LAKEKEEPER_BOOTSTRAP"
)

func GetEnvOr(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}

	return v
}

func GetEnvSlice(key, sep string, fallback []string) []string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}

	return strings.Split(v, sep)
}

func GetBoolEnv(key string) bool {
	v := os.Getenv(key)
	return strings.EqualFold(v, "true")
}
