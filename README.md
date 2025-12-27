# Queue Manager 

A simple service for managing the queue manager system. Core functionality is in the go as service which provides APIs to manage nodes, resources etc. 
UI is used for simulating and visualizing the queue manager system and to do tests. 



## UI Features

- **Visual Resource Boxes**: Each resource is displayed as a box showing:
  - Resource ID and current capacity usage
  - Service Queue (nodes currently consuming capacity)
  - Waiting Queue (nodes assigned but waiting for capacity)

- **Node Management**:
  - Create new nodes with optional immediate assignment to a resource
  - Move nodes between resources
  - Allocate nodes from waiting queue to service queue (respects capacity limits)
  - Complete nodes (removes them from resources)

- **Real-time Updates**: The UI automatically refreshes every 2 seconds to show the latest state

## Usage 

- Web UI: `http://localhost:3000`
- API: `http://localhost:8080`


### Using Docker in production 

```bash
docker compose up --build 
```

### Development (hot reload)

```bash
docker compose -f docker-compose.dev.yml up --build
```

### For manual running

1. **Start the backend service**:
   ```bash
   cd nodequeue-service
   go run .
   ```
   The service will start on `http://localhost:8080`

2. **Use the Next.js UI (recommended)**:
   ```bash
   cd nodequeue-frontend
   # Create a local env file based on env.example:
   cp env.example .env.local
   npm install
   npm run dev
   ```
   Then open `http://localhost:3000`.


## UI Controls

### Create Node
- Enter a node name in the input field
- Optionally select a resource to add the node to immediately
- Click "Create Node" button

### Node Actions
Each node displays action buttons based on its state:

- **Allocate**: Available for nodes in the waiting queue. Moves the node to the service queue if capacity allows.
- **Move**: Available for nodes assigned to a resource. Moves the node to a different resource.
- **Add to Resource**: Available for unassigned nodes. Adds the node to a resource's waiting queue.
- **Complete**: Marks the node as completed and removes it from its resource.

## Visual Indicators

- **Service Queue**: Green dashed border - nodes currently consuming resource capacity
- **Waiting Queue**: Orange dashed border - nodes waiting for available capacity
- **Completed Nodes**: Grayed out and cannot be moved or allocated
- **Capacity Display**: Shows current usage vs. total capacity (e.g., "3 / 5")

## API Integration

The UI uses the following API endpoints:
- `GET /resources` - List all resources
- `GET /nodes` - List all nodes
- `POST /nodes` - Create a new node (with optional `resource_id`)
- `POST /nodes/{id}/move` - Move a node to another resource
- `POST /nodes/{id}/allocate` - Allocate a waiting node to service queue
- `POST /nodes/{id}/complete` - Complete a node

All endpoints support CORS for cross-origin requests.
