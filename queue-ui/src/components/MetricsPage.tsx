"use client";

import React, { useCallback, useEffect, useState } from "react";
import { getNodesMetrics, getResourcesMetrics } from "../lib/api";
import type { NodesMetricsResponse, ResourcesSessionMetricsResponse } from "../lib/types";
import { NodeMetricsFrame } from "./NodeMetricsFrame";
import { ResourceMetricsFrame } from "./ResourceMetricsFrame";

export function MetricsPage() {
  const [metrics, setMetrics] = useState<NodesMetricsResponse | null>(null);
  const [metricsLoading, setMetricsLoading] = useState(false);
  const [metricsError, setMetricsError] = useState<string | null>(null);
  const [metricsLastUpdated, setMetricsLastUpdated] = useState<string | null>(null);

  const [resourceMetrics, setResourceMetrics] = useState<ResourcesSessionMetricsResponse | null>(null);
  const [resourceMetricsLoading, setResourceMetricsLoading] = useState(false);
  const [resourceMetricsError, setResourceMetricsError] = useState<string | null>(null);
  const [resourceMetricsLastUpdated, setResourceMetricsLastUpdated] = useState<string | null>(null);

  const refreshMetrics = useCallback(async () => {
    setMetricsLoading(true);
    setMetricsError(null);
    try {
      const data = await getNodesMetrics();
      setMetrics(data);
      setMetricsLastUpdated(new Date().toLocaleTimeString());
    } catch (e) {
      const err = e as Error;
      setMetricsError(err.message);
    } finally {
      setMetricsLoading(false);
    }
  }, []);

  const refreshResourceMetrics = useCallback(async () => {
    setResourceMetricsLoading(true);
    setResourceMetricsError(null);
    try {
      const data = await getResourcesMetrics();
      setResourceMetrics(data);
      setResourceMetricsLastUpdated(new Date().toLocaleTimeString());
    } catch (e) {
      const err = e as Error;
      setResourceMetricsError(err.message);
    } finally {
      setResourceMetricsLoading(false);
    }
  }, []);

  // Poll metrics every 10s (same behavior as before).
  useEffect(() => {
    refreshMetrics().catch(() => {});
    refreshResourceMetrics().catch(() => {});
    const t = setInterval(() => {
      refreshMetrics().catch(() => {});
      refreshResourceMetrics().catch(() => {});
    }, 10000);
    return () => clearInterval(t);
  }, [refreshMetrics, refreshResourceMetrics]);

  return (
    <div className="mx-auto max-w-6xl">
      <header className="mb-6 text-center text-white">
        <h1 className="text-3xl font-semibold tracking-tight">Metrics</h1>
        <p className="mt-2 text-white/80">Room + Node metrics (refreshes every 10s).</p>
      </header>

      <ResourceMetricsFrame
        metrics={resourceMetrics}
        loading={resourceMetricsLoading}
        error={resourceMetricsError}
        lastUpdatedAt={resourceMetricsLastUpdated}
      />

      <NodeMetricsFrame
        metrics={metrics}
        loading={metricsLoading}
        error={metricsError}
        lastUpdatedAt={metricsLastUpdated}
      />
    </div>
  );
}

