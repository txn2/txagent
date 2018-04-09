package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/cjimti/iotagent/iotagent"
)

func main() {

	// Get environment vars or use as defaults if they exist
	cfgUrl := iotagent.SetEnvIfEmpty("AGENT_CFG_URL", "file://example/defs.json")
	authUrl := iotagent.SetEnvIfEmpty("AGENT_AUTH_URL", "file://example/auth.json")
	cfgPoll := iotagent.SetEnvIfEmpty("AGENT_CFG_POLL", "30")

	// cast poll to int
	cfgPollInt, err := strconv.Atoi(cfgPoll)
	if err != nil {
		panic(err)
	}

	// use env vars as defaults for command line arguments.
	// command line arguments override environment variables.
	cfgPtr := flag.String("cfg", cfgUrl, " Location of json configuration file.")
	authPtr := flag.String("auth", authUrl, " Location of json authentication file.")
	pollPtr := flag.Int("poll", cfgPollInt, " Poll every N seconds.")
	rmPtr := flag.Bool("rm", false, " Stop and remove containers defined in "+cfgUrl)

	flag.Parse()

	// get a new agent
	agent, err := iotagent.NewAgent(*cfgPtr, *authPtr, *pollPtr)

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

	err = agent.PollContainers()
	if err != nil {
		panic(err)
	}
}