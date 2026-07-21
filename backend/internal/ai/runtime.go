package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type Runtime struct {
	client    *Client
	autoSleep bool
	mu        sync.Mutex
}

func NewRuntime(
	client *Client,
	autoSleep bool,
) *Runtime {
	return &Runtime{
		client:    client,
		autoSleep: autoSleep,
	}
}

func (runtime *Runtime) Acquire(
	ctx context.Context,
) (func(), error) {
	runtime.mu.Lock()

	if err := runtime.client.EnsureAwake(ctx); err != nil {
		runtime.mu.Unlock()

		return nil, fmt.Errorf(
			"wake VLM runtime: %w",
			err,
		)
	}

	var once sync.Once

	release := func() {
		once.Do(func() {
			defer runtime.mu.Unlock()

			if !runtime.autoSleep {
				return
			}

			sleepContext, cancel :=
				context.WithTimeout(
					context.Background(),
					90*time.Second,
				)

			defer cancel()

			// Sleep 失败不能覆盖已经成功的业务结果。
			_ = runtime.client.Sleep(
				sleepContext,
			)
		})
	}

	return release, nil
}

func (client *Client) EnsureAwake(
	ctx context.Context,
) error {
	sleeping, err := client.IsSleeping(ctx)
	if err != nil {
		return err
	}

	if !sleeping {
		return nil
	}

	return client.Wake(ctx)
}

func (client *Client) IsSleeping(
	ctx context.Context,
) (bool, error) {
	response, err := client.controlRequest(
		ctx,
		http.MethodGet,
		"/is_sleeping",
		nil,
	)
	if err != nil {
		return false, err
	}

	defer response.Body.Close()

	body, err := io.ReadAll(
		io.LimitReader(response.Body, 64*1024),
	)
	if err != nil {
		return false, fmt.Errorf(
			"read VLM sleep response: %w",
			err,
		)
	}

	var sleeping bool

	if err := json.Unmarshal(body, &sleeping); err == nil {
		return sleeping, nil
	}

	var object struct {
		IsSleeping bool `json:"is_sleeping"`
		Sleeping   bool `json:"sleeping"`
	}

	if err := json.Unmarshal(body, &object); err == nil {
		return object.IsSleeping ||
			object.Sleeping,
			nil
	}

	parsed, err := strconv.ParseBool(
		string(bytes.TrimSpace(body)),
	)
	if err != nil {
		return false, fmt.Errorf(
			"decode VLM sleeping state %q",
			string(body),
		)
	}

	return parsed, nil
}

func (client *Client) Wake(
	ctx context.Context,
) error {
	response, err := client.controlRequest(
		ctx,
		http.MethodPost,
		"/wake_up",
		nil,
	)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	return nil
}

func (client *Client) Sleep(
	ctx context.Context,
) error {
	response, err := client.controlRequest(
		ctx,
		http.MethodPost,
		"/sleep?level=1",
		nil,
	)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	return nil
}

func (client *Client) controlRequest(
	ctx context.Context,
	method string,
	path string,
	body io.Reader,
) (*http.Response, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		method,
		client.baseURL+path,
		body,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create VLM control request: %w",
			err,
		)
	}

	if client.apiKey != "" {
		request.Header.Set(
			"Authorization",
			"Bearer "+client.apiKey,
		)
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf(
			"execute VLM control request: %w",
			err,
		)
	}

	if response.StatusCode < 200 ||
		response.StatusCode >= 300 {
		defer response.Body.Close()

		message, _ := io.ReadAll(
			io.LimitReader(
				response.Body,
				64*1024,
			),
		)

		return nil, fmt.Errorf(
			"VLM control %s %s returned %d: %s",
			method,
			path,
			response.StatusCode,
			string(message),
		)
	}

	return response, nil
}
