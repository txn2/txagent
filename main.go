package main

import (
	"iotagent/iotagent"
	"strconv"
)

func main() {

	ctrCfgUrl := iotagent.GetEnv("AGENT_CFG_URL", "file://example/defs.json")
	ctrCfgPoll, _ := strconv.Atoi(iotagent.GetEnv("AGENT_CFG_POLL", "30000"))

	agent, err := iotagent.NewAgent(ctrCfgUrl, ctrCfgPoll)

	err = agent.CreateVolumes()
	if err != nil {
		panic(err)
	}

	err = agent.CreateNetworks()
	if err != nil {
		panic(err)
	}

	err = agent.PullContainers()
	if err != nil {
		panic(err)
	}

}
