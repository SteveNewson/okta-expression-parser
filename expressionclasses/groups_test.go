package expressionclasses_test

import (
	"testing"

	"github.com/stevenewson/okta-expression-parser/expressionclasses"
	"github.com/stevenewson/okta-expression-parser/values"
)

func TestGroups_GetFilteredGroups(t *testing.T) {
	t.Parallel()

	// Given: getFilteredGroups looks keys up directly on each group's data
	// (a flat map), unlike the isMemberOfGroupName family of builtins.
	g := &expressionclasses.Groups{
		GroupData: map[string]any{
			"00g1": map[string]any{"name": "Engineering"},
			"00g2": map[string]any{"name": "Sales"},
		},
	}

	// When
	got, err := g.Call("getFilteredGroups", values.Array{"00g1", "00g2", "00g3"}, "user.name")

	// Then
	if err != nil {
		t.Fatalf("Groups.getFilteredGroups: unexpected error %v", err)
	}
	want := values.Array{"Engineering", "Sales"}
	if !values.EqualOperands(got, want) {
		t.Errorf("Groups.getFilteredGroups: got %#v, want %#v", got, want)
	}
}

func TestGroups_GetFilteredGroups_UnknownGroupsAreDropped(t *testing.T) {
	t.Parallel()

	// Given: no configured group data at all.
	g := &expressionclasses.Groups{}

	// When
	got, err := g.Call("getFilteredGroups", values.Array{"00g1", "00g2"}, "user.name")

	// Then
	if err != nil {
		t.Fatalf("Groups.getFilteredGroups: unexpected error %v", err)
	}
	if !values.EqualOperands(got, values.Array{}) {
		t.Errorf("Groups.getFilteredGroups: got %#v, want an empty array", got)
	}
}

func TestGroups_GetFilteredGroups_RespectsLimit(t *testing.T) {
	t.Parallel()

	// Given
	g := &expressionclasses.Groups{
		GroupData: map[string]any{
			"g1": map[string]any{"name": "A"},
			"g2": map[string]any{"name": "B"},
			"g3": map[string]any{"name": "C"},
		},
	}

	// When
	got, err := g.Call("getFilteredGroups", values.Array{"g1", "g2", "g3"}, "user.name", 2)

	// Then
	if err != nil {
		t.Fatalf("Groups.getFilteredGroups: unexpected error %v", err)
	}
	gotArr, ok := got.(values.Array)
	if !ok || len(gotArr) != 2 {
		t.Errorf("Groups.getFilteredGroups with limit 2: got %#v, want 2 elements", got)
	}
}
