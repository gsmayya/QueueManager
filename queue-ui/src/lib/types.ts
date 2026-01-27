export type Entity = {
  id?: string;
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
  node_name?: string;
  resource_id?: string;
  schedule_id?: string;
  time_limit_seconds?: number;
  waiting_expiry_seconds?: number;
  assigned_at?: string;
  due_at?: string;
  expires_at?: string;
  delay_flag?: boolean;
  expired?: boolean;
  expired_at?: string;
  completed: boolean;
  created_at: string;
  log: NodeLog[];
};

export type Resource = {
  id: string;
  name?: string;
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
  resource_name: string;
  start_ts: string;
  end_ts: string;
  duration_ms: number;
};

export type NodeMetrics = {
  id: string;
  entity_name: string;
  node_name?: string;
  created_at: string;
  schedule_id?: string;
  assigned_at?: string;
  due_at?: string;
  delay_flag?: boolean;
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
  resource_name: string;
  resource_capacity: number;
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

// --- Admin (master-service)

export type AdminEntity = {
  id: string;
  name: string;
  phone: string;
  email?: string;
  joining_date: string;
};

export type AdminUser = {
  id: string;
  user_id: string;
  name: string;
  email: string;
  is_admin: boolean;
  created_at: string;
};

export type AdminRoom = {
  id: string;
  name: string;
  capacity: number;
  deleted_at?: string;
  created_at: string;
};

// --- Scheduling (queue-service)

export type Schedule = {
  id: string;
  entity_id: string;
  resource_id: string;
  interval_seconds: number;
  time_limit_seconds: number;
  waiting_expiry_seconds: number;
  ends_at?: string;
  enabled: boolean;
  next_run_at: string;
  created_at: string;
  updated_at: string;
};

export type ScheduleMetrics = {
  schedule_id: string;
  entity_id: string;
  resource_id: string;
  interval_seconds: number;
  time_limit_seconds: number;
  waiting_expiry_seconds: number;
  ends_at?: string;
  enabled: boolean;
  next_run_at: string;
  created_at: string;
  updated_at: string;

  fired_count: number;
  completed_count: number;
  expired_count: number;
  completed_within_time_limit_count: number;

  avg_assigned_to_allocate_ms?: number;
  avg_assigned_to_complete_ms?: number;
  avg_assigned_to_expired_ms?: number;
};

export type SchedulesMetricsTotals = {
  now: string;
  total_schedules: number;
  enabled_schedules: number;
  ended_schedules: number;
  fired_count: number;
  completed_count: number;
  expired_count: number;
  completed_within_time_limit_count: number;
  avg_assigned_to_allocate_ms?: number;
  avg_assigned_to_complete_ms?: number;
  avg_assigned_to_expired_ms?: number;
};

export type SchedulesMetricsResponse = {
  totals: SchedulesMetricsTotals;
  schedules: ScheduleMetrics[];
};


