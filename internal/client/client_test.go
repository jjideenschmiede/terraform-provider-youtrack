// Copyright 2026 J&J Ideenschmiede GmbH. All rights reserved.

package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestNewRejectsInvalidURL(t *testing.T) {
	for _, value := range []string{"", "example.com", "://broken"} {
		if _, err := New(value, "token", ""); err == nil {
			t.Errorf("New(%q) returned no error", value)
		}
	}
}

func TestGetSendsHeadersAndFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer perm-token" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q", got)
		}
		if r.URL.Path != "/youtrack/api/admin/notificationTemplates/helpdesk_footer" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("fields"); got != NotificationTemplateFields {
			t.Errorf("fields = %q", got)
		}
		_, _ = w.Write([]byte(`{"id":"helpdesk_footer","content":"<#macro footer/>","$type":"NotificationTemplate"}`))
	}))
	defer server.Close()

	c, err := New(server.URL+"/youtrack/", "perm-token", "test")
	if err != nil {
		t.Fatal(err)
	}
	var template NotificationTemplate
	if err := c.Get(context.Background(), "admin/notificationTemplates/helpdesk_footer", NotificationTemplateFields, &template); err != nil {
		t.Fatal(err)
	}
	if template.ID != "helpdesk_footer" || template.Content != "<#macro footer/>" {
		t.Errorf("unexpected template %+v", template)
	}
}

func TestPostSendsJSONBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q", got)
		}
		if got := r.URL.Query().Get("template"); got != "scrum" {
			t.Errorf("template = %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["shortName"] != "DEMO" {
			t.Errorf("shortName = %v", body["shortName"])
		}
		_, _ = w.Write([]byte(`{"id":"0-1","shortName":"DEMO"}`))
	}))
	defer server.Close()

	c, _ := New(server.URL, "token", "")
	var project Project
	query := map[string][]string{"template": {"scrum"}}
	if err := c.PostWithQuery(context.Background(), "admin/projects", ProjectFields, query, map[string]any{"shortName": "DEMO"}, &project); err != nil {
		t.Fatal(err)
	}
	if project.ID != "0-1" {
		t.Errorf("id = %q", project.ID)
	}
}

func TestErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/missing":
			http.Error(w, `{"error":"Not Found"}`, http.StatusNotFound)
		case "/api/forbidden":
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":"Forbidden","error_description":"Low-level Admin Write is required"}`))
		default:
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("boom"))
		}
	}))
	defer server.Close()

	c, _ := New(server.URL, "token", "")
	ctx := context.Background()

	if err := c.Get(ctx, "missing", "", nil); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing: got %v", err)
	}

	var apiErr *APIError
	err := c.Delete(ctx, "forbidden")
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusForbidden || apiErr.Description != "Low-level Admin Write is required" {
		t.Errorf("forbidden: got %#v", err)
	}

	err = c.Post(ctx, "broken", "", map[string]any{}, nil)
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusInternalServerError || apiErr.Description != "boom" {
		t.Errorf("broken: got %#v", err)
	}
}

func TestListPaginates(t *testing.T) {
	const total = 230
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		skip, _ := strconv.Atoi(r.URL.Query().Get("$skip"))
		top, _ := strconv.Atoi(r.URL.Query().Get("$top"))
		var page []UserGroup
		for i := skip; i < total && i < skip+top; i++ {
			page = append(page, UserGroup{ID: fmt.Sprintf("3-%d", i)})
		}
		_ = json.NewEncoder(w).Encode(page)
	}))
	defer server.Close()

	c, _ := New(server.URL, "token", "")
	groups, err := List[UserGroup](context.Background(), c, "groups", UserGroupFields)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != total {
		t.Errorf("len = %d, want %d", len(groups), total)
	}
	if requests != 3 {
		t.Errorf("requests = %d, want 3", requests)
	}
}
