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
            # client_id, topic, discovery.device_prefix, and
            # discovery.device_name default to values derived from
            # `inventory_hostname` so the role is safe to apply across
            # multiple hosts without extra overrides.
            discovery:
              enabled: true
              # prefix must match Home Assistant's mqtt.discovery_prefix
              # (default "homeassistant"). Only override if you changed it
              # on the HA side.
              prefix: homeassistant
              # Optional overrides — leave unset to accept the defaults:
              #   device_prefix: "mqttrooper_{{ inventory_hostname }}"
              #   device_name:   "mqttrooper {{ inventory_hostname }}"
          daemon:
            cwd: /var/lib/mqttrooper
            log_file_path: /var/log/mqttrooper/mqttrooper.log
            error_file_path: /var/log/mqttrooper/mqttrooper.error.log
```

See `defaults/main.yml` for the full set of variables and their defaults.

### Home Assistant MQTT autodiscovery

When `mqtt.discovery.enabled: true`, mqttrooper publishes a retained
`button` discovery config per service on startup. Each service becomes a
`button` entity in HA, grouped under a single HA device (one per host).

- All entities share a `device_prefix` and are named
  `<device_prefix>_<service>`; the defaults derive this from
  `inventory_hostname` so multiple hosts produce distinct devices and
  non-colliding `unique_id`s.
- On startup, mqttrooper also clears any stale retained discovery topics
  under its `device_prefix` whose service is no longer in the config — so
  removing a service from `mqttrooper_config.services` makes the
  corresponding HA entity disappear on next deploy/restart.
- `button` entities open the "more info" dialog on tap by default in HA.
  To press on tap from a dashboard card, override `tap_action` on the
  card:

  ```yaml
  tap_action:
    action: perform-action
    perform_action: button.press
    target:
      entity_id: button.mqttrooper_<host>_<service>
  ```

## What it creates

- System user and group `mqttrooper` (nologin shell).
- `/etc/mqttrooper/config.yaml` (`0640 root:mqttrooper`).
- `/var/lib/mqttrooper/` (working dir) and `/var/log/mqttrooper/` (logs).
- `/usr/local/bin/mqttrooper` (`0755 root:root`).
- `/etc/systemd/system/mqttrooper.service`, enabled and started.

Config or binary changes trigger a service restart via handlers.
