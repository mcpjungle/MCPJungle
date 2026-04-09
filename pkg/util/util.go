package util

// DiffItems detects which mcp items have been added and removed between two slices containing tool names.
func DiffItems(oldItems, newItems []string) (added, removed []string) {
	oldSet := make(map[string]struct{}, len(oldItems))
	newSet := make(map[string]struct{}, len(newItems))

	for _, t := range oldItems {
		oldSet[t] = struct{}{}
	}
	for _, t := range newItems {
		newSet[t] = struct{}{}
	}

	// Find removed tools
	for t := range oldSet {
		if _, exists := newSet[t]; !exists {
			removed = append(removed, t)
		}
	}
	// Find added tools
	for t := range newSet {
		if _, exists := oldSet[t]; !exists {
			added = append(added, t)
		}
	}
	return
}
