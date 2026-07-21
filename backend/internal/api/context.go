package api

import (
	"net/http"
	"time"
	"context"
)

func contextWithTimeout(
	request *http.Request,
	timeout time.Duration,
) (context.Context, context.CancelFunc) {
	return context.WithTimeout(
		request.Context(),
		timeout,
	)
}
