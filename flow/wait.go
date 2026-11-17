package flow

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/flowswiss/goclient/common"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// waiting for real state: the api marks an order as processed while the
// resource is still coming up (a server keeps booting for seconds to minutes),
// so callers poll the actual status with a deadline

// TODO: these deadlines are fixed defaults — per-resource `timeouts {}`
// overrides come with terraform-plugin-framework-timeouts once the framework
// is on 1.x; the waits then read their deadline from the resource instead of the constant
const (
	defaultWaitInterval = 3 * time.Second
	serverBootTimeout   = 10 * time.Minute
	clusterWaitTimeout  = 20 * time.Minute
	volumeSettleTimeout = 5 * time.Minute
	// a load balancer create stays working for about a minute, pool and member
	// changes for seconds
	loadBalancerTimeout = 10 * time.Minute
	// snapshot create and volume restore copy the data and scale with its size
	snapshotTimeout = 30 * time.Minute
	// an order is processed within seconds — a stuck order worker would
	// otherwise keep the apply hanging until ctrl-c
	orderTimeout = 10 * time.Minute
	// the longest synchronous call is the volume detach, which the backend
	// holds for up to 30 seconds
	responseHeaderTimeout = 2 * time.Minute
)

// the sdk polls an order until it succeeds, fails or the context ends — this
// bounds it and names the order in the error
func waitForOrder(ctx context.Context, service common.OrderService, ordering common.Ordering) (common.Order, error) {
	ctx, cancel := context.WithTimeout(ctx, orderTimeout)
	defer cancel()

	order, err := service.WaitUntilProcessed(ctx, ordering)
	if err == nil {
		return order, nil
	}

	id, _ := ordering.ExtractIdentifier()
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return order, fmt.Errorf("timeout after %s waiting for order %d to be processed", orderTimeout, id)
	case errors.Is(err, common.ErrOrderFailed):
		if order.Status.Name != "" {
			return order, fmt.Errorf("order %d failed (%s)", id, order.Status.Name)
		}
		return order, fmt.Errorf("order %d failed", id)
	}
	return order, err
}

// waitFor polls check until it reports done, the deadline passes or the
// context is cancelled — an error from check does not abort the wait, it only
// surfaces in the timeout error if it never went away
func waitFor(ctx context.Context, timeout, interval time.Duration, name string, check func(ctx context.Context) (bool, error)) error {
	deadline := time.Now().Add(timeout)
	var lastErr error

	for attempt := 1; ; attempt++ {
		done, err := check(ctx)
		if err == nil && done {
			return nil
		}
		if err != nil {
			lastErr = err
		}

		if time.Now().Add(interval).After(deadline) {
			if lastErr != nil {
				return fmt.Errorf("timeout after %s waiting for %s (last error: %w)", timeout, name, lastErr)
			}
			return fmt.Errorf("timeout after %s waiting for %s", timeout, name)
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

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
