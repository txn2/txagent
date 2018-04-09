# iotagent - Simplified Docker container orchestration.

# Stable / Work in progress.

This code is used in production on a few hundred iot devices. It was developed for with a very specific, yet recurring use case. Professionally I develop software for a number of platforms, including iot. As a hobbyist, I have a dozen or so different devices that I provision with Docker. This project is in the early stages of becoming more general purpose. In the mean time, you may have a similar need. **iotagent** is an uncomplicated way to provision iot devices running your Docker containers. Let me know if this project is useful to you and what features you would like to see.

----

This project is intended to be a very simple docker container for iot devices, that are able to run docker. **iotagent** has been tested on a few arm based devices, but is primarily used for the Raspberry Pi.

Pushing updates and configuration to devices is a complex and complicated problem, having to keep track of each device and it's state. **iotagent** uses a **pull** model for basic provisioning.

The container runs a compiled Go application that can be configured to get a remote or local json configuration file, then uses the configuration to pull, run and monitor containers.



## Supported Devices

Anything that can run Docker. (Windows is untested)

## Getting Started

The agent can be configured with environment variables, command line flags, or a combination on both. Command line argument will override environment variables.

### Environment Variables

| Prupose                    | Environment Variable | Flag   | Default Value |
| -------                    | -------------------- | ----   | ------------- |
| Container configuration.   | AGENT_CFG_URL        | -cfg  | file://conf/defs.json |
| Repository authentication. | AGENT_AUTH_URL       | -auth | file://conf/auth.json |
| Poll frequency.            | AGENT_CFG_POLL       | -poll | 30    |
| Remove existing containers on start. |            | -rm   | false |


## Testing (with source)

Get a list of commands.

```bash
go run ./agent.go --help
```

### Example #1: Run example configurations from source.

```bash
go run ./agent.go
```

## Using as a lib

see GoDocs
https://godoc.org/github.com/cjimti/iotagent/iotagent

### Development

Uses [goreleaser](https://goreleaser.com):

Install goreleaser with brew (mac):
`brew install goreleaser/tap/goreleaser`

Build without releasing:
`goreleaser --skip-publish --rm-dist --skip-validate`


## TODO

- Get & check configuration on interval (ensure state)
- Configuration auth
- Validate configs
- Documentation, Use Case and Examples

## DONE

- Registry authentication
- Command line flag `--rm` to remove running containers
- Added command line flags `--cfg` and `--poll` (see `--help`)
- Run containers
- Pull containers
- Create volumes
- Create networks (if they do not exists)
- Load and marshal configuration json
