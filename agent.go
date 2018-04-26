package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/txn2/txagent/txagent"
)

func main() {

	// Get environment vars or use as defaults if they do not exist
	cfgUrl := txagent.SetEnvIfEmpty("AGENT_CFG_URL", "file://conf/defs.json")
	authUrl := txagent.SetEnvIfEmpty("AGENT_AUTH_URL", "file://conf/auth.json")
	cfgPoll := txagent.SetEnvIfEmpty("AGENT_CFG_POLL", "30")

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
	agent, err := txagent.NewAgent(*cfgPtr, *authPtr, *pollPtr, txagent.AgentOptions{
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

	err = agent.Run()
	if err != nil {
		panic(err)
	}
}
