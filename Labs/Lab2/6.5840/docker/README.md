# Dockerized MapReduce

Two container images are provided:

| Image | Dockerfile | Purpose |
|-------|------------|---------|
| `coordinator` | `docker/Dockerfile.coordinator` | Runs `mrcoordinator` |
| `worker` | `docker/Dockerfile.worker` | Runs `mrworker` |

## Building

```bash
cd Labs/Lab2/6.5840
docker build -f docker/Dockerfile.coordinator -t mr-coordinator .
docker build -f docker/Dockerfile.worker -t mr-worker .
```

## Running locally

1. Start the coordinator. It already contains the `pg-*.txt` inputs and automatically
   runs `mrcoordinator /data/pg-*.txt` unless you pass explicit arguments.

```bash
docker run --rm \
  -e MR_COORDINATOR_ADDRESS=0.0.0.0:7777 \
  -p 7777:7777 \
  mr-coordinator
```

2. Each worker image ships with the freshly built `wc.so` plugin in `/usr/local/lib/wc.so`.
   Launch workers by pointing them at the coordinator; pass `/usr/local/lib/wc.so`
   explicitly if you override the entrypoint or use `--entrypoint /bin/sh`.

```bash
docker run --rm \
  -e MR_COORDINATOR_ADDRESS=<public-ip>:7777 \
  mr-worker
```

Replace `<public-ip>` with the reachable coordinator address (e.g., `host.docker.internal`
on Docker Desktop). Use `MR_WORKER_LISTEN_ADDRESS` (bind inside the container) and
`MR_WORKER_PUBLIC_ADDRESS` (host/IP that other workers should dial) when you run multiple workers:

```bash
docker run --rm \
  -e MR_COORDINATOR_ADDRESS=<public-ip>:7777 \
  -e MR_WORKER_LISTEN_ADDRESS=0.0.0.0:9000 \
  -e MR_WORKER_PUBLIC_ADDRESS=<host-ip>:9000 \
  -p 9000:9000 \
  mr-worker
```

