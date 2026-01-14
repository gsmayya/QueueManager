package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"queue-common/models"
	"queue-common/store"

	. "queue-admin/handlers"

	"golang.org/x/crypto/bcrypt"
)

type memStore struct {
	entities map[string]models.Entity
	users    map[string]models.User

	// For testing hashing behavior
	lastPasswordHash string
}

func newMemStore() *memStore {
	return &memStore{
		entities: map[string]models.Entity{},
		users:    map[string]models.User{},
	}
}

func (m *memStore) CreateEntity(ctx context.Context, in models.CreateEntityRequest) (models.Entity, error) {
	// enforce unique(name,phone)
	for _, e := range m.entities {
		if e.Name == in.Name && e.Phone == in.Phone {
			return models.Entity{}, store.ErrConflict
		}
	}
	e := models.Entity{
		ID:          "11111111-1111-1111-1111-111111111111",
		Name:        in.Name,
		Phone:       in.Phone,
		Email:       in.Email,
		JoiningDate: mustTime("2025-01-01T00:00:00Z"),
	}
	m.entities[e.ID] = e
	return e, nil
}

func (m *memStore) ListEntities(ctx context.Context) ([]models.Entity, error) {
	out := make([]models.Entity, 0, len(m.entities))
	for _, e := range m.entities {
		out = append(out, e)
	}
	return out, nil
}

func (m *memStore) ListEntitiesByPhone(ctx context.Context, phone string) ([]models.Entity, error) {
	out := make([]models.Entity, 0)
	for _, e := range m.entities {
		if e.Phone == phone {
			out = append(out, e)
		}
	}
	return out, nil
}

func (m *memStore) GetEntity(ctx context.Context, id string) (models.Entity, error) {
	e, ok := m.entities[id]
	if !ok {
		return models.Entity{}, store.ErrNotFound
	}
	return e, nil
}

func (m *memStore) UpdateEntity(ctx context.Context, id string, in models.UpdateEntityRequest) (models.Entity, error) {
	e, ok := m.entities[id]
	if !ok {
		return models.Entity{}, store.ErrNotFound
	}
	if in.Name != nil {
		e.Name = *in.Name
	}
	if in.Phone != nil {
		e.Phone = *in.Phone
	}
	if in.Email != nil {
		e.Email = in.Email
	}
	if in.JoiningDate != nil {
		e.JoiningDate = *in.JoiningDate
	}
	// enforce unique(name,phone)
	for k, other := range m.entities {
		if k == id {
			continue
		}
		if other.Name == e.Name && other.Phone == e.Phone {
			return models.Entity{}, store.ErrConflict
		}
	}
	m.entities[id] = e
	return e, nil
}

func (m *memStore) DeleteEntity(ctx context.Context, id string) error {
	if _, ok := m.entities[id]; !ok {
		return store.ErrNotFound
	}
	delete(m.entities, id)
	return nil
}

func (m *memStore) CreateUser(ctx context.Context, userID, name, email, passwordHash string) (models.User, error) {
	// enforce uniqueness
	for _, u := range m.users {
		if u.UserID == userID || u.Email == email {
			return models.User{}, store.ErrConflict
		}
	}
	m.lastPasswordHash = passwordHash
	u := models.User{
		ID:        "22222222-2222-2222-2222-222222222222",
		UserID:    userID,
		Name:      name,
		Email:     email,
		CreatedAt: mustTime("2025-01-01T00:00:00Z"),
	}
	m.users[u.ID] = u
	return u, nil
}

func (m *memStore) ListUsers(ctx context.Context) ([]models.User, error) {
	out := make([]models.User, 0, len(m.users))
	for _, u := range m.users {
		out = append(out, u)
	}
	return out, nil
}

func (m *memStore) GetUser(ctx context.Context, id string) (models.User, error) {
	u, ok := m.users[id]
	if !ok {
		return models.User{}, store.ErrNotFound
	}
	return u, nil
}

func (m *memStore) UpdateUser(ctx context.Context, id string, userID, name, email *string, passwordHash *string) (models.User, error) {
	u, ok := m.users[id]
	if !ok {
		return models.User{}, store.ErrNotFound
	}
	if userID != nil {
		u.UserID = *userID
	}
	if name != nil {
		u.Name = *name
	}
	if email != nil {
		u.Email = *email
	}
	if passwordHash != nil {
		m.lastPasswordHash = *passwordHash
	}
	// enforce uniqueness
	for k, other := range m.users {
		if k == id {
			continue
		}
		if other.UserID == u.UserID || other.Email == u.Email {
			return models.User{}, store.ErrConflict
		}
	}
	m.users[id] = u
	return u, nil
}

func (m *memStore) DeleteUser(ctx context.Context, id string) error {
	if _, ok := m.users[id]; !ok {
		return store.ErrNotFound
	}
	delete(m.users, id)
	return nil
}

func mustTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestEntitiesCRUD(t *testing.T) {
	st := newMemStore()
	svc := NewService(st, nil)

	// Create
	body, _ := json.Marshal(map[string]any{"name": "Acme", "phone": "123", "email": "a@b.com"})
	req := httptest.NewRequest(http.MethodPost, "/entities", bytes.NewReader(body))
	w := httptest.NewRecorder()
	svc.CreateEntity(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Conflict on same (name, phone)
	req = httptest.NewRequest(http.MethodPost, "/entities", bytes.NewReader(body))
	w = httptest.NewRecorder()
	svc.CreateEntity(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 conflict, got %d: %s", w.Code, w.Body.String())
	}

	// List
	req = httptest.NewRequest(http.MethodGet, "/entities", nil)
	w = httptest.NewRecorder()
	svc.ListEntities(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestUsersCreate_HashesPassword(t *testing.T) {
	st := newMemStore()
	svc := NewService(st, nil)

	body, _ := json.Marshal(map[string]any{
		"user_id":  "u1",
		"name":     "User One",
		"email":    "u1@example.com",
		"password": "secret",
	})
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
	w := httptest.NewRecorder()
	svc.CreateUser(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if st.lastPasswordHash == "" || st.lastPasswordHash == "secret" {
		t.Fatalf("expected hashed password, got %q", st.lastPasswordHash)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(st.lastPasswordHash), []byte("secret")); err != nil {
		t.Fatalf("stored hash does not match password: %v", err)
	}
}
