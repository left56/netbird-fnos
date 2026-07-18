package parser

// ExitNodes returns the networks which the CLI reports as exit-node capable.
func ExitNodes(networks []Network) []Network {
	result := []Network{}
	for _, network := range networks {
		if network.ExitNode {
			result = append(result, network)
		}
	}
	return result
}
