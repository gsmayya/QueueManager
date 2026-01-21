package main

import (
	"net/http"
	"strings"

	"queue-admin/handlers"
)

func setupRoutes(svc *handlers.Service) {
	http.HandleFunc("/health", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	http.HandleFunc("/entities", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			svc.ListEntities(w, r)
		case http.MethodPost:
			svc.CreateEntity(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	http.HandleFunc("/entities/", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		path, ok := strings.CutPrefix(r.URL.Path, "/entities/")
		id, _, _ := strings.Cut(path, "/")
		if !ok || id == "" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		switch r.Method {
		case http.MethodGet:
			svc.GetEntity(w, r, id)
		case http.MethodPut:
			svc.UpdateEntity(w, r, id)
		case http.MethodDelete:
			svc.DeleteEntity(w, r, id)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	http.HandleFunc("/users", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			svc.ListUsers(w, r)
		case http.MethodPost:
			svc.CreateUser(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	http.HandleFunc("/users/", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		path, ok := strings.CutPrefix(r.URL.Path, "/users/")
		id, _, _ := strings.Cut(path, "/")
		if !ok || id == "" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		switch r.Method {
		case http.MethodGet:
			svc.GetUser(w, r, id)
		case http.MethodPut:
			svc.UpdateUser(w, r, id)
		case http.MethodDelete:
			svc.DeleteUser(w, r, id)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	http.HandleFunc("/auth/login", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		svc.Login(w, r)
	}))

	http.HandleFunc("/rooms", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			svc.ListRooms(w, r)
		case http.MethodPost:
			svc.CreateRoom(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	http.HandleFunc("/rooms/", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		path, ok := strings.CutPrefix(r.URL.Path, "/rooms/")
		id, _, _ := strings.Cut(path, "/")
		if !ok || id == "" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		switch r.Method {
		case http.MethodGet:
			svc.GetRoom(w, r, id)
		case http.MethodPut:
			svc.UpdateRoom(w, r, id)
		case http.MethodDelete:
			svc.DeleteRoom(w, r, id)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}
