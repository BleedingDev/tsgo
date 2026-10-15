package typeparser

import (
	"slices"

	"github.com/microsoft/TypeScript/tsc/shim/ast"
	"github.com/microsoft/TypeScript/tsc/shim/checker"
)

func (tp *TypeParser) CachedObjectLiteralContextualType(node *ast.Node) *checker.Type {
	if tp == nil || tp.checker == nil || node == nil || node.Kind != ast.KindObjectLiteralExpression {
		return nil
	}
	return Cached(&tp.links.ObjectLiteralContextualType, node, func() *checker.Type {
		return tp.GetContextualType(node, checker.ContextFlagsNone)
	})
}

// CachedObjectLiteralConstraintContextualType returns the contextual type of an
// object literal inside a generic call argument, as if each type parameter of
// the call's resolved signature were its constraint. It returns nil when the
// literal is not inside a call argument or the signature has no constrained
// type parameter.
//
// The call is never resolved again. An inference-blocked contextual query
// (ContextFlagsIgnoreNodeInferences) would drop the resolved signatures of the
// enclosing calls and resolve them with the argument's inferences removed. On
// the shared checker this can instantiate types the original check never
// built, for example Match.when patterns over a wide union, and leak
// TS2589 errors and error types into the program's diagnostics.
func (tp *TypeParser) CachedObjectLiteralConstraintContextualType(node *ast.Node) *checker.Type {
	if tp == nil || tp.checker == nil || node == nil || node.Kind != ast.KindObjectLiteralExpression {
		return nil
	}
	return Cached(&tp.links.ObjectLiteralConstraintContextualType, node, func() *checker.Type {
		return tp.objectLiteralConstraintContextualType(node)
	})
}

func (tp *TypeParser) objectLiteralConstraintContextualType(node *ast.Node) *checker.Type {
	argument, call := containingCallArgument(node)
	if call == nil {
		return nil
	}
	arguments := call.Arguments()
	argumentIndex := slices.Index(arguments, argument)
	if argumentIndex < 0 || slices.ContainsFunc(arguments[:argumentIndex+1], ast.IsSpreadElement) {
		return nil
	}
	c := tp.checker
	signature := c.GetResolvedSignature(call)
	if signature == nil || signature.Target() == nil {
		return nil
	}
	target := signature.Target()
	typeParameters := target.TypeParameters()
	typeArguments := c.GetTypeArgumentsForResolvedSignature(signature)
	if len(typeParameters) == 0 || len(typeArguments) != len(typeParameters) {
		return nil
	}
	// Constraints may mention other type parameters, so instantiate them with
	// the inferences of the resolved signature.
	inferred := checker.NewTypeMapper(typeParameters, typeArguments)
	constrained := slices.Clone(typeArguments)
	hasConstraint := false
	for i, typeParameter := range typeParameters {
		if constraint := c.GetConstraintOfTypeParameter(typeParameter); constraint != nil {
			constrained[i] = checker.Checker_instantiateType(c, constraint, inferred)
			hasConstraint = true
		}
	}
	if !hasConstraint {
		return nil
	}
	parameterType := c.GetTypeAtPosition(target, argumentIndex)
	if parameterType == nil {
		return nil
	}
	argumentType := checker.Checker_instantiateType(c, parameterType, checker.NewTypeMapper(typeParameters, constrained))
	// Let the checker derive the literal's contextual type from the argument,
	// as it does during checking, with the constrained argument type in place.
	checker.Checker_pushContextualType(c, argument, argumentType, false /*isCache*/)
	defer checker.Checker_popContextualType(c)
	return tp.GetContextualType(node, checker.ContextFlagsNone)
}

// containingCallArgument returns the argument of the nearest enclosing call or
// new expression that contains node, together with that call.
func containingCallArgument(node *ast.Node) (*ast.Node, *ast.Node) {
	for current := node; current.Parent != nil; current = current.Parent {
		parent := current.Parent
		if ast.IsCallLikeExpression(parent) {
			if ast.IsCallOrNewExpression(parent) && slices.Contains(parent.Arguments(), current) {
				return current, parent
			}
			return nil, nil
		}
	}
	return nil, nil
}
