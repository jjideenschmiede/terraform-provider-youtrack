// Copyright 2026 J&J Ideenschmiede GmbH
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const pageSize = 100

var ErrNotFound = errors.New("youtrack: entity not found")

type Client struct {
	baseURL    *url.URL
	token      string
	userAgent  string
	httpClient *http.Client
}

type APIError struct {
	StatusCode  int
	Message     string
	Description string
}

func (e *APIError) Error() string {
	if e.Description != "" {
		return fmt.Sprintf("youtrack: %d %s: %s", e.StatusCode, e.Message, e.Description)
	}
	return fmt.Sprintf("youtrack: %d %s", e.StatusCode, e.Message)
}

func New(baseURL, token, userAgent string) (*Client, error) {
	parsed, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("invalid url %q: %w", baseURL, err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid url %q: scheme and host are required", baseURL)
	}
	return &Client{
		baseURL:    parsed,
		token:      token,
		userAgent:  userAgent,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}, nil
}

func (c *Client) Get(ctx context.Context, path, fields string, out any) error {
	return c.do(ctx, http.MethodGet, path, fieldsQuery(fields), nil, out)
}

func (c *Client) Post(ctx context.Context, path, fields string, in, out any) error {
	return c.do(ctx, http.MethodPost, path, fieldsQuery(fields), in, out)
}

func (c *Client) Delete(ctx context.Context, path string) error {
	return c.do(ctx, http.MethodDelete, path, nil, nil, nil)
}

func (c *Client) DeleteWithQuery(ctx context.Context, path string, query url.Values) error {
	return c.do(ctx, http.MethodDelete, path, query, nil, nil)
}

func (c *Client) PostWithQuery(ctx context.Context, path, fields string, query url.Values, in, out any) error {
	merged := fieldsQuery(fields)
	for key, values := range query {
		for _, value := range values {
			merged.Add(key, value)
		}
	}
	return c.do(ctx, http.MethodPost, path, merged, in, out)
}

func List[T any](ctx context.Context, c *Client, path, fields string) ([]T, error) {
	var all []T
	for skip := 0; ; skip += pageSize {
		query := fieldsQuery(fields)
		query.Set("$top", strconv.Itoa(pageSize))
		query.Set("$skip", strconv.Itoa(skip))
		var page []T
		if err := c.do(ctx, http.MethodGet, path, query, nil, &page); err != nil {
			return nil, err
		}
		all = append(all, page...)
		if len(page) < pageSize {
			return all, nil
		}
	}
}

func fieldsQuery(fields string) url.Values {
	query := url.Values{}
	if fields != "" {
		query.Set("fields", fields)
	}
	return query
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, in, out any) error {
	endpoint := c.baseURL.JoinPath("api", path)
	endpoint.RawQuery = query.Encode()

	var body io.Reader
	if in != nil {
		payload, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("encode request body: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	request, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+c.token)
	request.Header.Set("Accept", "application/json")
	if in != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if c.userAgent != "" {
		request.Header.Set("User-Agent", c.userAgent)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, endpoint.Path, err)
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if response.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if response.StatusCode >= http.StatusBadRequest {
		return parseError(response.StatusCode, data)
	}

	if out == nil || len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decode response of %s %s: %w", method, endpoint.Path, err)
	}
	return nil
}

func parseError(statusCode int, data []byte) error {
	var payload struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	apiErr := &APIError{StatusCode: statusCode, Message: http.StatusText(statusCode)}
	if err := json.Unmarshal(data, &payload); err == nil && payload.Error != "" {
		apiErr.Message = payload.Error
		apiErr.Description = payload.ErrorDescription
	} else if text := strings.TrimSpace(string(data)); text != "" {
		apiErr.Description = text
	}
	return apiErr
}
