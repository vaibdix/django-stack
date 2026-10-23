package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

const simpleJSON = "application/vnd.pypi.simple.v1+json"

type dateIndex struct {
	upstream string
	cutoff   time.Time
	client   *http.Client

	mu    sync.Mutex
	pages map[string][]byte
}

var projectName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

func startDateIndex(ctx context.Context, upstream string, cutoff time.Time) (string, func(), error) {
	var config net.ListenConfig
	listener, err := config.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}
	index := &dateIndex{
		upstream: strings.TrimRight(upstream, "/") + "/",
		cutoff:   cutoff,
		client:   &http.Client{Timeout: time.Minute},
		pages:    map[string][]byte{},
	}
	server := &http.Server{Handler: index, ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = server.Serve(listener) }()
	stop := func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}
	return "http://" + listener.Addr().String() + "/simple/", stop, nil
}

func (d *dateIndex) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name, ok := strings.CutPrefix(r.URL.Path, "/simple/")
	name = strings.Trim(name, "/")
	if !ok || !projectName.MatchString(name) {
		http.NotFound(w, r)
		return
	}
	page, err := d.page(r.Context(), name)
	var status statusError
	switch {
	case errors.As(err, &status):
		http.Error(w, err.Error(), int(status))
		return
	case err != nil:
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", simpleJSON)
	_, _ = w.Write(page)
}

type statusError int

func (s statusError) Error() string { return http.StatusText(int(s)) }

func (d *dateIndex) page(ctx context.Context, name string) ([]byte, error) {
	d.mu.Lock()
	cached, ok := d.pages[name]
	d.mu.Unlock()
	if ok {
		return cached, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.upstream+name+"/", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", simpleJSON)
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, statusError(resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	filtered, err := filterByUploadTime(body, d.cutoff)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}

	d.mu.Lock()
	d.pages[name] = filtered
	d.mu.Unlock()
	return filtered, nil
}

func filterByUploadTime(body []byte, cutoff time.Time) ([]byte, error) {
	var project map[string]json.RawMessage
	if err := json.Unmarshal(body, &project); err != nil {
		return nil, err
	}
	var files []map[string]json.RawMessage
	if err := json.Unmarshal(project["files"], &files); err != nil {
		return nil, err
	}
	kept := make([]map[string]json.RawMessage, 0, len(files))
	for _, file := range files {
		var uploaded string
		if err := json.Unmarshal(file["upload-time"], &uploaded); err != nil {
			continue
		}
		at, err := time.Parse(time.RFC3339Nano, uploaded)
		if err != nil || at.After(cutoff) {
			continue
		}
		kept = append(kept, file)
	}
	encoded, err := json.Marshal(kept)
	if err != nil {
		return nil, err
	}
	project["files"] = encoded
	return json.Marshal(project)
}
