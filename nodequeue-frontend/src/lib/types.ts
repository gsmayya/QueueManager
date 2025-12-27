export type Entity = {
  name: string;
};

export type NodeLog = {
  action: string;
  resource_id?: string;
  timestamp: string;
};

export type Node = {
  id: string;
  entity: Entity;
  resource_id?: string;
  completed: boolean;
  created_at: string;
  log: NodeLog[];
};

export type Resource = {
  id: string;
  capacity: number;
  // Service queue (consumes capacity)
  nodes: Node[];
  // Waiting queue (does not consume capacity)
  waiting_queue?: Node[];
};

export type ErrorResponse = {
  error: string;
};


