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
├── main.go           # Entry point, HTTP server setup
├── models.go         # Entity, Node, Resource models
├── service.go        # QueueService business logic
├── handlers.go       # HTTP request handlers
├── models_test.go    # Unit tests for models
├── service_test.go   # Unit tests for service
├── handlers_test.go  # Unit tests for handlers
├── go.mod            # Go module dependencies
└── README.md         # This file
```

## Testing

The project includes comprehensive unit tests for:
- Resource operations (add, remove, capacity checks)
- Node creation and management
- Queue service operations (create, move, complete)
- HTTP handlers for all endpoints

All tests follow Go testing conventions and can be run with `go test`.
