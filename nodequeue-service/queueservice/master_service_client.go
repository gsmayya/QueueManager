package queueservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type masterEntity struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Phone string  `json:"phone"`
	Email *string `json:"email,omitempty"`
}

func masterServiceBaseURL() string {
	u := os.Getenv("MASTER_SERVICE_URL")
	if u == "" {
		u = "http://localhost:8081"
	}
	return strings.TrimRight(u, "/")
}

func fetchMasterEntityByID(ctx context.Context, id string) (masterEntity, error) {
	if strings.TrimSpace(id) == "" {
		return masterEntity{}, errors.New("entity_id is required")
	}
	url := fmt.Sprintf("%s/entities/%s", masterServiceBaseURL(), id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return masterEntity{}, err
	}

	cli := &http.Client{Timeout: 5 * time.Second}
	res, err := cli.Do(req)
	if err != nil {
		return masterEntity{}, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return masterEntity{}, fmt.Errorf("master-service returned %d", res.StatusCode)
	}
	var out masterEntity
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return masterEntity{}, err
	}
	return out, nil
}
