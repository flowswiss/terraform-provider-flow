package flow

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/flowswiss/goclient/v2/common"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// waiting for real state: the api marks an order as processed while the
// resource is still coming up, so callers poll the actual status with a deadline
const (
	defaultWaitInterval   = 3 * time.Second
	serverBootTimeout     = 10 * time.Minute
	clusterWaitTimeout    = 20 * time.Minute
	volumeSettleTimeout   = 5 * time.Minute
	loadBalancerTimeout   = 10 * time.Minute
	snapshotTimeout       = 30 * time.Minute
	orderTimeout          = 10 * time.Minute
	responseHeaderTimeout = 2 * time.Minute
	goneTimeout           = 10 * time.Minute
)

type timeoutGetter func(context.Context, time.Duration) (time.Duration, diag.Diagnostics)

func withTimeout(ctx context.Context, get timeoutGetter, diagnostics *diag.Diagnostics) (context.Context, context.CancelFunc) {
	timeout, diags := get(ctx, 0)
	diagnostics.Append(diags...)

	if timeout <= 0 {
		return ctx, func() {}
	}

	tflog.Debug(ctx, "applying configured timeout", map[string]interface{}{
		"timeout": timeout.String(),
	})
	return context.WithTimeout(ctx, timeout)
}

func timeoutDescription(what string) string {
	return what + `; a string that can be ` +
		`[parsed as a duration](https://pkg.go.dev/time#ParseDuration), such as "30s" or "2h45m"`
}

func remaining(ctx context.Context, fallback time.Duration) time.Duration {
	if deadline, ok := ctx.Deadline(); ok {
		return time.Until(deadline)
	}
	return fallback
}

func waitForOrder(ctx context.Context, service *common.OrderService, ordering common.Ordering) (common.Order, error) {
	start := time.Now()

	ctx, cancel := context.WithTimeout(ctx, remaining(ctx, orderTimeout))
	defer cancel()

	order, err := service.WaitUntilProcessed(ctx, ordering)
	if err == nil {
		return order, nil
	}

	id, _ := ordering.ExtractIdentifier()
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return order, fmt.Errorf("timeout after %s waiting for order %d to be processed", time.Since(start).Round(time.Second), id)
	case errors.Is(err, common.ErrOrderFailed):
		if order.Status.Name != "" {
			return order, fmt.Errorf("order %d failed (%s)", id, order.Status.Name)
		}
		return order, fmt.Errorf("order %d failed", id)
	}
	return order, err
}

type terminalError struct{ err error }

func (t terminalError) Error() string { return t.err.Error() }
func (t terminalError) Unwrap() error { return t.err }

func stopWaiting(err error) error {
	return terminalError{err: err}
}

func waitFor(ctx context.Context, timeout, interval time.Duration, name string, check func(ctx context.Context) (bool, error)) error {
	start := time.Now()
	deadline := start.Add(remaining(ctx, timeout))
	var lastErr error

	for attempt := 1; ; attempt++ {
		done, err := check(ctx)
		if err == nil && done {
			return nil
		}
		if err != nil {
			var terminal terminalError
			if errors.As(err, &terminal) {
				return fmt.Errorf("waiting for %s: %w", name, terminal.err)
			}
			lastErr = err
		}

		if time.Now().Add(interval).After(deadline) {
			waited := time.Since(start).Round(time.Second)
			if lastErr != nil {
				return fmt.Errorf("timeout after %s waiting for %s (last error: %w)", waited, name, lastErr)
			}
			return fmt.Errorf("timeout after %s waiting for %s", waited, name)
		}

		tflog.Debug(ctx, "waiting", map[string]interface{}{
			"for":     name,
			"attempt": attempt,
			"error":   errString(err),
		})

		select {
		case <-time.After(interval):
		case <-ctx.Done():
			return fmt.Errorf("cancelled while waiting for %s: %w", name, ctx.Err())
		}
	}
}

// waitForGone polls get until the api answers 404
func waitForGone(ctx context.Context, timeout time.Duration, name string, get func(ctx context.Context) error) error {
	return waitFor(ctx, timeout, defaultWaitInterval, name+" to be gone", func(ctx context.Context) (bool, error) {
		err := get(ctx)
		if isNotFound(err) {
			return true, nil
		}
		return false, err
	})
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
