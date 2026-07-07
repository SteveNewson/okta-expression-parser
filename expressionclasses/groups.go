package expressionclasses

import (
	"fmt"
	"strings"

	"github.com/stevenewson/okta-expression-parser/values"
)

// Groups implements the Okta Expression Language "Groups" class.
//
// GroupData maps a group ID to that group's data, e.g.:
//
//	map[string]any{"00g1": map[string]any{"name": "Engineering"}}
//
// Note that getFilteredGroups looks keys up directly on each group's data
// (group_data[id][key]), unlike the isMemberOfGroupName family of builtins,
// which look under a nested "profile" key. That inconsistency exists in the
// source library too; it's preserved here rather than "fixed" since the two
// features are independent and callers may reasonably shape group data
// differently for each.
type Groups struct {
	GroupData map[string]any
}

func (g *Groups) Call(method string, args ...any) (any, error) {
	switch method {
	case "getFilteredGroups":
		return g.getFilteredGroups(args)
	default:
		return nil, fmt.Errorf("Groups has no method %q", method)
	}
}

// getFilteredGroups returns the values of groupKey (the last segment of
// groupExpression, e.g. "name" from "user.name") for every group in
// allowList that exists in GroupData, up to limit results (default 5),
// dropping any that resolve to nil.
func (g *Groups) getFilteredGroups(args []any) (any, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, argCountError("Groups", "getFilteredGroups", "2 or 3", len(args))
	}
	allowList, ok := args[0].(values.Array)
	if !ok {
		return nil, fmt.Errorf("Groups.getFilteredGroups: allow_list must be an array, got %s", values.TypeName(args[0]))
	}
	groupExpression, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("Groups.getFilteredGroups: group_expression must be a string, got %s", values.TypeName(args[1]))
	}
	limit := 5
	if len(args) == 3 {
		l, err := toInt(args[2])
		if err != nil {
			return nil, fmt.Errorf("Groups.getFilteredGroups: %w", err)
		}
		limit = l
	}

	parts := strings.Split(groupExpression, ".")
	groupKey := parts[len(parts)-1]

	res := make([]any, 0, len(allowList))
	for _, idAny := range allowList {
		id, ok := idAny.(string)
		if !ok {
			res = append(res, nil)
			continue
		}
		var val any
		if group, ok := g.GroupData[id].(map[string]any); ok {
			val = group[groupKey]
		}
		res = append(res, val)
	}

	if len(res) > limit {
		res = res[:limit]
	}

	filtered := make(values.Array, 0, len(res))
	for _, v := range res {
		if v != nil {
			filtered = append(filtered, v)
		}
	}
	return filtered, nil
}
