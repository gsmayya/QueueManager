package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	. "queue-admin/handlers"
	"queue-common/models"
)

type listByPhoneStore struct {
	*memStore
	lastPhone string
}

func (s *listByPhoneStore) ListEntitiesByPhone(ctx context.Context, phone string) ([]models.Entity, error) {
	s.lastPhone = phone
	// return empty by default for this unit test
	return []models.Entity{}, nil
}

func TestListEntities_WithPhoneQuery_UsesPhoneFilter(t *testing.T) {
	st := &listByPhoneStore{memStore: newMemStore()}
	svc := NewService(st, st, nil)

	req := httptest.NewRequest(http.MethodGet, "/entities?phone=123", nil)
	w := httptest.NewRecorder()
	svc.ListEntities(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if st.lastPhone != "123" {
		t.Fatalf("expected phone filter '123', got %q", st.lastPhone)
	}
	var got []models.Entity
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
}
