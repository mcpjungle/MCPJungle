package util

// DiffTools detects which tools have been added and removed between two slices containing tool names.
func DiffTools(oldTools, newTools []string) (added, removed []string) {
	oldSet := make(map[string]struct{}, len(oldTools))
	newSet := make(map[string]struct{}, len(newTools))

	for _, t := range oldTools {
		oldSet[t] = struct{}{}
	}
	for _, t := range newTools {
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
