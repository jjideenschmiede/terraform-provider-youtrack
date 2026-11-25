// Copyright 2026 J&J Ideenschmiede GmbH
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type fakeYouTrack struct {
	mu        sync.Mutex
	nextID    int
	templates map[string]string
	projects  map[string]map[string]any
	bundles   map[string]*fakeBundle
}

type fakeBundle struct {
	name   string
	values []map[string]any
}

func newFakeYouTrack(t *testing.T) *httptest.Server {
	fake := &fakeYouTrack{
		templates: map[string]string{"helpdesk_footer": "default"},
		projects:  map[string]map[string]any{},
		bundles:   map[string]*fakeBundle{},
	}
	server := httptest.NewServer(http.HandlerFunc(fake.serve))
	t.Cleanup(server.Close)
	return server
}

func (f *fakeYouTrack) id(prefix string) string {
	f.nextID++
	return fmt.Sprintf("%s-%d", prefix, f.nextID)
}

func (f *fakeYouTrack) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if r.Header.Get("Authorization") != "Bearer test-token" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var body map[string]any
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/"), "/")
	route := strings.Join(parts, "/")

	switch {
	case len(parts) == 3 && parts[0] == "admin" && parts[1] == "notificationTemplates":
		f.template(w, r, parts[2], body)
	case route == "admin/projects" && r.Method == http.MethodPost:
		id := f.id("0")
		project := map[string]any{"id": id, "archived": false, "fromEmail": "", "replyToEmail": nil, "description": nil}
		merge(project, body)
		f.projects[id] = project
		write(w, project)
	case len(parts) == 3 && parts[0] == "admin" && parts[1] == "projects":
		project, ok := f.projects[parts[2]]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		switch r.Method {
		case http.MethodDelete:
			delete(f.projects, parts[2])
		case http.MethodPost:
			merge(project, body)
			write(w, project)
		default:
			write(w, project)
		}
	case strings.HasPrefix(route, "admin/customFieldSettings/bundles/enum"):
		f.bundle(w, r, parts[4:], body)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (f *fakeYouTrack) template(w http.ResponseWriter, r *http.Request, id string, body map[string]any) {
	if r.Method == http.MethodPost {
		f.templates[id] = body["content"].(string)
	}
	content, ok := f.templates[id]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	write(w, map[string]any{"id": id, "content": content, "$type": "NotificationTemplate"})
}

func (f *fakeYouTrack) bundle(w http.ResponseWriter, r *http.Request, parts []string, body map[string]any) {
	if len(parts) == 0 {
		id := f.id("66")
		f.bundles[id] = &fakeBundle{name: body["name"].(string)}
		write(w, map[string]any{"id": id})
		return
	}
	bundle, ok := f.bundles[parts[0]]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	switch {
	case len(parts) == 1 && r.Method == http.MethodDelete:
		delete(f.bundles, parts[0])
	case len(parts) == 1 && r.Method == http.MethodPost:
		bundle.name = body["name"].(string)
		write(w, map[string]any{"id": parts[0]})
	case len(parts) == 1:
		write(w, map[string]any{"id": parts[0], "name": bundle.name, "values": bundle.values})
	case len(parts) == 2 && r.Method == http.MethodPost:
		value := map[string]any{"id": f.id("67")}
		merge(value, body)
		bundle.values = append(bundle.values, value)
		write(w, value)
	case len(parts) == 3:
		for index, value := range bundle.values {
			if value["id"] != parts[2] {
				continue
			}
			if r.Method == http.MethodDelete {
				bundle.values = append(bundle.values[:index], bundle.values[index+1:]...)
				return
			}
			merge(value, body)
			write(w, value)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func merge(target, source map[string]any) {
	for key, value := range source {
		target[key] = value
	}
}

func write(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
