# Distributed LLM Gateway Platform

A distributed AI inference platform supporting async request scheduling, streaming response, dynamic batching, autoscaling, and high-throughput inference orchestration.

## Architecture

```
Client (Browser)
    ↓
React Frontend
    ↓
Go API Gateway  (Auth, Rate Limiting, Routing)
    ↓
Redis Stream    (Async Queue)
    ↓
Python Worker   (LLM Inference)
    ↓
Streaming Response → Client
```

## Stack

- **Gateway**: Go + Gin
- **Queue**: Redis Streams
- **Worker**: Python + FastAPI
- **Frontend**: React + Tailwind CSS
- **Infra**: Docker, Kubernetes
- **Monitoring**: Prometheus + Grafana

## Getting Started

```bash
docker-compose up
```

## Progress

- [ ] Day 1: Project structure
- [ ] Day 2: Go Gateway + Redis Stream
- [ ] Day 3: Python Worker
- [ ] Day 4: Result return (Pub/Sub)
- [ ] Day 5: React Chat UI
- [ ] Day 6: End-to-end integration
- [ ] Day 7: Phase 1 complete
