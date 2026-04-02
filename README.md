<div align="center">
  <h1>Overwatch: Scalable Threat Intelligence API</h1>
  <p><strong>Advanced Training in Go (Golang)</strong></p>
  
  [![Go Report Card](https://goreportcard.com/badge/github.com/Sou-Daroh/http-server)](https://goreportcard.com/report/github.com/Sou-Daroh/http-server)
  [![Continuous Integration](https://github.com/Sou-Daroh/http-server/actions/workflows/ci.yml/badge.svg)](https://github.com/Sou-Daroh/http-server/actions/workflows/ci.yml)
</div>

## Project Overview
Overwatch is a robust, concurrent backend API system configured in Go. It operates as a Threat Intelligence Honeypot designed to intercept, log, and broadcast malicious HTTP cyber-attacks in real time. 

The architecture strictly adheres to the core learning objectives of the **Scalable Backend API Development** module, leveraging Go's static typing, native concurrency primitives, and structured module design to deliver a production-ready service.

---

## Enterprise Architecture Features

- **Gin Framework Routing**: Replaced native routing with the `gin-gonic/gin` web infrastructure to perform isolated Context trapping and middleware chaining seamlessly.
- **Stateless JWT Authorization**: Cryptographically locks the core Administrative dashboard and REST endpoints utilizing standard Bearer tokens.
- **WebSocket Pub/Sub Hub**: Migrated from legacy client polling to implement a pure Go Channel concurrency structure `(chan []byte)`. This architectural shift instantly broadcasts threats to all connected user contexts within a <100ms latency period.
- **Web Application Firewall (The Ban Hammer)**: Engineered a native Go `sync.Map` interceptor. It caches attacker IPs in memory and automatically terminates connections yielding a `403 Forbidden` response after exactly 3 strikes, bypassing heavy Database IO reads completely block repetitive malicious behavior.
- **Containerization Orchestration**: The entire deployment architecture is isolated inside Docker Compose deploying lightweight Alpine Linux images interconnected over bridged virtual subnets.
- **Continuous Integration (CI/CD)**: GitHub Actions workflows test core throughput Benchmarks locally, downloading modern dependencies, and performing Docker environment dry-runs upon every single upstream repository execution.

## The Vue.js Dashboard
The remote front-end renders a beautifully minimal visualization of the trapped threat intelligence feed. 
It features:
- A responsive, animated `Leaflet.js` map correlating payload inputs to global GeoIP coordinate endpoints.
- A seamless **WebSocket auto-reconnection loop** guaranteeing persistent data streaming across transient network states.
- A **Simulate Attack** execution method utilizing randomized global mathematics to safely evaluate the WebSocket architectural stream without triggering the Administrator-level WAF constraints.

## Deployment Instructions

Ensure the built-in Docker Engine is active. 

```bash
# Clone the enterprise architecture 
git clone https://github.com/Sou-Daroh/http-server.git
cd http-server

# Bootstrap and provision the isolated environment
docker-compose up -d --build
```
Navigate your browser to [http://localhost:8080](http://localhost:8080) and authenticate using the Bcrypt-seeded default credentials:

- **Username:** `admin`
- **Password:** `admin123`

---
> Detailed system mechanics, technical stack outlines, and testing directives are formally mapped within the `/docs` registry.
