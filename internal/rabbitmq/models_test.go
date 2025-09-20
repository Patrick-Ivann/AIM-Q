package rabbitmq_test

import (
	"testing"

	"github.com/Patrick-Ivann/AIM-Q/internal/cli"
	"github.com/Patrick-Ivann/AIM-Q/internal/rabbitmq"
	"github.com/stretchr/testify/assert"
)

func TestTopology_Filter(t *testing.T) {
	testTopology := &rabbitmq.Topology{
		Exchanges: []rabbitmq.Exchange{
			{Name: "ex1", Vhost: "vh1", Type: "direct"},
			{Name: "ex2", Vhost: "vh2", Type: "fanout"},
		},
		Queues: []rabbitmq.Queue{
			{Name: "q1", Vhost: "vh1"},
			{Name: "q2", Vhost: "vh2"},
		},
		Bindings: []rabbitmq.Binding{
			{Source: "ex1", Destination: "q1", DestType: "queue", Vhost: "vh1"},
			{Source: "ex2", Destination: "q2", DestType: "queue", Vhost: "vh2"},
		},
		Consumers: []rabbitmq.Consumer{
			{Queue: "q1", Vhost: "vh1", ConsumerTag: "ctag1"},
			{Queue: "q2", Vhost: "vh2", ConsumerTag: "ctag2"},
		},
	}

	tests := map[string]struct {
		opts     cli.Options
		expected *rabbitmq.Topology
	}{
		"filter by vhost vh1": {
			opts: cli.Options{FilterVhost: "vh1"},
			expected: &rabbitmq.Topology{
				Exchanges: []rabbitmq.Exchange{
					{Name: "ex1", Vhost: "vh1", Type: "direct"},
				},
				Queues: []rabbitmq.Queue{
					{Name: "q1", Vhost: "vh1"},
				},
				Bindings: []rabbitmq.Binding{
					{Source: "ex1", Destination: "q1", DestType: "queue", Vhost: "vh1"},
				},
				Consumers: []rabbitmq.Consumer{
					{Queue: "q1", Vhost: "vh1", ConsumerTag: "ctag1"},
				},
			},
		},
		"filter by exchange ex1": {
			opts: cli.Options{FilterExchange: "ex1"},
			expected: &rabbitmq.Topology{
				Exchanges: []rabbitmq.Exchange{
					{Name: "ex1", Vhost: "vh1", Type: "direct"},
				},
				Queues:    testTopology.Queues,
				Bindings:  testTopology.Bindings,
				Consumers: testTopology.Consumers,
			},
		},
		"no filter": {
			opts:     cli.Options{},
			expected: testTopology,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			res := testTopology.Filter(tc.opts)
			assert.Equal(t, tc.expected.Exchanges, res.Exchanges)
			assert.Equal(t, tc.expected.Queues, res.Queues)
			assert.Equal(t, tc.expected.Bindings, res.Bindings)
			assert.Equal(t, tc.expected.Consumers, res.Consumers)
		})
	}
}

func TestExchangeFields(t *testing.T) {
	ex := rabbitmq.Exchange{
		Name:       "exname",
		Type:       "direct",
		Durable:    true,
		AutoDelete: false,
		Arguments:  map[string]interface{}{"arg1": "val"},
	}
	assert.Equal(t, "exname", ex.Name)
	assert.Equal(t, "direct", ex.Type)
	assert.True(t, ex.Durable)
	assert.False(t, ex.AutoDelete)
	assert.Contains(t, ex.Arguments, "arg1")
}

func TestQueueFields(t *testing.T) {
	q := rabbitmq.Queue{
		Name:       "quename",
		Vhost:      "vh1",
		Durable:    false,
		AutoDelete: true,
	}
	q.MessageStats.Messages = 10
	q.MessageStats.MessagesReady = 5
	q.MessageStats.MessagesUnacked = 3

	assert.Equal(t, "quename", q.Name)
	assert.Equal(t, "vh1", q.Vhost)
	assert.False(t, q.Durable)
	assert.True(t, q.AutoDelete)
	assert.Equal(t, 10, q.MessageStats.Messages)
	assert.Equal(t, 5, q.MessageStats.MessagesReady)
	assert.Equal(t, 3, q.MessageStats.MessagesUnacked)
}

func TestBindingFields(t *testing.T) {
	b := rabbitmq.Binding{
		Source:      "ex1",
		Destination: "queue1",
		DestType:    "queue",
		Vhost:       "vh1",
		RoutingKey:  "rk",
	}
	assert.Equal(t, "ex1", b.Source)
	assert.Equal(t, "queue1", b.Destination)
	assert.Equal(t, "queue", b.DestType)
	assert.Equal(t, "vh1", b.Vhost)
	assert.Equal(t, "rk", b.RoutingKey)
}

func TestConsumerFields(t *testing.T) {
	c := rabbitmq.Consumer{
		Queue:       "queue1",
		ConsumerTag: "tag",
		Vhost:       "vh1",
	}
	c.ChannelDetail.PID = 123
	assert.Equal(t, "queue1", c.Queue)
	assert.Equal(t, "tag", c.ConsumerTag)
	assert.Equal(t, "vh1", c.Vhost)
	assert.Equal(t, 123, c.ChannelDetail.PID)
}
