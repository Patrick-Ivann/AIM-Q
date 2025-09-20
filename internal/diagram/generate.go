package diagram

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strings"

	"github.com/Patrick-Ivann/AIM-Q/internal/cli"
	"github.com/Patrick-Ivann/AIM-Q/internal/rabbitmq"
)

// Generate returns the PlantUML code for the given topology and CLI options
func Generate(topology *rabbitmq.Topology, opts cli.Options) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("@startuml %s\n", opts.URI))
	sb.WriteString("skinparam shadowing false\n\n")

	groupKey := determineGroups(topology, opts)

	// Track defined exchanges (to prevent double-definitions)
	definedExchanges := make(map[string]struct{})

	for _, group := range groupKey {
		sb.WriteString(fmt.Sprintf("package \"%s\" {\n", group))

		// Exchanges
		for _, ex := range topology.Exchanges {
			if !matchesGroup(opts, ex.Vhost, ex.Type, group) {
				continue
			}
			exID := sanitize("ex_" + ex.Vhost + "_" + ex.Name)
			definedExchanges[exID] = struct{}{}
			label := fmt.Sprintf("%s exchange: %s\\n(type=%s)", icon(ex.Type), ex.Name, ex.Type)
			sb.WriteString(fmt.Sprintf("rectangle \"%s\" as %s #%s\n", label, exID, color(ex.Type)))
		}

		// Queues
		for _, q := range topology.Queues {
			if !matchesGroup(opts, q.Vhost, "", group) {
				continue
			}
			qID := sanitize("qu_" + q.Vhost + "_" + q.Name)
			label := fmt.Sprintf("📦 queue: %s", q.Name)
			if opts.ShowMsgStats {
				label += fmt.Sprintf("\\nmessages: %d", q.MessageStats.Messages)
				label += fmt.Sprintf("\\nready: %d", q.MessageStats.MessagesReady)
				label += fmt.Sprintf("\\nunacked: %d", q.MessageStats.MessagesUnacked)
				if msgs, ok := q.Arguments["messages"]; ok {
					label += fmt.Sprintf("\\nmsgs: %v", msgs)
				}
			}
			sb.WriteString(fmt.Sprintf("rectangle \"%s\" as %s #white\n", label, qID))
		}

		// Bindings
		for _, b := range topology.Bindings {
			if !matchesGroup(opts, b.Vhost, "", group) {
				continue
			}

			// Fix: support unnamed (default) exchange
			source := b.Source
			if source == "" {
				source = "default"
			}
			src := sanitize("ex_" + b.Vhost + "_" + source)

			// Ensure default exchange is rendered if missing
			if _, exists := definedExchanges[src]; !exists {
				definedExchanges[src] = struct{}{}
				label := "➡️ exchange: default\\n(type=direct)"
				sb.WriteString(fmt.Sprintf("rectangle \"%s\" as %s #%s\n", label, src, vhostColor(b.Vhost)))
			}

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

		// Consumers
		for _, c := range topology.Consumers {
			if !matchesGroup(opts, c.Vhost, "", group) {
				continue
			}
			qID := sanitize("qu_" + c.Vhost + "_" + c.Queue)
			conID := sanitize("cons_" + c.ConsumerTag)
			sb.WriteString(fmt.Sprintf("actor \"consumer: %s\" as %s\n", c.ConsumerTag, conID))
			sb.WriteString(fmt.Sprintf("%s --> %s : delivers\n", qID, conID))
		}

		sb.WriteString("}\n")
	}

	sb.WriteString("@enduml\n")
	return sb.String()
}
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

// Assign consistent color per vhost using hashing
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

func sanitize(s string) string {
	replacer := strings.NewReplacer("/", "_", "-", "_", ".", "_")
	return replacer.Replace(s)
}

func escapeLabel(s string) string {
	// Escape quotes and colons
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

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

func matchesGroup(opts cli.Options, vhost, typ, group string) bool {
	if opts.GroupBy == "type" {
		return typ == group
	}
	return vhost == group
}
