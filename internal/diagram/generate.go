// Package diagram provides functions to generate PlantUML diagrams for a RabbitMQ topology.
package diagram

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strings"

	"github.com/Patrick-Ivann/AIM-Q/internal/cli"
	"github.com/Patrick-Ivann/AIM-Q/internal/rabbitmq"
)

// Generate produces PlantUML source code visualizing the given RabbitMQ topology
// based on CLI options (e.g., groupings, message stats, etc).
func Generate(topology *rabbitmq.Topology, opts cli.Options) string {
	var sb strings.Builder

	// Begin PlantUML document and set visual options
	sb.WriteString(fmt.Sprintf("@startuml %s\n", opts.URI))
	sb.WriteString("skinparam shadowing false\n\n")

	// Compute diagram groups (vhost or type)
	groupKey := determineGroups(topology, opts)

	// Track already-defined exchanges to avoid duplicate renderings.
	definedExchanges := make(map[string]struct{})

	for _, group := range groupKey {
		writeDiagramGroup(&sb, topology, opts, group, definedExchanges)
	}

	sb.WriteString("@enduml\n")
	return sb.String()
}

// writeDiagramGroup emits one diagram group (package block) for PlantUML rendering.
// This includes exchanges, queues, bindings, and consumers.
func writeDiagramGroup(
	sb *strings.Builder, topology *rabbitmq.Topology, opts cli.Options,
	group string, definedExchanges map[string]struct{},
) {
	sb.WriteString(fmt.Sprintf("package \"%s\" {\n", group))
	writeExchanges(sb, topology.Exchanges, opts, group, definedExchanges)
	writeQueues(sb, topology.Queues, opts, group)
	writeBindings(sb, topology.Bindings, opts, group, definedExchanges)
	writeConsumers(sb, topology.Consumers, opts, group)
	sb.WriteString("}\n")
}

// writeExchanges emits rectangle definitions for exchanges belonging to the group.
func writeExchanges(
	sb *strings.Builder, exchanges []rabbitmq.Exchange, opts cli.Options,
	group string, definedExchanges map[string]struct{},
) {
	for _, ex := range exchanges {
		if !matchesGroup(opts, ex.Vhost, ex.Type, group) {
			continue
		}
		exID := sanitize("ex_" + ex.Vhost + "_" + ex.Name)
		definedExchanges[exID] = struct{}{}
		label := fmt.Sprintf("%s exchange: %s\\n(type=%s)", icon(ex.Type), ex.Name, ex.Type)
		sb.WriteString(fmt.Sprintf("rectangle \"%s\" as %s #%s\n", label, exID, color(ex.Type)))
	}
}

// writeQueues emits rectangle definitions for queues belonging to the group.
func writeQueues(
	sb *strings.Builder, queues []rabbitmq.Queue, opts cli.Options, group string,
) {
	for _, q := range queues {
		if !matchesGroup(opts, q.Vhost, "", group) {
			continue
		}
		qID := sanitize("qu_" + q.Vhost + "_" + q.Name)
		label := fmt.Sprintf("📦 queue: %s", q.Name)

		if opts.ShowMsgStats {
			label += formatMsgStats(q)
			if msgs, ok := q.Arguments["messages"]; ok {
				label += fmt.Sprintf("\\nmsgs: %v", msgs)
			}
		}
		sb.WriteString(fmt.Sprintf("rectangle \"%s\" as %s #white\n", label, qID))
	}
}

// writeBindings emits PlantUML arrows for all queue & exchange linkages in this group.
func writeBindings(
	sb *strings.Builder, bindings []rabbitmq.Binding, opts cli.Options,
	group string, definedExchanges map[string]struct{},
) {
	for _, b := range bindings {
		if !matchesGroup(opts, b.Vhost, "", group) {
			continue
		}

		// Ensure exchange source is always rendered: special-case for default ("")
		source := b.Source
		if source == "" {
			source = "default"
		}
		src := sanitize("ex_" + b.Vhost + "_" + source)
		if _, exists := definedExchanges[src]; !exists {
			definedExchanges[src] = struct{}{}
			label := "➡️ exchange: default\\n(type=direct)"
			sb.WriteString(fmt.Sprintf("rectangle \"%s\" as %s #%s\n", label, src, vhostColor(b.Vhost)))
		}

		// Connections: source → destination (queue or exchange)
		var dst string
		if b.DestType == "queue" {
			dst = sanitize("qu_" + b.Vhost + "_" + b.Destination)
		} else {
			dst = sanitize("ex_" + b.Vhost + "_" + b.Destination)
		}

		label := ""
		if b.RoutingKey != "" {
			label = fmt.Sprintf(" : \"%s\"", escapeLabel(b.RoutingKey))
		}
		sb.WriteString(fmt.Sprintf("%s --> %s%s\n", src, dst, label))
	}
}

// writeConsumers emits PlantUML "actor" and delivery edges for consumer processes.
func writeConsumers(
	sb *strings.Builder, consumers []rabbitmq.Consumer, opts cli.Options, group string,
) {
	for _, c := range consumers {
		if !matchesGroup(opts, c.Vhost, "", group) {
			continue
		}
		qID := sanitize("qu_" + c.Vhost + "_" + c.Queue)
		conID := sanitize("cons_" + c.ConsumerTag)
		sb.WriteString(fmt.Sprintf("actor \"consumer: %s\" as %s\n", c.ConsumerTag, conID))
		sb.WriteString(fmt.Sprintf("%s --> %s : delivers\n", qID, conID))
	}
}

// icon returns an emoji prefix based on exchange type for visual clarity.
func icon(t string) string {
	switch t {
	case "direct":
		return "➡️"
	case "fanout":
		return "🔄"
	case "topic":
		return "🧩"
	case "headers":
		return "📋"
	default:
		return "❓"
	}
}

// color maps exchange type to a PlantUML color string.
func color(t string) string {
	switch t {
	case "direct":
		return "2196F3"
	case "fanout":
		return "FFEB3B"
	case "topic":
		return "4CAF50"
	case "headers":
		return "9C27B0"
	default:
		return "BBBBBB"
	}
}

// vhostColor returns a stable color for a given vhost name using a hash.
func vhostColor(vhost string) string {
	colors := []string{
		"#F44336", "#E91E63", "#9C27B0", "#3F51B5",
		"#03A9F4", "#009688", "#4CAF50", "#CDDC39",
		"#FFC107", "#FF9800", "#795548", "#607D8B",
	}
	h := fnv.New32a()
	h.Write([]byte(vhost))
	return colors[h.Sum32()%uint32(len(colors))]
}

// sanitize creates a safe PlantUML alias by replacing special chars.
func sanitize(s string) string {
	replacer := strings.NewReplacer("/", "_", "-", "_", ".", "_")
	return replacer.Replace(s)
}

// escapeLabel safely escapes routing keys and other labels for PlantUML.
func escapeLabel(s string) string {
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

// determineGroups returns a sorted group key list per options (group by vhost or type).
func determineGroups(topo *rabbitmq.Topology, opts cli.Options) []string {
	groups := make(map[string]struct{})
	if opts.GroupBy == "type" {
		for _, ex := range topo.Exchanges {
			groups[ex.Type] = struct{}{}
		}
	} else {
		for _, ex := range topo.Exchanges {
			groups[ex.Vhost] = struct{}{}
		}
		for _, q := range topo.Queues {
			groups[q.Vhost] = struct{}{}
		}
	}
	keys := make([]string, 0, len(groups))
	for g := range groups {
		keys = append(keys, g)
	}
	sort.Strings(keys)
	return keys
}

// matchesGroup tells if an exchange/queue/etc should be shown in group.
func matchesGroup(opts cli.Options, vhost, typ, group string) bool {
	if opts.GroupBy == "type" {
		return typ == group
	}
	return vhost == group
}

// formatMsgStats returns a summary string for queue message statistics.
func formatMsgStats(q rabbitmq.Queue) string {
	return fmt.Sprintf(
		"\\nmessages: %d\\nready: %d\\nunacked: %d",
		q.MessageStats.Messages,
		q.MessageStats.MessagesReady,
		q.MessageStats.MessagesUnacked,
	)
}
