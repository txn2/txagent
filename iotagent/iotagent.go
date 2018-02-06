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
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

// DockerStatus messages
type DockerStatus struct {
	Status string
}

// AgentContainerCfg each container in the json configuration file
type AgentContainerCfg struct {
	Config           container.Config
	HostConfig       container.HostConfig
	NetworkingConfig network.NetworkingConfig
}

// AgentCfg represents the entire json configuration file
type AgentCfg struct {
	Volumes    []volume.VolumesCreateBody
	Networks   map[string]types.NetworkCreate
	Containers map[string]AgentContainerCfg
}

// Agent is the main agent object for pulling and running containers.
type Agent struct {
	CfgUrl string
	Poll   int
	Log    *bunyan.Logger

	// Cli is the Docker client
	// see https://godoc.org/github.com/moby/moby/client
	Cli *client.Client

	// Cfg holds a AgentCfg marshaled from the external json
	Cfg *AgentCfg
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

// CreateVolumes creates docker volumes defined in the json configuration.
func (agent *Agent) CreateVolumes() error {
	ctx := context.Background()

	for _, cfgVolume := range agent.Cfg.Volumes {
		_, err := agent.Cli.VolumeCreate(ctx, cfgVolume)
		if err != nil {
			agent.Log.Warn("Volume Create returned %s", err.Error())
			return err
		}

		agent.Log.Info("Volume %s created.", cfgVolume.Name)
	}

	return nil
}

// CreateNetworks create networks defined in the config. Will not create a
// network if it already exists.
func (agent *Agent) CreateNetworks() error {
	ctx := context.Background()

	nets, err := agent.Cli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		agent.Log.Warn("Network List returned %s", err.Error())
		return err
	}

	for name, cfgNetwork := range agent.Cfg.Networks {
		// look though list of network to see if this one already exists
		for _, netRes := range nets {
			if netRes.Name == name {
				agent.Log.Warn("Network Create: Noting to do, %s already exists.", name)
				return nil
			}
		}

		agent.Log.Info("Got Network: %s, type: %s", name, cfgNetwork.Driver)
		resp, err := agent.Cli.NetworkCreate(ctx, name, cfgNetwork)
		if err != nil {
			agent.Log.Warn("Network Create returned %s", err.Error())
			return err
		}

		agent.Log.Info("Network Create returned %s: %s", resp.ID, resp.Warning)
	}
	return nil
}

// PullContainers as defined in the configuration file located at
// environment variable AGENT_CFG_URL
func (agent *Agent) PullContainers() error {

	ctx := context.Background()
	opts := types.ImagePullOptions{All: false}

	for name, cfgContainer := range agent.Cfg.Containers {
		agent.Log.Info("Pull image %s for %s.", cfgContainer.Config.Image, name)

		// pull container
		responseBody, err := agent.Cli.ImagePull(ctx, cfgContainer.Config.Image, opts)
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

			agent.Log.Info("%s image pull status: %s", cfgContainer.Config.Image, dockerStatus.Status)
		}

		responseBody.Close()
	}

	return nil
}

func (agent *Agent) CreateContainers() error {

	ctx := context.Background()

	for name, cfgContainer := range agent.Cfg.Containers {
		agent.Log.Info("Creating container %s from %s image.", name, cfgContainer.Config.Image)

		cb, err := agent.Cli.ContainerCreate(ctx, &cfgContainer.Config, &cfgContainer.HostConfig, &cfgContainer.NetworkingConfig, name)
		if err != nil {
			agent.Log.Warn("Create container for %s received %s", name, err.Error())
			return err
		}

		agent.Log.Info("Create container for %s received %s with warnings %s", name, cb.ID, cb.Warnings)

		agent.Log.Info("Starting container %s", name)

		// TODO START CONTAINER

	}

	return nil
}

func (agent *Agent) marshalCfg(cfgJson []byte) error {

	// make a new agent configuration object
	agent.Cfg = &AgentCfg{}

	err := json.Unmarshal(cfgJson, agent.Cfg)
	if err != nil {
		agent.Log.Error(err.Error())
		return err
	}

	agent.Log.Info("Found %d volumes(s) in config.", len(agent.Cfg.Volumes))
	agent.Log.Info("Found %d network(s) in config.", len(agent.Cfg.Networks))
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
