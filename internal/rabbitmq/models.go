// Package rabbitmq defines types representing RabbitMQ objects retrieved
//
// via the management API, as well as helper methods for topology filtering.
package rabbitmq

import "github.com/Patrick-Ivann/AIM-Q/internal/cli"

// Exchange describes a RabbitMQ exchange configuration.
//
// An Exchange routes messages through configured bindings to Queues or other Exchanges.
// See RabbitMQ docs for type details: direct, fanout, topic, headers, etc.
//
// Fields correspond to API JSON response fields.
type Exchange struct {
	Name       string         `json:"name"`        // Exchange name
	Type       string         `json:"type"`        // Exchange type (direct, fanout, topic, headers)
	Vhost      string         `json:"vhost"`       // Virtual host the exchange belongs to
	Durable    bool           `json:"durable"`     // True if the exchange survives broker restart
	AutoDelete bool           `json:"auto_delete"` // True if the exchange is auto-deleted when unused
	Arguments  map[string]any `json:"arguments"`   // Additional arguments or policies
}

// Queue describes a RabbitMQ queue configuration.
//
// A Queue stores and forwards messages to consumers.
//
// MessageStats contains runtime counters reflecting the queue state.
type Queue struct {
	Name       string         `json:"name"`        // Queue name
	Vhost      string         `json:"vhost"`       // Virtual host the queue belongs to
	Durable    bool           `json:"durable"`     // True if the queue survives broker restart
	AutoDelete bool           `json:"auto_delete"` // True if the queue is auto-deleted when unused
	Arguments  map[string]any `json:"arguments"`   // Additional arguments or policies

	MessageStats struct {
		Messages        int `json:"messages"`                // Total messages in the queue
		MessagesReady   int `json:"messages_ready"`          // Messages ready for delivery to consumers
		MessagesUnacked int `json:"messages_unacknowledged"` // Messages delivered but unacknowledged
	} `json:"message_stats"`
}

// Binding represents a relationship connecting an exchange to a queue or another exchange.
//
// The Binding routes messages sent to the source exchange to the destination target,
// optionally filtered by a routing key.
type Binding struct {
	Source      string `json:"source"`           // Name of the source exchange
	Destination string `json:"destination"`      // Name of the destination (queue or exchange)
	DestType    string `json:"destination_type"` // "queue" or "exchange"
	Vhost       string `json:"vhost"`            // Virtual host where the binding lives
	RoutingKey  string `json:"routing_key"`      // Key used to filter/routing messages
}

// Consumer represents a consumer subscribed to a queue.
//
// Contains consumer tag and the PID of the channel consuming from the queue.
type Consumer struct {
	Queue         string `json:"queue"`        // Queue name the consumer listens on
	ConsumerTag   string `json:"consumer_tag"` // Consumer tag identifier
	Vhost         string `json:"vhost"`        // Virtual host of the consumer
	ChannelDetail struct {
		PID int `json:"pid"` // Process ID of the AMQP channel consuming messages
	} `json:"channel_details"`
}

// Topology represents the full snapshot of RabbitMQ server configuration.
//
// Aggregates all Exchanges, Queues, Bindings, and Consumers from the management API,
// usually obtained by Client.FetchTopology.
type Topology struct {
	Exchanges []Exchange
	Queues    []Queue
	Bindings  []Binding
	Consumers []Consumer
}

// Filter applies CLI options filtering to the topology.
//
// Currently filters by virtual host and exchange name.
// Returns a new Topology pointer containing only matching resources.
func (t *Topology) Filter(opts cli.Options) *Topology {
	filtered := &Topology{}

	for _, ex := range t.Exchanges {
		if opts.FilterVhost != "" && ex.Vhost != opts.FilterVhost {
			continue
		}
		if opts.FilterExchange != "" && ex.Name != opts.FilterExchange {
			continue
		}
		filtered.Exchanges = append(filtered.Exchanges, ex)
	}

	for _, q := range t.Queues {
		if opts.FilterVhost != "" && q.Vhost != opts.FilterVhost {
			continue
		}
		filtered.Queues = append(filtered.Queues, q)
	}

	for _, b := range t.Bindings {
		if opts.FilterVhost != "" && b.Vhost != opts.FilterVhost {
			continue
		}
		filtered.Bindings = append(filtered.Bindings, b)
	}

	for _, c := range t.Consumers {
		if opts.FilterVhost != "" && c.Vhost != opts.FilterVhost {
			continue
		}
		filtered.Consumers = append(filtered.Consumers, c)
	}

	return filtered
}
