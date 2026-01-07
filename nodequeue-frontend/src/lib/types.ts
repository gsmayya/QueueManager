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

export type WaitingSegment = {
  resource_id: string;
  start_ts: string;
  end_ts: string;
  duration_ms: number;
};

export type NodeMetrics = {
  id: string;
  entity_name: string;
  created_at: string;
  completed: boolean;
  total_time_in_system_ms: number;
  waiting_segments: WaitingSegment[];
};

export type NodesMetricsResponse = {
  active_nodes: NodeMetrics[];
  completed_nodes: NodeMetrics[];
};

export type ResourceSessionMetrics = {
  resource_id: string;
  total_added: number;
  total_allocated: number;
  current_waiting: number;
  current_allocated: number;
  waiting_segments_count: number;
  waiting_total_ms: number;
  avg_waiting_time_ms: number;
  service_segments_count: number;
  service_total_ms: number;
  avg_service_time_ms: number;
};

export type ResourcesSessionMetricsResponse = {
  session_start: string;
  now: string;
  resources: ResourceSessionMetrics[];
};


