package main

import (
	"iotagent/iotagent"
	"strconv"
)

func main() {

	ctr_cfg_url := iotagent.GetEnv("AGENT_CFG_URL", "file://example/defs.json")
	ctr_cfg_poll, _ := strconv.Atoi(iotagent.GetEnv("AGENT_CFG_POLL", "30000"))

	agent, err := iotagent.NewAgent(ctr_cfg_url, ctr_cfg_poll)

	err = agent.PullContainers()
	if err != nil {
		panic(err)
	}
}
