package iotagent

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/bhoriuchi/go-bunyan/bunyan"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type Agent struct {
	cfgUrl     string
	poll       int
	log        *bunyan.Logger
	cli        *client.Client
	containers []container.Config
}

// NewAgent creates a new agent from a configuration url and a polling interval
func NewAgent(cfgUrl string, poll int) Agent {

	logConfig := bunyan.Config{
		Name:   "iotagent",
		Stream: os.Stdout,
		Level:  bunyan.LogLevelDebug,
	}

	bunyanLogger, err := bunyan.CreateLogger(logConfig)
	if err != nil {
		panic(err)
	}
	bunyanLogger.Info("Loading IoT Agent...")

	// load docker client
	dockerApiVersion := SetEnvIfEmpty("DOCKER_API_VERSION", "1.35")
	bunyanLogger.Info("Loading Docker Client for API version %s.", dockerApiVersion)

	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	agent := Agent{
		cfgUrl: cfgUrl,
		poll:   poll,
		log:    &bunyanLogger,
		cli:    cli,
	}

	cfgJson := agent.loadCfg()
	agent.marshalCfg(cfgJson)

	agent.PullContainers()

	return agent
}

// PullContainers as defined in the configuration file located at
// environment variable AGENT_CFG_URL
func (agent *Agent) PullContainers() error {

	for _, cfgContainer := range agent.containers {
		agent.log.Info("Pull image %s.", cfgContainer.Image)
	}

	// TODO pull container

	return nil
}

func (agent *Agent) marshalCfg(cfgJson []byte) {

	cfgContainers := make([]container.Config, 0)
	err := json.Unmarshal(cfgJson, &cfgContainers)
	if err != nil {
		agent.log.Error(err.Error())
		return
	}

	agent.log.Info("Found %d container(s) in config.", len(cfgContainers))

	agent.containers = cfgContainers
}

func (agent *Agent) loadCfg() (cfgJson []byte) {

	agent.log.Info("Loading %s", agent.cfgUrl)

	proto, loc := agent.convertUrl(agent.cfgUrl)

	agent.log.Info("Got protocol: %s, at location: %s", proto, loc)

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
