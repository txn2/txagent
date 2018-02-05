package iotagent

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/bhoriuchi/go-bunyan/bunyan"
	"github.com/docker/docker/api/types/container"
	//	"github.com/docker/docker/client"
)

type Agent struct {
	cfgUrl     string
	poll       int
	log        bunyan.Logger
	containers *[]container.Config
}

// NewAgent creates a new agent from a configuration url and a polling interval
func NewAgent(cfgUrl string, poll int) Agent {

	logConfig := bunyan.Config{
		Name:   "iotagent",
		Stream: os.Stdout,
		Level:  bunyan.LogLevelDebug,
	}

	logger, err := bunyan.CreateLogger(logConfig)
	if err != nil {
		panic(err)
	}

	logger.Info("Loading IoT Agent...")

	agent := Agent{
		cfgUrl: cfgUrl,
		poll:   poll,
		log:    logger,
	}

	cfgJson := agent.loadCfg()
	agent.marshalCfg(cfgJson)

	return agent
}

func (agent *Agent) marshalCfg(cfgJson []byte) {

	cfgContainers := make([]container.Config, 0)
	err := json.Unmarshal(cfgJson, &cfgContainers)
	if err != nil {
		agent.log.Error(err.Error())
		return
	}

	agent.log.Info("Found %d container(s) in config.", len(cfgContainers))

	agent.containers = &cfgContainers
}

func (agent *Agent) loadCfg() (cfgJson []byte) {

	agent.log.Info("Loading %s", agent.cfgUrl)

	proto, loc := agent.convertUrl(agent.cfgUrl)

	agent.log.Info("proto: %s, loc: %s", proto, loc)

	if proto == "file" {
		return agent.loadFile(loc)
	}

	if proto == "http" {
		return agent.loadUrl(loc)
	}

	return []byte{}
}

func (agent *Agent) loadFile(file string) (cfgJson []byte) {

	b, err := ioutil.ReadFile(file)
	if err != nil {
		agent.log.Fatal(err.Error())
		os.Exit(1)
	}

	return b
}

func (agent *Agent) loadUrl(url string) (cfgJson []byte) {

	res, err := http.Get(url)
	if err != nil {
		agent.log.Fatal(err.Error())
		os.Exit(1)
	}

	defer res.Body.Close()

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		agent.log.Fatal(err.Error())
		os.Exit(1)
	}

	return b
}

func (agent *Agent) convertUrl(url string) (proto, loc string) {
	proto = url[0:4]
	loc = url

	if proto == "file" {
		loc = url[7:]
	}

	return proto, loc
}
