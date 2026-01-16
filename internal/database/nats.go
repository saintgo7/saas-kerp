package database

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/saintgo7/saas-kerp/internal/config"
)

// NewNATSConnection creates a new NATS connection
func NewNATSConnection(cfg *config.NATSConfig) (*nats.Conn, error) {
	opts := []nats.Option{
		nats.Name("kerp-api"),
		nats.Timeout(5 * time.Second),
		nats.PingInterval(20 * time.Second),
		nats.MaxPingsOutstanding(5),
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(-1), // Unlimited reconnects
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				fmt.Printf("NATS disconnected: %v\n", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			fmt.Printf("NATS reconnected to %s\n", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			fmt.Println("NATS connection closed")
		}),
	}

	nc, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return nc, nil
}

// NewJetStream creates a JetStream context from a NATS connection
func NewJetStream(nc *nats.Conn) (nats.JetStreamContext, error) {
	js, err := nc.JetStream(
		nats.PublishAsyncMaxPending(256),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	return js, nil
}

// EnsureStream creates a stream if it doesn't exist
func EnsureStream(js nats.JetStreamContext, cfg *nats.StreamConfig) (*nats.StreamInfo, error) {
	stream, err := js.StreamInfo(cfg.Name)
	if err == nil {
		return stream, nil
	}

	if err == nats.ErrStreamNotFound {
		return js.AddStream(cfg)
	}

	return nil, fmt.Errorf("failed to get stream info: %w", err)
}

// CloseNATS closes the NATS connection
func CloseNATS(nc *nats.Conn) {
	if nc != nil {
		nc.Drain()
		nc.Close()
	}
}

// StreamConfigs returns predefined stream configurations for K-ERP
func StreamConfigs() map[string]*nats.StreamConfig {
	return map[string]*nats.StreamConfig{
		"KERP_JOBS": {
			Name:        "KERP_JOBS",
			Description: "Background job processing",
			Subjects:    []string{"jobs.>"},
			Retention:   nats.WorkQueuePolicy,
			MaxAge:      24 * time.Hour,
			Storage:     nats.FileStorage,
			Replicas:    1,
		},
		"KERP_EVENTS": {
			Name:        "KERP_EVENTS",
			Description: "Domain events",
			Subjects:    []string{"events.>"},
			Retention:   nats.LimitsPolicy,
			MaxAge:      7 * 24 * time.Hour,
			Storage:     nats.FileStorage,
			Replicas:    1,
		},
		"KERP_NOTIFICATIONS": {
			Name:        "KERP_NOTIFICATIONS",
			Description: "User notifications",
			Subjects:    []string{"notifications.>"},
			Retention:   nats.LimitsPolicy,
			MaxAge:      72 * time.Hour,
			Storage:     nats.FileStorage,
			Replicas:    1,
		},
	}
}
