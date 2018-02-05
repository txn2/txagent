# WIP: IoT Docker Agent

This project is intended to be a very simple docker container for IoT devices (that are able to run docker.) In developing application for IoT devices it is not always possible to push updates and configuration. The project uses a pull on interval model for basic docker container organization.

The container runs a compiled Go application that can be configured to get a remote or local json configuration file, then uses the configuration to pull, run and update containers on a device.

# Supported Devices

Intended for Raspberry Pi. Will be testing on other ARM boards as project progresses.

# Work in progress. Do not use.

This is very early development as I refactor a number of working concepts into this new model. I am open to help and suggestions.

# TODO

- Pull containers
- Run containers
- Check configuration on interval
- Update containers on interval (like watchtower)
- Remote logging

# DONE

- Load and marshal configuration json
