// Package oktaexpr evaluates Okta Expression Language expressions, such as
// those used in Okta group rules, against a user profile and group
// memberships.
//
// This is a Go port of https://github.com/mathewmoon/okta-expression-parser.
// See the README for a list of deliberate deviations from that source.
package oktaexpr

import "github.com/stevenewson/okta-expression-parser/expressionclasses"

// Parser evaluates Okta Expression Language expressions against a user
// profile, group memberships, and a set of expression classes.
type Parser struct {
	groupIDs    []string
	userProfile map[string]any
	groupData   map[string]any
	classes     expressionclasses.Registry
}

// Option configures a Parser constructed by New.
type Option func(*Parser)

// WithGroupIDs sets the group IDs the user is considered a member of, used
// by isMemberOfGroup and isMemberOfAnyGroup.
func WithGroupIDs(groupIDs []string) Option {
	return func(p *Parser) { p.groupIDs = groupIDs }
}

// WithUserProfile sets the profile data resolved by "user" and "user.<attr>"
// path expressions.
func WithUserProfile(profile map[string]any) Option {
	return func(p *Parser) { p.userProfile = profile }
}

// WithGroupData sets group metadata keyed by group ID, used by the
// isMemberOfGroupName family of builtins. Each entry is expected to have a
// nested "profile" map with a "name" key, e.g.:
//
//	map[string]any{"00g1": map[string]any{"profile": map[string]any{"name": "Engineering"}}}
//
// If the expression class registry contains a *expressionclasses.Groups,
// its GroupData is also set to this value for use by
// Groups.getFilteredGroups — which expects a differently-shaped, flat map
// instead (see the expressionclasses.Groups doc comment).
func WithGroupData(groupData map[string]any) Option {
	return func(p *Parser) { p.groupData = groupData }
}

// WithExpressionClasses replaces the default expression class registry
// (Arrays, String, Convert, Iso3166Convert, Groups), letting callers add or
// override the classes available to CLASS.method(...) expressions.
func WithExpressionClasses(classes expressionclasses.Registry) Option {
	return func(p *Parser) { p.classes = classes }
}

// New constructs a Parser. With no options, it has no group memberships, an
// empty user profile, and the default expression classes.
func New(opts ...Option) *Parser {
	p := &Parser{
		userProfile: map[string]any{},
		classes:     expressionclasses.Default(),
	}
	for _, opt := range opts {
		opt(p)
	}
	if groups, ok := p.classes["Groups"].(*expressionclasses.Groups); ok && p.groupData != nil {
		groups.GroupData = p.groupData
	}
	return p
}

// Parse evaluates expression and returns its result, which may be a bool,
// int, float64, string, nil, Array, Tuple, or map[string]any depending on
// the expression.
//
// Unlike the Python source, which sometimes silently swallows a syntax
// error and returns None, Parse always returns a non-nil error when the
// expression could not be evaluated.
func (p *Parser) Parse(expression string) (any, error) {
	toks, err := tokenize(expression)
	if err != nil {
		return nil, err
	}
	ctx := &evalContext{
		toks:        toks,
		userProfile: p.userProfile,
		groupIDs:    p.groupIDs,
		groupData:   p.groupData,
		classes:     p.classes,
	}
	return ctx.parse()
}
