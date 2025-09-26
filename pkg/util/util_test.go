package util

import (
	"reflect"
	"sort"
	"testing"
)

func TestDiffTools_NoChange(t *testing.T) {
	old := []string{"a", "b", "c"}
	newT := []string{"a", "b", "c"}
	added, removed := DiffTools(old, newT)
	if len(added) != 0 || len(removed) != 0 {
		t.Errorf("Expected no changes, got added=%v removed=%v", added, removed)
	}
}

func TestDiffTools_OnlyAdded(t *testing.T) {
	old := []string{"a", "b"}
	newT := []string{"a", "b", "c", "d"}
	added, removed := DiffTools(old, newT)
	expectedAdded := []string{"c", "d"}

	// ensure order does not affect the test outcome
	sort.Strings(added)
	sort.Strings(expectedAdded)

	if !reflect.DeepEqual(added, expectedAdded) || len(removed) != 0 {
		t.Errorf("Expected added=%v, removed=[], got added=%v removed=%v", expectedAdded, added, removed)
	}
}

func TestDiffTools_OnlyRemoved(t *testing.T) {
	old := []string{"a", "b", "c"}
	newT := []string{"a"}
	added, removed := DiffTools(old, newT)
	expectedRemoved := []string{"b", "c"}

	// ensure order does not affect the test outcome
	sort.Strings(removed)
	sort.Strings(expectedRemoved)

	if !reflect.DeepEqual(removed, expectedRemoved) || len(added) != 0 {
		t.Errorf("Expected added=[], removed=%v, got added=%v removed=%v", expectedRemoved, added, removed)
	}
}

func TestDiffTools_AddedAndRemoved(t *testing.T) {
	old := []string{"a", "b", "c"}
	newT := []string{"b", "d", "e"}
	added, removed := DiffTools(old, newT)
	expectedAdded := []string{"d", "e"}
	expectedRemoved := []string{"a", "c"}

	// ensure order does not affect the test outcome
	sort.Strings(added)
	sort.Strings(expectedAdded)
	sort.Strings(removed)
	sort.Strings(expectedRemoved)

	if !reflect.DeepEqual(added, expectedAdded) || !reflect.DeepEqual(removed, expectedRemoved) {
		t.Errorf("Expected added=%v, removed=%v, got added=%v removed=%v", expectedAdded, expectedRemoved, added, removed)
	}
}

func TestDiffTools_EmptyOld(t *testing.T) {
	var old []string
	newT := []string{"x", "y"}
	added, removed := DiffTools(old, newT)
	expectedAdded := []string{"x", "y"}

	// ensure order does not affect the test outcome
	sort.Strings(added)
	sort.Strings(expectedAdded)

	if !reflect.DeepEqual(added, expectedAdded) || len(removed) != 0 {
		t.Errorf("Expected added=%v, removed=[], got added=%v removed=%v", expectedAdded, added, removed)
	}
}

func TestDiffTools_EmptyNew(t *testing.T) {
	old := []string{"x", "y"}
	var newT []string
	added, removed := DiffTools(old, newT)
	expectedRemoved := []string{"x", "y"}

	// ensure order does not affect the test outcome
	sort.Strings(removed)
	sort.Strings(expectedRemoved)

	if !reflect.DeepEqual(removed, expectedRemoved) || len(added) != 0 {
		t.Errorf("Expected added=[], removed=%v, got added=%v removed=%v", expectedRemoved, added, removed)
	}
}

func TestDiffTools_BothEmpty(t *testing.T) {
	var old []string
	var newT []string
	added, removed := DiffTools(old, newT)
	if len(added) != 0 || len(removed) != 0 {
		t.Errorf("Expected no changes, got added=%v removed=%v", added, removed)
	}
}

func TestDiffTools_Duplicates(t *testing.T) {
	old := []string{"a", "a", "b"}
	newT := []string{"a", "b", "b", "c"}
	added, removed := DiffTools(old, newT)
	expectedAdded := []string{"c"}
	var expectedRemoved []string

	// ensure order does not affect the test outcome
	sort.Strings(added)
	sort.Strings(expectedAdded)
	sort.Strings(removed)
	sort.Strings(expectedRemoved)

	if !reflect.DeepEqual(added, expectedAdded) || !reflect.DeepEqual(removed, expectedRemoved) {
		t.Errorf("Expected added=%v, removed=%v, got added=%v removed=%v", expectedAdded, expectedRemoved, added, removed)
	}
}

func TestDiffTools_OrderDoesNotMatter(t *testing.T) {
	old := []string{"a", "b", "c"}
	newT := []string{"c", "b", "a"}
	added, removed := DiffTools(old, newT)
	if len(added) != 0 || len(removed) != 0 {
		t.Errorf("Expected no changes, got added=%v removed=%v", added, removed)
	}
}
