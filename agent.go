package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/cjimti/iotagent/iotagent"
)

func main() {

	// Get environment vars or use as defaults if they do not exist
	cfgUrl := iotagent.SetEnvIfEmpty("AGENT_CFG_URL", "file://conf/defs.json")
	authUrl := iotagent.SetEnvIfEmpty("AGENT_AUTH_URL", "file://conf/auth.json")
	cfgPoll := iotagent.SetEnvIfEmpty("AGENT_CFG_POLL", "30")

	// cast poll to int
	cfgPollInt, err := strconv.Atoi(cfgPoll)
	if err != nil {
		panic(err)
	}

	// flag usage
	cfgPtrUsage  := " Location of json configuration file. Overrides AGENT_CFG_URL."
	authPtrUsage := " Location of json authentication file. Overrides AGENT_AUTH_URL."
	pollPtrUsage := " Poll every N seconds. Overrides AGENT_CFG_POLL."
	rmPtrUsage   := " Stop and remove containers defined in configuration."

	// use env vars as defaults for command line arguments.
	// command line arguments override environment variables.
	cfgPtr := flag.String("cfg", cfgUrl, cfgPtrUsage)
	authPtr := flag.String("auth", authUrl, authPtrUsage)
	pollPtr := flag.Int("poll", cfgPollInt, pollPtrUsage)
	rmPtr := flag.Bool("rm", false, rmPtrUsage)

	// parse flags
	flag.Parse()

	// get a new agent
	agent, err := iotagent.NewAgent(*cfgPtr, *authPtr, *pollPtr, iotagent.AgentOptions{
		LogOut: os.Stdout,
	})

	// stop and remove defined containers (exit application when complete)
	if *rmPtr {
		fmt.Printf("Removing all containers defined %s\n", cfgUrl)
		err = agent.StopRemoveContainers()
		if err != nil {
			panic(err)
		}

		os.Exit(0)
	}

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

	err = agent.CreateContainers()
	if err != nil {
		panic(err)
	}

	// Run
	err = agent.PollContainers()
	if err != nil {
		panic(err)
	}
}
