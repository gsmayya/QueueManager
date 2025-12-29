# NodeQueue Service

A Go REST service for managing a queue system with nodes and resources.

## Overview

This service provides a queue management system where:
- **Nodes** represent entities that need to be serviced
- **Resources** are capacity-limited abstractions where nodes get serviced
- Each node takes one capacity unit of a resource
- Nodes can be moved between resources until they are completed
- Completed nodes cannot be moved to any resource

## Architecture

### Core Components

- **Entity**: Represents an entity with a name (currently just a name field)
- **Node**: Contains a reference to an Entity and tracks which Resource it's currently in
- **Resource**: A capacity-limited container that can hold multiple nodes
- **QueueService**: Manages nodes and resources, handles business logic

## API Endpoints

### Create Node
```
POST /nodes
Content-Type: application/json

{
  "entity_name": "my-entity"
}
```

### List All Nodes
```
GET /nodes
```

### Get Node by ID
```
GET /nodes/{id}
```

### Move Node to Another Resource
```
POST /nodes/{id}/move
Content-Type: application/json

{
  "target_resource_id": "resource-1"
}
```

### Complete Node
```
POST /nodes/{id}/complete
```

### List All Resources
```
GET /resources
```

## Running the Service

1. Install dependencies:
```bash
go mod download
```

2. Run the service:
```bash
go run .
```

The service will start on port 8080 by default. You can change the port by setting the `PORT` environment variable:
```bash
PORT=3000 go run .
```

## Running Tests

Run all tests:
```bash
go test ./...
```

Run tests with verbose output:
```bash
go test -v ./...
```

Run tests with coverage:
```bash
go test -cover ./...
```

## Initial Configuration

The service initializes with 3 resources by default:
- `resource-1` with capacity 5
- `resource-2` with capacity 3
- `resource-3` with capacity 4

These can be modified in `main.go`.

## Example Usage

### Create a node
```bash
curl -X POST http://localhost:8080/nodes \
  -H "Content-Type: application/json" \
  -d '{"entity_name": "task-1"}'
```

### Move a node to a resource
```bash
curl -X POST http://localhost:8080/nodes/{node-id}/move \
  -H "Content-Type: application/json" \
  -d '{"target_resource_id": "resource-1"}'
```

### Complete a node
```bash
curl -X POST http://localhost:8080/nodes/{node-id}/complete
```

### List all nodes
```bash
curl http://localhost:8080/nodes
```

### List all resources
```bash
curl http://localhost:8080/resources
```

## Project Structure

```
nodequeue-service/
├── db/                # Optional database persistence (store.go, postgres.go)
│   ├── postgres.go
│   └── store.go
├── node/              # Node and entity definitions
│   └── node.go
├── queueservice/      # Core queue and resource management logic
│   └── queue_service.go
├── resource/          # Resource model and configuration
│   └── resource.go
├── main.go            # Entry point, HTTP server setup
├── routes.go          # HTTP route registration
├── README.md          # This file
```

- `main.go`: Sets up server, routes, configuration, starts service.
- `routes.go`: Registers HTTP endpoints (nodes, resources, moves, allocation, completion, etc).
- `resource/`: Resource model & config (load from CSV or use defaults).
- `node/`: Node and entity struct definitions (ID, entity data, logs).
- `queueservice/`: All queue, node, and resource management logic.
- `db/`: Optional persistence layer and interfaces.
- `README.md`: Documentation (usage, structure, HTTP API, tests).

## Persistence with Postgres

The NodeQueue service can optionally persist its state (nodes, resources, and logs) in a Postgres database. This is *optional*—if database configuration is omitted or the database is unavailable, the service runs in memory only.

### Enabling Postgres Persistence

1. **Configure Environment Variables**

   Set the following environment variables for Postgres in your shell or host environment:

   ```
   POSTGRES_HOST=localhost
   POSTGRES_PORT=5432
   POSTGRES_USER=your-username
   POSTGRES_PASSWORD=your-password
   POSTGRES_DB=your-database
   ```

   By default, if these are not set, the service will skip database setup and store data only in memory.

2. **Start the Service**

   Start the service as usual (`go run main.go`). On startup, it will attempt to connect to the database; if successful, it persists node, resource, and log data. If not, it will log a warning and continue in-memory.

3. **Schema Initialization**

   The service will attempt to create required tables automatically if they don’t exist. You do not need to manually initialize tables; however, you may want to check or customize Postgres permissions as needed.

### When is data saved?

- **On node or resource mutation** (create, move, allocate, complete), operations are recorded in the Postgres tables.
- **On startup**, if persistence is enabled, historical node/resource state is restored from the database.
- **If Postgres is unavailable**, all actions are stored in memory only (non-durable).

### Disabling Persistence

Just unset (or do not set) the `POSTGRES_*` environment variables and the service will use memory-only operation.

### Database Table Structure (autogenerated)

The tables created (see `db/` package) are:

- `nodes`: Metadata for each node
- `resources`: Resource definitions
- `node_logs`: Actions/events associated with each node
- (Optionally) other bookkeeping tables as required

This persistence implementation is intended as a simple best-effort mechanism—your service will not crash if the database is misconfigured or missing (see logs for warnings).

**Note:** For scaled/high-availability deployments, you may wish to enhance the DB logic for migrations, backups, and concurrency.

