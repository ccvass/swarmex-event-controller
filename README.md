# Swarmex Event Controller

Central event dispatcher for the Swarmex ecosystem.

Part of [Swarmex](https://github.com/ccvass/swarmex) — enterprise-grade orchestration for Docker Swarm.

## What It Does

Listens to the Docker event stream in real time and dispatches relevant events to all Swarmex controllers. It acts as the nervous system of the platform, ensuring every service lifecycle change (create, update, health status) is captured and forwarded.

## Labels

This is an infrastructure controller — no service labels required.

## How It Works

1. Connects to the Docker socket and subscribes to the event stream.
2. Filters events for service-related actions (create, update, remove, health_status).
3. Dispatches each event to registered Swarmex controllers via internal channels.
4. Maintains a persistent connection with automatic reconnection on failure.

## Quick Start

```bash
docker service create \
  --name swarmex-event-controller \
  --mount type=bind,src=/var/run/docker.sock,dst=/var/run/docker.sock \
  --constraint node.role==manager \
  ghcr.io/ccvass/swarmex-event-controller:latest
```

## Verified

Real-time create, update, and health events captured and dispatched correctly during live testing.

## License

Apache-2.0
