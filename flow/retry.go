package flow

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/flowswiss/goclient/v2/core"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Every mutating API call is retried with a bounded backoff, regardless of the
// error message. Not retried: 401/403, 404 (success on a delete, an error on everything else),
// context cancellation, and transport failures on a create.
// The budget is configurable via the provider's `retry_timeout`.

const (
	defaultRetryTimeout      = 90 * time.Second
	defaultRetryInitialDelay = 1 * time.Second
	defaultRetryMaxDelay     = 15 * time.Second
)

type retryPolicy struct {
	Timeout      time.Duration
	InitialDelay time.Duration
	MaxDelay     time.Duration
}

var defaultRetryPolicy = retryPolicy{
	Timeout:      defaultRetryTimeout,
	InitialDelay: defaultRetryInitialDelay,
	MaxDelay:     defaultRetryMaxDelay,
}

type retryOperation int

const (
	opMutate retryOperation = iota
	opCreate
	opDelete
)

func retry(ctx context.Context, what string, fn func() error) error {
	return retryWith(ctx, defaultRetryPolicy, opMutate, what, fn)
}

func retryCreate(ctx context.Context, what string, fn func() error) error {
	return retryWith(ctx, defaultRetryPolicy, opCreate, what, fn)
}

func retryDelete(ctx context.Context, what string, fn func() error) error {
	return retryWith(ctx, defaultRetryPolicy, opDelete, what, fn)
}

func retryWith(ctx context.Context, policy retryPolicy, op retryOperation, what string, fn func() error) error {
	start := time.Now()
	delay := policy.InitialDelay

	for attempt := 1; ; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		switch classifyRetry(err, op) {
		case retryTreatAsSuccess:
			tflog.Debug(ctx, "object already gone, treating as success", map[string]interface{}{
				"operation": what,
				"error":     err.Error(),
			})
			return nil
		case retryStop:
			return err
		}

		elapsed := time.Since(start)
		if elapsed+delay > policy.Timeout {
			tflog.Debug(ctx, "retry budget exhausted", map[string]interface{}{
				"operation": what,
				"attempts":  attempt,
				"elapsed":   elapsed.String(),
				"error":     err.Error(),
			})
			return err
		}

		tflog.Debug(ctx, "retrying after error", map[string]interface{}{
			"operation": what,
			"attempt":   attempt,
			"wait":      delay.String(),
			"status":    statusCode(err),
			"error":     err.Error(),
		})

		select {
		case <-time.After(jitter(delay)):
		case <-ctx.Done():
			return err
		}

		delay *= 2
		if delay > policy.MaxDelay {
			delay = policy.MaxDelay
		}
	}
}

type retryDecision int

const (
	retryStop retryDecision = iota
	retryAgain
	retryTreatAsSuccess
)

func classifyRetry(err error, op retryOperation) retryDecision {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return retryStop
	}

	var apiErr core.APIError
	if errors.As(err, &apiErr) && apiErr.Response() != nil {
		switch apiErr.Response().StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return retryStop
		case http.StatusNotFound:
			if op == opDelete {
				return retryTreatAsSuccess
			}
			return retryStop
		default:
			return retryAgain
		}
	}

	if op == opCreate && isTransportError(err) {
		return retryStop
	}

	return retryAgain
}

func isTransportError(err error) bool {
	for e := err; e != nil; e = errors.Unwrap(e) {
		if strings.HasPrefix(e.Error(), "parse response body") {
			return false
		}
	}
	return true
}

func statusCode(err error) int {
	var apiErr core.APIError
	if errors.As(err, &apiErr) && apiErr.Response() != nil {
		return apiErr.Response().StatusCode
	}
	return 0
}

func jitter(d time.Duration) time.Duration {
	return d + time.Duration(rand.Int63n(int64(d)/4+1))
}

type readRetryTransport struct {
	base http.RoundTripper
}

func (t readRetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodGet {
		return t.base.RoundTrip(req)
	}

	ctx := req.Context()
	policy := defaultRetryPolicy
	start := time.Now()
	delay := policy.InitialDelay

	for attempt := 1; ; attempt++ {
		res, err := t.base.RoundTrip(req.Clone(ctx))
		if err == nil && res.StatusCode < http.StatusInternalServerError {
			return res, nil
		}
		if err != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
			return res, err
		}

		elapsed := time.Since(start)
		if elapsed+delay > policy.Timeout {
			return res, err
		}

		reason := ""
		if err != nil {
			reason = err.Error()
		} else {
			reason = res.Status
			_, _ = io.Copy(io.Discard, res.Body)
			_ = res.Body.Close()
		}

		tflog.Debug(ctx, "retrying read after error", map[string]interface{}{
			"url":     req.URL.String(),
			"attempt": attempt,
			"wait":    delay.String(),
			"error":   reason,
		})

		select {
		case <-time.After(jitter(delay)):
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		delay *= 2
		if delay > policy.MaxDelay {
			delay = policy.MaxDelay
		}
	}
}

func parseRetryTimeout(value string) (time.Duration, error) {
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q: %w", value, err)
	}
	if d < 0 {
		return 0, fmt.Errorf("retry timeout must not be negative, got %s", d)
	}
	return d, nil
}
