package iotagent

import (
	"bufio"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/bhoriuchi/go-bunyan/bunyan"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// DockerStatus messages
type DockerStatus struct {
	Status string `json:"status"`
}

type AgentContainerCfg struct {
	Containers []container.Config `json:"containers"`
}

// Agent is the main agent object for pulling and running containers.
type Agent struct {
	CfgUrl string
	Poll   int
	Log    *bunyan.Logger
	Cli    *client.Client
	Cfg    *AgentContainerCfg
}

// NewAgent creates a new agent from a configuration url and a polling interval
func NewAgent(cfgUrl string, poll int) (agent Agent, err error) {

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
		return Agent{}, err
	}

	agent = Agent{
		CfgUrl: cfgUrl,
		Poll:   poll,
		Log:    &bunyanLogger,
		Cli:    cli,
	}

	cfgJson := agent.loadCfg()

	agent.marshalCfg(cfgJson)
	if err != nil {
		return Agent{}, err
	}

	return agent, nil
}

// PullContainers as defined in the configuration file located at
// environment variable AGENT_CFG_URL
func (agent *Agent) PullContainers() error {

	ctx := context.Background()
	opts := types.ImagePullOptions{All: false}

	for _, cfgContainer := range agent.Cfg.Containers {
		agent.Log.Info("Pull image %s.", cfgContainer.Image)

		// pull container
		responseBody, err := agent.Cli.ImagePull(ctx, cfgContainer.Image, opts)
		if err != nil {
			return err
		}

		scanner := bufio.NewScanner(responseBody)
		for scanner.Scan() {

			dockerStatus := &DockerStatus{}
			err := json.Unmarshal([]byte(scanner.Text()), dockerStatus)
			if err != nil {
				return err
			}

			agent.Log.Info("%s image pull status: %s", cfgContainer.Image, dockerStatus.Status)
		}

		responseBody.Close()
	}

	return nil
}

func (agent *Agent) marshalCfg(cfgJson []byte) error {

	// make a new agent configuration object
	agent.Cfg = &AgentContainerCfg{}

	err := json.Unmarshal(cfgJson, agent.Cfg)
	if err != nil {
		agent.Log.Error(err.Error())
		return err
	}

	agent.Log.Info("Found %d container(s) in config.", len(agent.Cfg.Containers))

	return nil
}

func (agent *Agent) loadCfg() (cfgJson []byte) {

	agent.Log.Info("Loading %s", agent.CfgUrl)

	proto, loc := agent.convertUrl(agent.CfgUrl)

	agent.Log.Info("Reading protocol: %s, at location: %s", proto, loc)

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
		agent.Log.Fatal(err.Error())
		os.Exit(1)
	}

	return b
}

func (agent *Agent) loadUrl(url string) (cfgJson []byte) {

	res, err := http.Get(url)
	if err != nil {
		agent.Log.Fatal(err.Error())
		os.Exit(1)
	}

	defer res.Body.Close()

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		agent.Log.Fatal(err.Error())
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
