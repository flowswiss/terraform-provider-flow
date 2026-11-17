package flow

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/flowswiss/goclient/v2/compute"
	"github.com/flowswiss/goclient/v2/core"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func timeoutsValue(t *testing.T, create string) timeouts.Value {
	t.Helper()

	attrTypes := map[string]attr.Type{"create": types.StringType}
	if create == "" {
		return timeouts.Value{Object: types.ObjectNull(attrTypes)}
	}

	object, diags := types.ObjectValue(attrTypes, map[string]attr.Value{"create": types.StringValue(create)})
	if diags.HasError() {
		t.Fatalf("building the timeouts value: %v", diags)
	}
	return timeouts.Value{Object: object}
}

func TestWithTimeout(t *testing.T) {
	tests := []struct {
		name      string
		configure string
		want      time.Duration
		wantError bool
	}{
		{name: "unset leaves the context alone", configure: ""},
		{name: "configured sets the deadline", configure: "45m", want: 45 * time.Minute},
		{name: "unparseable is reported", configure: "soon", wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var diagnostics diag.Diagnostics

			ctx, cancel := withTimeout(context.Background(), timeoutsValue(t, test.configure).Create, &diagnostics)
			defer cancel()

			if got := diagnostics.HasError(); got != test.wantError {
				t.Fatalf("diagnostics error = %t, want %t (%v)", got, test.wantError, diagnostics)
			}

			deadline, ok := ctx.Deadline()
			if test.want == 0 {
				if ok {
					t.Fatalf("context carries a deadline of %s, want none", time.Until(deadline))
				}
				return
			}
			if !ok {
				t.Fatal("context carries no deadline")
			}
			if left := time.Until(deadline); left > test.want || left < test.want-time.Minute {
				t.Errorf("deadline in %s, want about %s", left, test.want)
			}
		})
	}
}

func TestRemaining(t *testing.T) {
	if got := remaining(context.Background(), time.Minute); got != time.Minute {
		t.Errorf("without a deadline: %s, want the fallback %s", got, time.Minute)
	}

	for _, budget := range []time.Duration{30 * time.Second, 10 * time.Minute} {
		ctx, cancel := context.WithTimeout(context.Background(), budget)
		got := remaining(ctx, time.Minute)
		cancel()

		if got > budget || got < budget-time.Second {
			t.Errorf("with a %s budget: %s, want about %s", budget, got, budget)
		}
	}
}

func TestWaitForStopsAtTheContextDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := waitFor(ctx, time.Hour, 10*time.Millisecond, "the sky to fall", func(ctx context.Context) (bool, error) {
		return false, nil
	})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("wait succeeded, want a timeout")
	}
	if elapsed > time.Second {
		t.Errorf("waited %s, want the 50ms budget", elapsed)
	}
	if !strings.Contains(err.Error(), "waiting for the sky to fall") {
		t.Errorf("error %q does not name what it waited for", err)
	}
}

func TestWaitForReportsTheLastError(t *testing.T) {
	err := waitFor(context.Background(), 20*time.Millisecond, 10*time.Millisecond, "a volume to settle", func(ctx context.Context) (bool, error) {
		return false, errStub
	})

	if err == nil {
		t.Fatal("wait succeeded, want a timeout")
	}
	if !strings.Contains(err.Error(), errStub.Error()) {
		t.Errorf("error %q drops the last error from the check", err)
	}
}

func TestWaitForSucceeds(t *testing.T) {
	calls := 0
	err := waitFor(context.Background(), time.Minute, time.Millisecond, "a server to boot", func(ctx context.Context) (bool, error) {
		calls++
		return calls == 3, nil
	})

	if err != nil {
		t.Fatalf("wait failed: %s", err)
	}
	if calls != 3 {
		t.Errorf("checked %d times, want 3", calls)
	}
}

func TestWaitForStopsOnATerminalError(t *testing.T) {
	calls := 0
	start := time.Now()
	err := waitFor(context.Background(), 2*time.Second, 10*time.Millisecond, "a pool to be deleted", func(ctx context.Context) (bool, error) {
		calls++
		return false, stopWaiting(errStub)
	})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("wait succeeded, want the terminal error")
	}
	if calls != 1 {
		t.Errorf("checked %d times, want 1 — the wait kept polling after a terminal error", calls)
	}
	if elapsed > time.Second {
		t.Errorf("waited %s, want an immediate return", elapsed)
	}
	if !errors.Is(err, errStub) {
		t.Errorf("error %q does not carry the terminal error", err)
	}
	if strings.Contains(err.Error(), "timeout after") {
		t.Errorf("error %q reports a timeout, so the wait ran its budget instead of stopping", err)
	}
	if !strings.Contains(err.Error(), "waiting for a pool to be deleted") {
		t.Errorf("error %q does not name what it waited for", err)
	}
}

var errStub = stubError("the volume is still working")

type stubError string

func (e stubError) Error() string { return string(e) }

func TestWaitForGone(t *testing.T) {
	ctx := context.Background()

	t.Run("already gone returns on the first poll", func(t *testing.T) {
		var calls int32
		client := fakeLoadBalancerAPI(t, func(writer http.ResponseWriter) {
			atomic.AddInt32(&calls, 1)
			notFound(writer)
		})

		start := time.Now()
		if err := waitForGone(ctx, goneTimeout, "load balancer 1", func(ctx context.Context) error {
			_, err := client.Compute.LoadBalancer.Get(ctx, compute.LoadBalancerGetReq{ID: 1})
			return err
		}); err != nil {
			t.Fatalf("waitForGone returned %s", err)
		}

		if got := atomic.LoadInt32(&calls); got != 1 {
			t.Errorf("polled %d times, want 1", got)
		}
		if waited := time.Since(start); waited > defaultWaitInterval {
			t.Errorf("waited %s before returning, want no sleep at all", waited.Round(time.Millisecond))
		}
	})

	t.Run("still there keeps polling until the api answers 404", func(t *testing.T) {
		var calls int32
		client := fakeLoadBalancerAPI(t, func(writer http.ResponseWriter) {
			if atomic.AddInt32(&calls, 1) == 1 {
				writer.Header().Set("Content-Type", "application/json")
				_, _ = writer.Write([]byte(`{"id":1,"name":"tearing-down"}`))
				return
			}
			notFound(writer)
		})

		if err := waitForGone(ctx, goneTimeout, "load balancer 1", func(ctx context.Context) error {
			_, err := client.Compute.LoadBalancer.Get(ctx, compute.LoadBalancerGetReq{ID: 1})
			return err
		}); err != nil {
			t.Fatalf("waitForGone returned %s", err)
		}

		if got := atomic.LoadInt32(&calls); got != 2 {
			t.Errorf("polled %d times, want 2", got)
		}
	})

	t.Run("never gone times out and names the resource", func(t *testing.T) {
		client := fakeLoadBalancerAPI(t, func(writer http.ResponseWriter) {
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"id":1,"name":"still-here"}`))
		})

		err := waitForGone(ctx, 100*time.Millisecond, "load balancer 1", func(ctx context.Context) error {
			_, err := client.Compute.LoadBalancer.Get(ctx, compute.LoadBalancerGetReq{ID: 1})
			return err
		})
		if err == nil {
			t.Fatal("waitForGone returned no error although the load balancer never went away")
		}
		if want := "load balancer 1 to be gone"; !strings.Contains(err.Error(), want) {
			t.Errorf("error is %q, want it to name %q", err, want)
		}
	})
}

func fakeLoadBalancerAPI(t *testing.T, handle func(writer http.ResponseWriter)) flowClient {
	t.Helper()

	api := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		handle(writer)
	}))
	t.Cleanup(api.Close)

	base, err := url.Parse(api.URL)
	if err != nil {
		t.Fatalf("parsing the test server url: %s", err)
	}

	return newFlowClient(core.ClientOpts{BaseURL: base, HTTPClient: &http.Client{}, Token: "test"})
}

func notFound(writer http.ResponseWriter) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusNotFound)
	_, _ = writer.Write([]byte(`{"error":{"message":{"en":"not found"}}}`))
}
