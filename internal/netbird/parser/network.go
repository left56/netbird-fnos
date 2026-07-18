package parser

import "strings"

// Network is the parsed form of `netbird networks list` output. Parsing lives
// here so services, not HTTP handlers, own all CLI presentation differences.
type Network struct {
	ID          string
	Name        string
	Selected    bool
	ExitNode    bool
	Overlapping bool
}

func NetworksText(output string) []Network {
	result := []Network{}
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || strings.EqualFold(fields[0], "ID") {
			continue
		}
		lower := strings.ToLower(line)
		result = append(result, Network{ID: fields[0], Name: fields[1], Selected: strings.Contains(line, "✓") || strings.Contains(lower, "selected"), ExitNode: strings.Contains(lower, "exit"), Overlapping: strings.Contains(lower, "overlap")})
	}
	return result
}
