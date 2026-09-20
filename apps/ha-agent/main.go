// ha-agent runs one application process while this pod owns a Kubernetes Lease.
// This provides failover, not fencing or exactly-once delivery.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	coordination "k8s.io/client-go/kubernetes/typed/coordination/v1"
	core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

type child struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	done    chan struct{}
	stopped bool
}

func (c *child) start(ctx context.Context, args []string, onExit func()) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stopped || ctx.Err() != nil {
		return context.Canceled
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	c.cmd, c.done = cmd, make(chan struct{})
	go func() {
		err := cmd.Wait()
		log.Printf("application exited: %v", err)
		close(c.done)
		onExit()
	}()
	return nil
}

// Stop is terminal, including when called before the election start callback.
// Kill immediately on lost leadership; do not keep publishing during a grace period.
func (c *child) stop() {
	c.mu.Lock()
	c.stopped = true
	cmd, done := c.cmd, c.done
	c.mu.Unlock()
	if cmd != nil {
		_ = cmd.Process.Kill()
		<-done
	}
}

func (c *child) running() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stopped || c.cmd == nil {
		return false
	}
	select {
	case <-c.done:
		return false
	default:
		return true
	}
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	if len(args) == 0 {
		return fmt.Errorf("usage: ha-agent -- /app/sense (or /app/lense)")
	}
	for _, key := range []string{"HA_LEASE_NAME", "POD_NAMESPACE", "POD_NAME", "POD_UID", "HA_READY_URL"} {
		if os.Getenv(key) == "" {
			return fmt.Errorf("%s is required", key)
		}
	}
	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}
	config.Timeout = 3 * time.Second
	client, err := coordination.NewForConfig(config)
	if err != nil {
		return err
	}
	coreClient, err := core.NewForConfig(config)
	if err != nil {
		return err
	}
	markActive := func(ctx context.Context, active bool) error {
		// Test the UID so a recreated pod can never be changed by its predecessor.
		patch := fmt.Sprintf(`[{"op":"test","path":"/metadata/uid","value":%q},{"op":"add","path":"/metadata/labels/otio.io~1active","value":"%t"}]`, os.Getenv("POD_UID"), active)
		_, err := coreClient.Pods(os.Getenv("POD_NAMESPACE")).Patch(ctx, os.Getenv("POD_NAME"), types.JSONPatchType, []byte(patch), metav1.PatchOptions{})
		return err
	}
	if err := markActive(context.Background(), false); err != nil {
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer cancel()
	process := &child{}
	defer process.stop()
	probeClient := &http.Client{Timeout: time.Second}
	healthy := func() bool {
		if !process.running() {
			return false
		}
		res, err := probeClient.Get(os.Getenv("HA_READY_URL"))
		if err != nil {
			return false
		}
		defer res.Body.Close()
		return res.StatusCode == http.StatusOK && process.running()
	}
	var leading atomic.Bool
	mux := http.NewServeMux()
	mux.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if ctx.Err() != nil || (leading.Load() && !healthy()) {
			http.Error(w, "application unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	server := &http.Server{Addr: ":8099", Handler: mux, ReadHeaderTimeout: 3 * time.Second}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Print(err)
			cancel()
		}
	}()
	defer server.Close()
	elector, err := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock: &resourcelock.LeaseLock{
			LeaseMeta:  metav1.ObjectMeta{Name: os.Getenv("HA_LEASE_NAME"), Namespace: os.Getenv("POD_NAMESPACE")},
			Client:     client,
			LockConfig: resourcelock.ResourceLockConfig{Identity: os.Getenv("POD_UID")},
		},
		LeaseDuration: 30 * time.Second, RenewDeadline: 15 * time.Second, RetryPeriod: 3 * time.Second,
		// Let the lease expire after stopping. Early release could admit a successor
		// before the old application has actually terminated.
		ReleaseOnCancel: false,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(leaderCtx context.Context) {
				leading.Store(true)
				log.Print("leadership acquired; starting application")
				if err := process.start(leaderCtx, args, cancel); err != nil {
					log.Print(err)
					cancel()
					return
				}
				ticker := time.NewTicker(time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-leaderCtx.Done():
						return
					case <-ticker.C:
						if healthy() {
							if err := markActive(leaderCtx, true); err == nil {
								return
							} else {
								log.Printf("activate endpoint: %v", err)
							}
						}
					}
				}
			},
			OnStoppedLeading: func() {
				cancel()
				process.stop()
				if err := markActive(context.Background(), false); err != nil {
					log.Printf("clear endpoint: %v", err)
				}
			},
		},
	})
	if err != nil {
		return err
	}
	elector.Run(ctx)
	return nil
}
