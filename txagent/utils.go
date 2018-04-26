package txagent

import "os"

// GetEnv gets an environment variable or sets a default if
// one does not exist.
func GetEnv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}

	return value
}

// SetEnvIfEmpty sets an environment variable to itself or
// fallback if empty.
func SetEnvIfEmpty(env string, fallback string) (envVal string) {
	envVal = GetEnv(env, fallback)
	os.Setenv(env, envVal)

	return envVal
}
