<div align="center">

# MQTTrooper

[![Go Version](https://img.shields.io/badge/go-1.23.1-blue.svg)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/jjtorroglosa/MQTTrooper)](https://goreportcard.com/report/github.com/jjtorroglosa/MQTTrooper)
[![Last Commit](https://img.shields.io/github/last-commit/jjtorroglosa/MQTTrooper.svg)](https://github.com/jjtorroglosa/MQTTrooper/commits/main)

MQTTrooper is a lightweight, flexible, and easy-to-use daemon written in go that listens for
commands via MQTT or HTTP and executes them on the host machine. It's designed to be a bridge
between your IoT devices, home automation system, or any other service that can send MQTT messages
or HTTP requests, and the scripts or commands you want to run on your server.

</div>

## Table of contents
<!-- mtoc-start -->

* [Features](#features)
* [Getting Started](#getting-started)
  * [Requirements](#requirements)
  * [Installation](#installation)
  * [Configuration](#configuration)
    * [`executor`](#executor)
    * [`services`](#services)
    * [`http`](#http)
    * [`mqtt`](#mqtt)
* [Usage](#usage)
  * [Running as a service](#running-as-a-service)
    * [Linux (systemd)](#linux-systemd)
    * [macOS (launchd)](#macos-launchd)
  * [Command-line interface](#command-line-interface)
  * [HTTP API](#http-api)
  * [MQTT API](#mqtt-api)
* [Security](#security)
  * [HTTP Endpoint](#http-endpoint)
  * [Command Execution and Security](#command-execution-and-security)
  * [Principle of Least Privilege](#principle-of-least-privilege)
  * [Shell Configuration](#shell-configuration)
    * [Example: Using `rbash`](#example-using-rbash)
* [Development](#development)
* [Building](#building)
* [Tests](#tests)
* [License](#license)

<!-- mtoc-end -->

## Features

- **Multiple Interfaces**: Execute commands via MQTT messages or HTTP requests.
- **Flexible Configuration**: Configure services and settings using a simple YAML file.
- **Service Management**: Run as a systemd service on Linux or a launchd service on macOS.
- **Dry Run Mode**: Test your configuration without executing any commands.
- **Easy to Deploy**: Single binary with minimal dependencies.

## Getting Started

### Requirements

- [Go](https://golang.org/) (1.23 or later)
- [Mosquitto](https://mosquitto.org/) or any other MQTT broker.

### Installation

1.  Clone the repository:
    ```bash
    git clone https://github.com/jjtorroglosa/MQTTrooper.git
    cd mqttrooper
    ```
2.  Build the project:
    ```bash
    make setup
    make all
    ```
3.  Copy the example configuration file:
    ```bash
    cp examples/config.yaml.example config.yaml
    ```
4.  Edit `config.yaml` to match your setup.

### Configuration

MQTTrooper is configured using a `config.yaml` file. Here's an overview of the configuration options:

#### `executor`

| Key      | Type    | Description                                     | Default         |
| -------- | ------- | ----------------------------------------------- | --------------- |
| `shell`  | `string`| The shell to use for executing commands.        | `/usr/bin/env bash` |
| `dry_run`| `boolean`| If `true`, commands will be logged but not executed. | `false`         |

#### `services`

The `services` section is a map of service names to the commands they execute.

```yaml
services:
  date: "date"
  volume_up: "amixer -D pulse sset Master 5%+"
  volume_down: "amixer -D pulse sset Master 5%-"
```

#### `http`

| Key              | Type    | Description                                       | Default     |
| ---------------- | ------- | ------------------------------------------------- | ----------- |
| `enabled`        | `boolean`| Enable or disable the HTTP interface.             | `false`     |
| `port`           | `integer`| The port for the HTTP server to listen on.        | `8080`      |
| `bind_address`   | `string` | The address for the HTTP server to bind to.       | `127.0.0.1` |
| `allowed_address`| `string` | The IP address allowed to make requests.          | `127.0.0.1` |

#### `mqtt`

| Key              | Type    | Description                                       | Default     |
| ---------------- | ------- | ------------------------------------------------- | ----------- |
| `enabled`        | `boolean`| Enable or disable the MQTT interface.             | `false`     |
| `address`        | `string` | The address of the MQTT broker (e.g., `tcp://127.0.0.1:1883`). |             |
| `client_id`      | `string` | The client ID to use when connecting to the broker. |             |
| `user`           | `string` | The username for MQTT authentication.             |             |
| `pass`           | `string` | The password for MQTT authentication.             |             |
| `topic`          | `string` | The MQTT topic to subscribe to for commands.      |             |
| `connection_timeout_seconds` | `integer` | The connection timeout in seconds. | `5` |

## Usage

### Running as a service

MQTTrooper can be run as a systemd service on Linux or a launchd service on macOS.

#### Linux (systemd)

1.  Generate the systemd service file:
    ```bash
    ./mqttrooper dump-systemd-service > mqttrooper.service
    ```
2.  Copy or link the service file to the systemd user directory:
    ```bash
    sudo ln -s $(pwd)/mqttrooper.service /etc/systemd/user/mqttrooper.service
    ```
3.  Enable and start the service:
    ```bash
    systemctl --user enable mqttrooper
    systemctl --user start mqttrooper
    ```

#### macOS (launchd)

1.  Generate the launchd plist file:
    ```bash
    ./mqttrooper dump-plist > com.user.mqttrooper.plist
    ```
2.  Copy the plist file to the LaunchAgents directory:
    ```bash
    ln -s $(pwd)/com.user.mqttrooper.plist ~/Library/LaunchAgents/
    ```
3.  Load the service:
    ```bash
    launchctl load ~/Library/LaunchAgents/com.user.mqttrooper.plist
    ```

### Command-line interface

```
Usage: mqttrooper [command]

Commands:
  serve                  Start the MQTTrooper service (default)
  dump-plist             Dump the launchd plist file for macOS
  dump-systemd-service   Dump the systemd service file for Linux
```

**Options**:

| Flag        | Description                               | Default       |
| ----------- | ----------------------------------------- | ------------- |
| `-c`        | Path to the `config.yaml` file.           | `config.yaml` |
| `-d`        | Enable dry run mode.                      | `false`       |
| `-user`     | MQTT user.                                |               |
| `-password` | MQTT password.                            |               |
| `-p`        | Port to listen for HTTP requests.         | `8080`        |
| `-b`        | Address to bind HTTP server to.           | `127.0.0.1`   |
| `-allow`    | Address to allow HTTP requests from.      | `127.0.0.1`   |

### HTTP API

> [!WARNING]
> This application doesn't implement any security for the http layer. Take a look to the security section

When the HTTP interface is enabled, you can execute commands by making GET requests to the `/r` endpoint.

- **URL**: `http://<bind_address>:<port>/r`
- **Method**: `GET`
- **Query Parameters**:
    - `s`: The name of the service to execute.

**Example:**

```bash
curl "http://127.0.0.1:8080/r?s=date"
```

This will execute the `date` command defined in your `config.yaml`.

You can also access the home page at `http://<bind_address>:<port>/` to see a list of available services.

### MQTT API

When the MQTT interface is enabled, MQTTrooper will subscribe to the specified topic and execute the received message as a service name.

- **Topic**: The topic specified in `mqtt.topic`.
- **Message**: The name of the service to execute.

**Example:**

Using `mosquitto_pub`:

```bash
mosquitto_pub -h 127.0.0.1 -t "/mqttrooper/commands" -m "date"
```

This will execute the `date` command defined in your `config.yaml`.

## Security

### HTTP Endpoint

The HTTP endpoint is not secure and should not be exposed to the internet. It does not provide any
authentication or encryption. It is recommended to use a reverse proxy with authentication and
SSL/TLS encryption if you need to expose it to the internet.

The only security measure is the `allowed_address` option, which restricts access to the specified IP address.

### Command Execution and Security

By design, MQTTrooper executes the command strings defined in your `config.yaml` file using the
shell specified in the `executor.shell` option. This allows for flexible and powerful commands.

User input (e.g., a service name from an MQTT message) is used only as a safe lookup key to find the
corresponding command in your configuration. Arbitrary input will not be executed.

The primary security consideration is that **any user with write access to your `config.yaml` file
can define any command to be executed by the MQTTrooper service.** Therefore, it is crucial to
restrict file permissions for `config.yaml` and to follow the Principle of Least Privilege, as detailed below.

### Principle of Least Privilege

It is recommended to run the MQTTrooper service with a dedicated user with the minimum required
privileges. Running the service as a privileged user (e.g., `root`) could allow a malicious user to
gain control over the entire system.

### Shell Configuration

The shell used to execute the commands should be as restrictive as possible. For example, you could
use a restricted shell that only allows the execution of specific commands.

#### Example: Using `rbash`

`rbash` is a restricted version of the `bash` shell. When a user's shell is set to `rbash`, they can
only execute commands that are in their `PATH` and cannot change their `PATH`.

To use `rbash` with MQTTrooper, you can do the following:

1.  **Create a dedicated user for MQTTrooper:**

    ```bash
    sudo useradd -m -s /bin/rbash mqttrooper
    ```

2.  **Create a directory for the allowed commands:**

    ```bash
    sudo mkdir -p /home/mqttrooper/bin
    ```

3.  **Set the `PATH` for the `mqttrooper` user:**

    Edit `/home/mqttrooper/.bash_profile` and add the following line:

    ```bash
    export PATH=$HOME/bin
    ```

4.  **Create symbolic links to the allowed commands:**

    For each command you want to allow, create a symbolic link in the `/home/mqttrooper/bin` directory.

    ```bash
    sudo ln -s /bin/date /home/mqttrooper/bin/date
    sudo ln -s /usr/bin/amixer /home/mqttrooper/bin/amixer
    ```

5.  **Configure MQTTrooper to run as the `mqttrooper` user:**

    When running MQTTrooper as a service, you need to specify the `mqttrooper` user.

    -   **systemd**: Edit the `mqttrooper.service` file and add `User=mqttrooper` under the `[Service]` section.
    -   **launchd**: Edit the `com.user.mqttrooper.plist` file and add the following keys:

        ```xml
        <key>UserName</key>
        <string>mqttrooper</string>
        ```

6.  **Update your `config.yaml`:**

    Make sure the commands in your `config.yaml` file do not contain any path information.

    ```yaml
    services:
      date: "date"
      volume_up: "amixer -D pulse sset Master 5%+"
      volume_down: "amixer -D pulse sset Master 5%-"
    ```

## Development

A `compose.yaml` file is provided for development. It sets up a Go container with the source code
mounted and a Mosquitto container.

To start the development environment:

```bash
docker-compose up -d
```

To get a shell inside the Go container:

```bash
docker-compose exec dev bash
```

## Building

To build the project, run:

```bash
make build
make all
```

This will create the `linux` and `darwin` (mac) binaries for `amd64` and `arm64` architectures:
`dist/mqttrooper.$os.$arch`

## Tests

```bash
make test
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
