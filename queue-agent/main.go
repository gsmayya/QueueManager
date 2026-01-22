package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"queue-agent/contextcache"
	"queue-agent/queueserviceclient"

	"queue-common/models"
)

// Minimal MCP server over HTTP JSON-RPC 2.0.
// Focused on fetching queue-service metrics and answering questions using cached context.

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type server struct {
	qsc   *queueserviceclient.Client
	cache *contextcache.Cache
}

func (s *server) handle(req rpcRequest) (*rpcResponse, bool) {
	// Returns (response, shouldExit)
	if req.JSONRPC != "" && req.JSONRPC != "2.0" {
		return &rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &rpcError{Code: -32600, Message: "invalid jsonrpc version"},
		}, false
	}

	switch req.Method {
	case "initialize":
		// MCP initialize:
		// https://modelcontextprotocol.io/specification (high level)
		// We accept any params for now and respond with a basic capability set.
		return &rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"protocolVersion": "2024-11-05",
				"serverInfo": map[string]any{
					"name":    "queue-agent",
					"version": "0.1.0",
				},
				"capabilities": map[string]any{
					"tools": map[string]any{},
				},
			},
		}, false

	case "tools/list":
		return &rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"tools": []any{
					map[string]any{
						"name":        "queue_nodes_metrics",
						"description": "Fetches node metrics from queue-service (GET /nodes/metrics), caches it, and returns an English summary plus raw JSON.",
						"inputSchema": map[string]any{
							"type":                 "object",
							"properties":           map[string]any{},
							"additionalProperties": false,
						},
					},
					map[string]any{
						"name":        "queue_resources_metrics",
						"description": "Fetches resource metrics from queue-service (GET /resources/metrics), caches it, and returns an English summary plus raw JSON.",
						"inputSchema": map[string]any{
							"type":                 "object",
							"properties":           map[string]any{},
							"additionalProperties": false,
						},
					},
					map[string]any{
						"name":        "queue_metrics_question",
						"description": "Answers a question in English using cached metrics context. Input: {\"question\": \"...\"}. (Run queue_nodes_metrics / queue_resources_metrics first.)",
						"inputSchema": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"question": map[string]any{
									"type":        "string",
									"description": "Natural-language question about node/resource metrics",
								},
							},
							"required":             []string{"question"},
							"additionalProperties": false,
						},
					},
				},
			},
		}, false

	case "tools/call":
		type toolCallParams struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments,omitempty"`
		}
		var p toolCallParams
		if len(req.Params) > 0 {
			if err := json.Unmarshal(req.Params, &p); err != nil {
				return rpcErr(req.ID, -32602, "invalid params"), false
			}
		}
		if p.Name == "" {
			return rpcErr(req.ID, -32602, "missing tool name"), false
		}

		switch p.Name {
		case "queue_nodes_metrics":
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			m, raw, err := s.qsc.FetchNodesMetrics(ctx)
			if err != nil {
				return rpcErr(req.ID, -32000, "queue_nodes_metrics failed: "+err.Error()), false
			}
			s.cache.SetNodes(contextcache.Snapshot[models.NodesMetricsResponse]{
				FetchedAt: time.Now(),
				RawJSON:   raw,
				Data:      m,
			})

			summary := summarizeNodes(m)
			return toolOK(req.ID,
				summary,
				"Raw JSON (/nodes/metrics):\n"+string(raw),
			), false

		case "queue_resources_metrics":
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			m, raw, err := s.qsc.FetchResourcesMetrics(ctx)
			if err != nil {
				return rpcErr(req.ID, -32000, "queue_resources_metrics failed: "+err.Error()), false
			}
			s.cache.SetResources(contextcache.Snapshot[models.ResourcesSessionMetricsResponse]{
				FetchedAt: time.Now(),
				RawJSON:   raw,
				Data:      m,
			})

			summary := summarizeResources(m)
			return toolOK(req.ID,
				summary,
				"Raw JSON (/resources/metrics):\n"+string(raw),
			), false

		case "queue_metrics_question":
			type qArgs struct {
				Question string `json:"question"`
			}
			var a qArgs
			if len(p.Arguments) > 0 && string(p.Arguments) != "null" {
				if err := json.Unmarshal(p.Arguments, &a); err != nil {
					return rpcErr(req.ID, -32602, "invalid tool arguments"), false
				}
			}
			q := strings.TrimSpace(a.Question)
			if q == "" {
				return rpcErr(req.ID, -32602, "question is required"), false
			}

			answer, rawContext, err := answerQuestion(s.cache, q)
			if err != nil {
				return rpcErr(req.ID, -32000, err.Error()), false
			}
			return toolOK(req.ID, answer, rawContext), false

		default:
			return rpcErr(req.ID, -32601, "unknown tool: "+p.Name), false
		}

	case "shutdown":
		return &rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}}, true

	default:
		// Ignore notifications we don't understand, but error on requests with an id.
		if req.ID == nil {
			return nil, false
		}
		return rpcErr(req.ID, -32601, "method not found: "+req.Method), false
	}
}

func rpcErr(id any, code int, msg string) *rpcResponse {
	return &rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: code, Message: msg},
	}
}

func toolOK(id any, parts ...string) *rpcResponse {
	content := make([]any, 0, len(parts))
	for _, p := range parts {
		content = append(content, map[string]any{
			"type": "text",
			"text": p,
		})
	}
	return &rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]any{
			"content": content,
			"isError": false,
		},
	}
}

func (s *server) serveHTTP(addr string) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	mux.HandleFunc("/rpc", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		defer r.Body.Close()

		var req rpcRequest
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(rpcResponse{
				JSONRPC: "2.0",
				Error:   &rpcError{Code: -32700, Message: "parse error"},
			})
			return
		}

		resp, _ := s.handle(req)
		if resp == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(resp)
	})

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	return srv.ListenAndServe()
}

func summarizeNodes(m models.NodesMetricsResponse) string {
	active := len(m.ActiveNodes)
	completed := len(m.CompletedNodes)

	// Top 3 by total time (active + completed)
	all := make([]models.NodeMetrics, 0, active+completed)
	all = append(all, m.ActiveNodes...)
	all = append(all, m.CompletedNodes...)
	sort.SliceStable(all, func(i, j int) bool {
		return all[i].TotalTimeInSystemMS > all[j].TotalTimeInSystemMS
	})

	topN := 3
	if len(all) < topN {
		topN = len(all)
	}
	lines := []string{
		fmt.Sprintf("Node metrics summary: %d active nodes, %d completed nodes.", active, completed),
	}
	if topN > 0 {
		lines = append(lines, "Top nodes by total time in system:")
		for i := 0; i < topN; i++ {
			n := all[i]
			name := n.NodeName
			if strings.TrimSpace(name) == "" {
				name = n.ID
			}
			lines = append(lines, fmt.Sprintf("- %s (entity=%s, completed=%v): %d ms", name, n.EntityName, n.Completed, n.TotalTimeInSystemMS))
		}
	}
	return strings.Join(lines, "\n")
}

func summarizeResources(m models.ResourcesSessionMetricsResponse) string {
	if len(m.Resources) == 0 {
		return "Resource metrics summary: 0 resources reported."
	}

	// Highest avg waiting.
	best := m.Resources[0]
	for _, r := range m.Resources[1:] {
		if r.AvgWaitingTimeMS > best.AvgWaitingTimeMS {
			best = r
		}
	}

	lines := []string{
		fmt.Sprintf("Resource metrics summary: %d resources reported.", len(m.Resources)),
		fmt.Sprintf("Highest avg waiting time: %s (%s) = %d ms (segments=%d).",
			best.ResourceName, best.ResourceID, best.AvgWaitingTimeMS, best.WaitingSegmentsCount),
	}
	return strings.Join(lines, "\n")
}

var topNRe = regexp.MustCompile(`\btop\s+(\d+)\b`)
var nodeRefRe = regexp.MustCompile(`\bnode(?:\s+id)?\s*[:=#]?\s*([a-zA-Z0-9_-]{4,})\b`)
var nodeName4DigitsRe = regexp.MustCompile(`\b(\d{4})\b`)

func findNode(m models.NodesMetricsResponse, ref string) (models.NodeMetrics, bool) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return models.NodeMetrics{}, false
	}
	refLower := strings.ToLower(ref)

	all := make([]models.NodeMetrics, 0, len(m.ActiveNodes)+len(m.CompletedNodes))
	all = append(all, m.ActiveNodes...)
	all = append(all, m.CompletedNodes...)

	for _, n := range all {
		if n.ID == ref {
			return n, true
		}
		if strings.EqualFold(n.NodeName, ref) {
			return n, true
		}
		// Allow short lookup by prefix for IDs (helpful in chat).
		if len(ref) >= 6 && strings.HasPrefix(strings.ToLower(n.ID), refLower) {
			return n, true
		}
	}
	return models.NodeMetrics{}, false
}

func listSomeNodes(m models.NodesMetricsResponse, max int) []string {
	all := make([]models.NodeMetrics, 0, len(m.ActiveNodes)+len(m.CompletedNodes))
	all = append(all, m.ActiveNodes...)
	all = append(all, m.CompletedNodes...)
	sort.SliceStable(all, func(i, j int) bool {
		return all[i].CreatedAt.Before(all[j].CreatedAt)
	})
	if len(all) > max {
		all = all[:max]
	}
	out := make([]string, 0, len(all))
	for _, n := range all {
		name := n.NodeName
		if strings.TrimSpace(name) == "" {
			name = n.ID
		}
		out = append(out, fmt.Sprintf("- %s (id=%s, completed=%v)", name, n.ID, n.Completed))
	}
	return out
}

func formatNodeDetails(n models.NodeMetrics) string {
	lines := []string{
		"Node details:",
		fmt.Sprintf("- id: %s", n.ID),
		fmt.Sprintf("- node_name: %s", n.NodeName),
		fmt.Sprintf("- entity_name: %s", n.EntityName),
		fmt.Sprintf("- created_at: %s", n.CreatedAt.Format(time.RFC3339)),
		fmt.Sprintf("- completed: %v", n.Completed),
		fmt.Sprintf("- total_time_in_system_ms: %d", n.TotalTimeInSystemMS),
	}
	if len(n.WaitingSegments) == 0 {
		lines = append(lines, "- waiting_segments: (none)")
		return strings.Join(lines, "\n")
	}
	lines = append(lines, fmt.Sprintf("- waiting_segments: %d", len(n.WaitingSegments)))
	// Show last up to 3 segments for brevity.
	start := 0
	if len(n.WaitingSegments) > 3 {
		start = len(n.WaitingSegments) - 3
	}
	for i := start; i < len(n.WaitingSegments); i++ {
		seg := n.WaitingSegments[i]
		lines = append(lines, fmt.Sprintf("  - %s (%s): %d ms [%s → %s]",
			seg.ResourceName, seg.ResourceID, seg.DurationMS,
			seg.StartTS.Format(time.RFC3339), seg.EndTS.Format(time.RFC3339)))
	}
	return strings.Join(lines, "\n")
}

func answerQuestion(c *contextcache.Cache, question string) (english string, rawContext string, _ error) {
	q := strings.ToLower(strings.TrimSpace(question))

	nodesSnap, haveNodes := c.GetNodes()
	resSnap, haveRes := c.GetResources()

	if !haveNodes && !haveRes {
		return "", "", fmt.Errorf("no cached metrics available yet. Run queue_nodes_metrics and/or queue_resources_metrics first")
	}

	// 1) Active vs completed nodes.
	if strings.Contains(q, "active") && strings.Contains(q, "completed") && strings.Contains(q, "node") {
		if !haveNodes {
			return "", "", fmt.Errorf("nodes metrics not cached yet. Run queue_nodes_metrics first")
		}
		english = fmt.Sprintf("There are %d active nodes and %d completed nodes (cached at %s).",
			len(nodesSnap.Data.ActiveNodes), len(nodesSnap.Data.CompletedNodes), nodesSnap.FetchedAt.Format(time.RFC3339))
		return english, "Raw JSON context (/nodes/metrics):\n" + string(nodesSnap.RawJSON), nil
	}

	// 2) Highest avg waiting resource.
	if strings.Contains(q, "highest") && strings.Contains(q, "avg") && strings.Contains(q, "wait") && strings.Contains(q, "resource") {
		if !haveRes {
			return "", "", fmt.Errorf("resources metrics not cached yet. Run queue_resources_metrics first")
		}
		if len(resSnap.Data.Resources) == 0 {
			return "No resources were returned in cached resource metrics.", "Raw JSON context (/resources/metrics):\n" + string(resSnap.RawJSON), nil
		}
		best := resSnap.Data.Resources[0]
		for _, r := range resSnap.Data.Resources[1:] {
			if r.AvgWaitingTimeMS > best.AvgWaitingTimeMS {
				best = r
			}
		}
		english = fmt.Sprintf("The resource with the highest average waiting time is %s (%s): %d ms.",
			best.ResourceName, best.ResourceID, best.AvgWaitingTimeMS)
		return english, "Raw JSON context (/resources/metrics):\n" + string(resSnap.RawJSON), nil
	}

	// 2b) Which resources still have nodes?
	if strings.Contains(q, "resource") && strings.Contains(q, "node") &&
		(strings.Contains(q, "still") || strings.Contains(q, "have") || strings.Contains(q, "non-empty") || strings.Contains(q, "remaining")) {
		if !haveRes {
			return "", "", fmt.Errorf("resources metrics not cached yet. Run queue_resources_metrics first")
		}
		type row struct {
			name string
			id   string
			w    int
			a    int
			t    int
		}
		rows := make([]row, 0)
		for _, r := range resSnap.Data.Resources {
			total := r.CurrentWaiting + r.CurrentAllocated
			if total <= 0 {
				continue
			}
			rows = append(rows, row{name: r.ResourceName, id: r.ResourceID, w: r.CurrentWaiting, a: r.CurrentAllocated, t: total})
		}
		if len(rows) == 0 {
			return "No resources currently have nodes in waiting or allocated queues.", "Raw JSON context (/resources/metrics):\n" + string(resSnap.RawJSON), nil
		}
		sort.SliceStable(rows, func(i, j int) bool {
			if rows[i].t != rows[j].t {
				return rows[i].t > rows[j].t
			}
			return rows[i].id < rows[j].id
		})
		lines := []string{"Resources that currently still have nodes:"}
		for _, r := range rows {
			lines = append(lines, fmt.Sprintf("- %s (%s): waiting=%d allocated=%d", r.name, r.id, r.w, r.a))
		}
		return strings.Join(lines, "\n"), "Raw JSON context (/resources/metrics):\n" + string(resSnap.RawJSON), nil
	}

	// 3) Top N nodes by total time.
	if strings.Contains(q, "top") && strings.Contains(q, "node") && (strings.Contains(q, "time") || strings.Contains(q, "system")) {
		if !haveNodes {
			return "", "", fmt.Errorf("nodes metrics not cached yet. Run queue_nodes_metrics first")
		}
		n := 5
		if m := topNRe.FindStringSubmatch(q); len(m) == 2 {
			if v, err := strconv.Atoi(m[1]); err == nil && v > 0 && v <= 50 {
				n = v
			}
		}

		all := make([]models.NodeMetrics, 0, len(nodesSnap.Data.ActiveNodes)+len(nodesSnap.Data.CompletedNodes))
		all = append(all, nodesSnap.Data.ActiveNodes...)
		all = append(all, nodesSnap.Data.CompletedNodes...)
		sort.SliceStable(all, func(i, j int) bool {
			return all[i].TotalTimeInSystemMS > all[j].TotalTimeInSystemMS
		})
		if len(all) < n {
			n = len(all)
		}
		if n == 0 {
			return "No nodes were returned in cached node metrics.", "Raw JSON context (/nodes/metrics):\n" + string(nodesSnap.RawJSON), nil
		}
		lines := []string{fmt.Sprintf("Top %d nodes by total time in system (cached at %s):", n, nodesSnap.FetchedAt.Format(time.RFC3339))}
		for i := 0; i < n; i++ {
			nm := all[i]
			name := nm.NodeName
			if strings.TrimSpace(name) == "" {
				name = nm.ID
			}
			lines = append(lines, fmt.Sprintf("- %s: %d ms (completed=%v)", name, nm.TotalTimeInSystemMS, nm.Completed))
		}
		english = strings.Join(lines, "\n")
		return english, "Raw JSON context (/nodes/metrics):\n" + string(nodesSnap.RawJSON), nil
	}

	// 4) Node details (by id or node_name).
	if strings.Contains(q, "node") && (strings.Contains(q, "detail") || strings.Contains(q, "details") || strings.Contains(q, "info")) {
		if !haveNodes {
			return "", "", fmt.Errorf("nodes metrics not cached yet. Run queue_nodes_metrics first")
		}
		ref := ""
		if m := nodeRefRe.FindStringSubmatch(question); len(m) == 2 {
			ref = m[1]
		} else if m := nodeName4DigitsRe.FindStringSubmatch(question); len(m) == 2 {
			ref = m[1]
		}

		if ref == "" {
			// If only one active node exists, show it; otherwise ask for a node identifier.
			if len(nodesSnap.Data.ActiveNodes) == 1 {
				n := nodesSnap.Data.ActiveNodes[0]
				english = formatNodeDetails(n)
				return english, "Raw JSON context (/nodes/metrics):\n" + string(nodesSnap.RawJSON), nil
			}
			lines := []string{
				"Please specify a node id or node name (4 digits). Examples:",
				"- \"node details for 1234\"",
				"- \"node details for <node-id>\"",
			}
			lines = append(lines, "Here are some nodes:")
			lines = append(lines, listSomeNodes(nodesSnap.Data, 5)...)
			return strings.Join(lines, "\n"), "Raw JSON context (/nodes/metrics):\n" + string(nodesSnap.RawJSON), nil
		}

		n, ok := findNode(nodesSnap.Data, ref)
		if !ok {
			lines := []string{
				fmt.Sprintf("I couldn’t find a node matching %q in cached metrics.", ref),
				"Here are some nodes to choose from:",
			}
			lines = append(lines, listSomeNodes(nodesSnap.Data, 8)...)
			return strings.Join(lines, "\n"), "Raw JSON context (/nodes/metrics):\n" + string(nodesSnap.RawJSON), nil
		}
		return formatNodeDetails(n), "Raw JSON context (/nodes/metrics):\n" + string(nodesSnap.RawJSON), nil
	}

	// 5) How long has a node been in a resource?
	// We interpret this as waiting time in a resource derived from node waiting segments.
	if strings.Contains(q, "how long") && strings.Contains(q, "node") && strings.Contains(q, "resource") {
		if !haveNodes {
			return "", "", fmt.Errorf("nodes metrics not cached yet. Run queue_nodes_metrics first")
		}
		ref := ""
		if m := nodeRefRe.FindStringSubmatch(question); len(m) == 2 {
			ref = m[1]
		} else if m := nodeName4DigitsRe.FindStringSubmatch(question); len(m) == 2 {
			ref = m[1]
		}
		if ref == "" {
			return "Please include a node id or node name (4 digits). Example: \"How long has node 1234 been in the resource?\"",
				"Raw JSON context (/nodes/metrics):\n" + string(nodesSnap.RawJSON), nil
		}
		n, ok := findNode(nodesSnap.Data, ref)
		if !ok {
			return fmt.Sprintf("I couldn’t find node %q in cached metrics.", ref),
				"Raw JSON context (/nodes/metrics):\n" + string(nodesSnap.RawJSON), nil
		}
		if len(n.WaitingSegments) == 0 {
			return fmt.Sprintf("Node %s has no recorded waiting segments in cached metrics.", n.ID),
				"Raw JSON context (/nodes/metrics):\n" + string(nodesSnap.RawJSON), nil
		}

		// Try to identify a specific resource mentioned in the question (by name or id).
		var (
			targetRID   string
			targetRName string
		)
		if haveRes {
			for _, r := range resSnap.Data.Resources {
				rl := strings.ToLower(r.ResourceName)
				if rl != "" && strings.Contains(q, rl) {
					targetRID = r.ResourceID
					targetRName = r.ResourceName
					break
				}
				if r.ResourceID != "" && strings.Contains(q, strings.ToLower(r.ResourceID)) {
					targetRID = r.ResourceID
					targetRName = r.ResourceName
					break
				}
			}
		}

		// If no explicit resource, infer the most recent waiting segment.
		if targetRID == "" {
			last := n.WaitingSegments[len(n.WaitingSegments)-1]
			targetRID = last.ResourceID
			targetRName = last.ResourceName
		}

		var totalMS int64
		var segCount int64
		for _, seg := range n.WaitingSegments {
			if seg.ResourceID == targetRID {
				totalMS += seg.DurationMS
				segCount++
			}
		}
		if segCount == 0 {
			return fmt.Sprintf("Node %s has no waiting-segment time recorded for resource %s (%s) in cached metrics.",
					n.ID, targetRName, targetRID),
				"Raw JSON context (/nodes/metrics):\n" + string(nodesSnap.RawJSON), nil
		}
		name := n.NodeName
		if strings.TrimSpace(name) == "" {
			name = n.ID
		}
		return fmt.Sprintf("Node %s has spent %d ms waiting in resource %s (%s) across %d waiting segments (from cached metrics).",
				name, totalMS, targetRName, targetRID, segCount),
			"Raw JSON context (/nodes/metrics):\n" + string(nodesSnap.RawJSON), nil
	}

	// Fallback.
	english = "I can’t answer that yet with the current deterministic question support. Try one of:\n" +
		"- \"How many active vs completed nodes are there?\"\n" +
		"- \"Which resource has the highest avg waiting time?\"\n" +
		"- \"Which resources still have nodes?\"\n" +
		"- \"Node details for 1234\"\n" +
		"- \"How long has node 1234 been in resource <resource-name>?\"\n" +
		"- \"Top 5 nodes by total time in system\""

	// For debugging, include whichever caches are available (may be large).
	ctxParts := make([]string, 0, 2)
	if haveNodes {
		ctxParts = append(ctxParts, "Raw JSON context (/nodes/metrics):\n"+string(nodesSnap.RawJSON))
	}
	if haveRes {
		ctxParts = append(ctxParts, "Raw JSON context (/resources/metrics):\n"+string(resSnap.RawJSON))
	}
	rawContext = strings.Join(ctxParts, "\n\n")
	return english, rawContext, nil
}

func main() {
	baseURL := os.Getenv("QUEUE_SERVICE_BASE_URL")
	httpAddr := os.Getenv("QUEUE_AGENT_HTTP_ADDR")
	if strings.TrimSpace(httpAddr) == "" {
		httpAddr = ":8090"
	}
	s := &server{
		qsc:   queueserviceclient.New(baseURL),
		cache: contextcache.New(),
	}

	if err := s.serveHTTP(httpAddr); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintln(os.Stderr, "queue-agent http server error:", err)
		os.Exit(1)
	}
}
