package iotagent

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

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

// AgentCfg represents the entire json configuration file
type AgentAuth struct {
	Volumes    []volume.VolumesCreateBody
	Networks   map[string]types.NetworkCreate
	Containers map[string]AgentContainerCfg
}

// iotAgent is the main iotAgent object for pulling and running containers.
type iotAgent struct {
	CfgUrl  string
	AuthUrl string
	Poll    time.Duration
	Log     *bunyan.Logger

	// Cli is the Docker client
	// see https://godoc.org/github.com/moby/moby/client
	Cli *client.Client

	// Cfg holds a AgentCfg marshaled from the external json
	Cfg *AgentCfg

	// Cfg holds a map of AuthConfig by server (as key)
	Auth map[string]types.AuthConfig

	// hold runtime options
	opts AgentOptions
}

type AgentOptions struct {
	LogOut io.Writer
	LogName string
}

// NewAgent creates a new iotAgent from a configuration url and a polling interval
func NewAgent(cfgUrl string, authUrl string, poll int, opts AgentOptions) (agent iotAgent, err error) {

	// Defaults
	if opts.LogOut == nil {
		opts.LogOut = os.Stdout
	}

	if opts.LogName == "" {
		opts.LogName = "iotagent"
	}

	logConfig := bunyan.Config{
		Name:   opts.LogName,
		Stream: opts.LogOut,
		Level:  bunyan.LogLevelDebug,
	}

	bunyanLogger, err := bunyan.CreateLogger(logConfig)
	if err != nil {
		panic(err)
	}
	bunyanLogger.Info("Loading IoT iotAgent...")

	// load docker client
	dockerApiVersion := SetEnvIfEmpty("DOCKER_API_VERSION", "1.35")
	bunyanLogger.Info("Loading Docker Client for API version %s.", dockerApiVersion)

	// get a Docker client
	cli, err := client.NewEnvClient()
	if err != nil {
		return iotAgent{}, err
	}

	// configure the agent
	a := iotAgent{
		CfgUrl:  cfgUrl,
		AuthUrl: authUrl,
		Poll:    time.Duration(poll) * time.Second,
		Log:     &bunyanLogger,
		Cli:     cli,
	}

	// load the configuration JSON
	// TODO: validate JSON
	// TODO: accept yaml?
	cfgJson := a.loadCfg()
	a.marshalCfg(cfgJson)
	if err != nil {
		return iotAgent{}, err
	}

	authJson := a.loadAuth()

	a.marshalAuth(authJson)
	if err != nil {
		return iotAgent{}, err
	}

	return a, nil
}

// Authenticate a
func (agent *iotAgent) Authenticate() error {

	return nil
}

// CreateVolumes creates docker volumes defined in the json configuration.
func (agent *iotAgent) CreateVolumes() error {
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
func (agent *iotAgent) CreateNetworks() error {
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

// PollContainers list the status of containers on interval
func (agent *iotAgent) PollContainers() error {
	err := agent.ContainerState()
	if err != nil {
		agent.Log.Error("Poll Containers received %s", err.Error())
		return err
	}

	for range time.NewTicker(agent.Poll).C {
		agent.PollContainers()
	}

	return nil
}

func (agent *iotAgent) ContainerState() error {
	ctx := context.Background()
	listOps := types.ContainerListOptions{All: true}

	// get a list of existing containers
	existingContainers, err := agent.Cli.ContainerList(ctx, listOps)
	if err != nil {
		agent.Log.Error("Container state received %s", err.Error())
		return err
	}

	for _, existingContainer := range existingContainers {
		for name, _ := range agent.Cfg.Containers {
			if existingContainer.Names[0][1:] == name {
				agent.Log.Info("Container State found container %s in state %s.", name, strings.ToUpper(existingContainer.State))
			}
		}
	}

	return nil
}

// PullContainers as defined in the configuration file located at
// environment variable AGENT_CFG_URL
func (agent *iotAgent) PullContainers() error {

	ctx := context.Background()

	opts := types.ImagePullOptions{All: false}

	for name, cfgContainer := range agent.Cfg.Containers {
		agent.Log.Info("Pull image %s for %s.", cfgContainer.Config.Image, name)

		// if we have authentication for this server then add it to opts
		s := strings.Split(cfgContainer.Config.Image, "/")
		server := s[0]
		auth := agent.Auth[server]

		if auth.Username != "" {
			b, _ := json.Marshal(auth)
			opts.RegistryAuth = base64.StdEncoding.EncodeToString(b)

			agent.Log.Info("Found authentication for %s", server)
			agent.Log.Info("Creds: %s", opts.RegistryAuth)
		}

		// pull container
		responseBody, err := agent.Cli.ImagePull(ctx, cfgContainer.Config.Image, opts)
		if err != nil {
			agent.Log.Error("Pull imaged received: %s", err.Error())
			// TODO: suppress error flag? (retry in the future?)
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

// StopRemoveContainers defined in configuration json
func (agent *iotAgent) StopRemoveContainers() error {

	ctx := context.Background()

	listOps := types.ContainerListOptions{All: true}
	rmOpts := types.ContainerRemoveOptions{
		Force: true,
	}

	// get a list of existing containers, no need to stop a container
	// if is does not exist
	existingContainers, err := agent.Cli.ContainerList(ctx, listOps)
	if err != nil {
		agent.Log.Error("Container stop and remove received %s", err.Error())
		return err
	}

	// loop and stop/remove containers
	for _, existingContainer := range existingContainers {

		for name := range agent.Cfg.Containers {
			// is this one of ours?
			if existingContainer.Names[0][1:] == name {
				agent.Log.Info("Found %s in state %s.", name, existingContainer.State)

				var timeout time.Duration = 30000
				if existingContainer.State == "running" {
					err = agent.Cli.ContainerStop(ctx, existingContainer.ID, &timeout)
					if err != nil {
						agent.Log.Error("Container stop remove for %s with id %s received %s", name, existingContainer.ID, err.Error())
						continue
					}
					agent.Log.Info("Stopped container %s", name)
				}

				err = agent.Cli.ContainerRemove(ctx, existingContainer.ID, rmOpts)
				if err != nil {
					agent.Log.Error("Container stop remove for %s with id %s received %s", name, existingContainer.ID, err.Error())
					continue
				}
				agent.Log.Info("Removed container %s", name)

			}
		}

	}

	return nil
}

// CreateContainers defined in configuration json
func (agent *iotAgent) CreateContainers() error {

	ctx := context.Background()

	listOps := types.ContainerListOptions{All: true}

	existingContainers, err := agent.Cli.ContainerList(ctx, listOps)
	if err != nil {
		agent.Log.Error("Container list received %s", err.Error())
		return err
	}

	var containerNames []string

	// log out found containers and their state
	for _, existingContainer := range existingContainers {
		agent.Log.Info("Found %s container with names %s", existingContainer.State, existingContainer.Names)
		containerNames = append(containerNames, existingContainer.Names...)
	}

	for name, cfgContainer := range agent.Cfg.Containers {

		skip := false

		// check for the existing of the same container name
		for _, existingContainerName := range containerNames {
			if existingContainerName[1:] == name {
				agent.Log.Warn("Create container found container named %s, nothing to do.", existingContainerName[1:])

				skip = true
				break
			}
		}

		if skip {
			continue
		}

		agent.Log.Info("Creating container %s from %s image.", name, cfgContainer.Config.Image)

		// creating container
		cb, err := agent.Cli.ContainerCreate(ctx, &cfgContainer.Config, &cfgContainer.HostConfig, &cfgContainer.NetworkingConfig, name)
		if err != nil {
			agent.Log.Warn("Create container for %s received %s", name, err.Error())
			return err
		}

		agent.Log.Info("Create container for %s received %s with warnings %s", name, cb.ID, cb.Warnings)

		agent.Log.Info("Starting container %s", name)

		// starting container
		err = agent.Cli.ContainerStart(ctx, cb.ID, types.ContainerStartOptions{})
		if err != nil {
			agent.Log.Warn("Container start received %s", err.Error())
			return err
		}

	}

	return nil
}

func (agent *iotAgent) marshalAuth(authJson []byte) error {

	err := json.Unmarshal(authJson, &agent.Auth)
	if err != nil {
		agent.Log.Error(err.Error())
		return err
	}

	agent.Log.Info("Found %d auth configs.", len(agent.Auth))

	return nil
}

func (agent *iotAgent) marshalCfg(cfgJson []byte) error {

	// make a new iotAgent configuration object
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

func (agent *iotAgent) loadAuth() (authJson []byte) {

	agent.Log.Info("Loading %s", agent.AuthUrl)

	proto, loc := agent.convertUrl(agent.AuthUrl)

	agent.Log.Info("Reading protocol: %s, at location: %s", proto, loc)

	if proto == "file" {
		return agent.loadFile(loc)
	}

	if proto == "http" {
		return agent.loadUrl(loc)
	}

	return []byte{}
}

func (agent *iotAgent) loadCfg() (cfgJson []byte) {

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

func (agent *iotAgent) loadFile(file string) (cfgJson []byte) {

	b, err := ioutil.ReadFile(file)
	if err != nil {
		agent.Log.Fatal(err.Error())
		os.Exit(1)
	}

	return b
}

func (agent *iotAgent) loadUrl(url string) (cfgJson []byte) {

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

func (agent *iotAgent) convertUrl(url string) (proto, loc string) {
	proto = url[0:4]
	loc = url

	if proto == "file" {
		loc = url[7:]
	}

	return proto, loc
}
