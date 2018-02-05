package main

import (
	"iotagent/iotagent"
	"os"
	"strconv"
)

func main() {

	ctr_cfg_url := getEnv("AGENT_CFG_URL", "file://example/defs.json")
	ctr_cfg_poll, _ := strconv.Atoi(getEnv("AGENT_CFG_POLL", "30000"))

	_ = iotagent.NewAgent(ctr_cfg_url, ctr_cfg_poll)
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}

	return value
}
