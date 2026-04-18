# mqttrooper Ansible role

Installs a prebuilt `mqttrooper` binary on Debian/Ubuntu, renders
`/etc/mqttrooper/config.yaml`, and manages a hardened systemd unit.

## Variables

By default the role downloads the binary matching the target host's
`ansible_system` / `ansible_architecture` from the GitHub release named by
`mqttrooper_version` (e.g. `v1.2.3`). Supported: linux/darwin on amd64/arm64.

- `mqttrooper_version` — release tag to pull (default `v0.1.0`).
- `mqttrooper_github_repo` — defaults to `jjtorroglosa/mqttrooper`.
- `mqttrooper_binary_src` — override to pin a specific URL or use a local
  path on the control node (e.g. `dist/mqttrooper.linux.amd64`).
- `mqttrooper_binary_checksum` — recommended for URL installs,
  e.g. `sha256:abcd...` (pull from the release's `SHA256SUMS`).

## Typical usage

```yaml
- hosts: mqttrooper_hosts
  become: true
  roles:
    - role: mqttrooper
      vars:
        mqttrooper_binary_src: ../../../dist/mqttrooper.linux.amd64
        mqttrooper_config:
          executor:
            shell: /usr/bin/env bash
          services:
            snapserver: "sudo systemctl restart snapserver.service"
          mqtt:
            enabled: true
            address: tcp://broker.lan:1883
            user: mqttuser
            pass: mqttpass
            topic: /mqttrooper/myhost
            discovery:
              enabled: true
              device_prefix: mqttrooper_myhost
          daemon:
            cwd: /var/lib/mqttrooper
            log_file_path: /var/log/mqttrooper/mqttrooper.log
            error_file_path: /var/log/mqttrooper/mqttrooper.error.log
```

See `defaults/main.yml` for the full set of variables and their defaults.

## What it creates

- System user and group `mqttrooper` (nologin shell).
- `/etc/mqttrooper/config.yaml` (`0640 root:mqttrooper`).
- `/var/lib/mqttrooper/` (working dir) and `/var/log/mqttrooper/` (logs).
- `/usr/local/bin/mqttrooper` (`0755 root:root`).
- `/etc/systemd/system/mqttrooper.service`, enabled and started.

Config or binary changes trigger a service restart via handlers.
