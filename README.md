# WIP: IoT Docker Agent

# Work in progress. Do not use.

This is very early development as I refactor a number of working concepts into this new model. I am open to help and suggestions.

----

This project is intended to be a very simple docker container for IoT devices (that are able to run docker.) In developing application for IoT devices it is not always possible to push updates and configuration. The project uses a pull on interval model for basic docker container organization.

The container runs a compiled Go application that can be configured to get a remote or local json configuration file, then uses the configuration to pull, run and update containers on a device.

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

- Check configuration on interval (ensure state)
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
