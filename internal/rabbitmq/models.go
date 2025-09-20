package rabbitmq

import "github.com/Patrick-Ivann/AIM-Q/internal/cli"

// Exchange describes a RabbitMQ exchange.
type Exchange struct {
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Vhost      string         `json:"vhost"`
	Durable    bool           `json:"durable"`
	AutoDelete bool           `json:"auto_delete"`
	Arguments  map[string]any `json:"arguments"`
}

// Queue describes a RabbitMQ queue.
type Queue struct {
	Name         string         `json:"name"`
	Vhost        string         `json:"vhost"`
	Durable      bool           `json:"durable"`
	AutoDelete   bool           `json:"auto_delete"`
	Arguments    map[string]any `json:"arguments"`
	MessageStats struct {
		Messages        int `json:"messages"`
		MessagesReady   int `json:"messages_ready"`
		MessagesUnacked int `json:"messages_unacknowledged"`
	} `json:"message_stats"`
}

// Binding connects an exchange to a queue or another exchange.
type Binding struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	DestType    string `json:"destination_type"`
	Vhost       string `json:"vhost"`
	RoutingKey  string `json:"routing_key"`
}

// Consumer represents a queue consumer.
type Consumer struct {
	Queue         string `json:"queue"`
	ConsumerTag   string `json:"consumer_tag"`
	Vhost         string `json:"vhost"`
	ChannelDetail struct {
		PID int `json:"pid"`
	} `json:"channel_details"`
}

// Topology contains the full RabbitMQ configuration snapshot.
type Topology struct {
	Exchanges []Exchange
	Queues    []Queue
	Bindings  []Binding
	Consumers []Consumer
}

// Filter applies command-line filters to the topology.
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
