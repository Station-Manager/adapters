package adapters

import (
	"fmt"
	"github.com/goccy/go-json"
)

// TypeConstraint defines the set of allowed high-level types for conversion.
// Keep this broad to avoid tight coupling across modules.
// We intentionally use 'any' here to allow adapters to be reused for various types
// without requiring those types to be declared in this module at compile time.
// Strong typing is still achieved by the generic parameter at call sites.
// See tests and call sites for expected usage.
// Note: Callers are responsible for ensuring compatible JSON field tags/structure.
//
//nolint:revive // exported constraint kept for compatibility
type TypeConstraint interface{ any }

// ModelConstraint defines the set of allowed DB model types for conversion.
// Similarly broad to avoid coupling to specific generated model types.
//
//nolint:revive // exported constraint kept for compatibility
type ModelConstraint interface{ any }

// convert serializes the input to JSON and deserializes it into the target output.
// This is a lossy mapping if source and destination do not have compatible JSON structures.
// Prefer explicit mappers for complex/nested types.
func convert[Input any, Output any](input Input, output *Output) error {
	data, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("adapters: marshal failed: %w", err)
	}
	if err = json.Unmarshal(data, output); err != nil {
		return fmt.Errorf("adapters: unmarshal failed: %w", err)
	}
	return nil
}

// ConvertModelToType converts a model into a strongly typed struct of the specified Type.
// It uses a JSON round-trip; for nested or non-aligned structures, consider writing an explicit mapper instead.
func ConvertModelToType[Type TypeConstraint](model interface{}) (*Type, error) {
	var result Type
	if err := convert(model, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ConvertTypeToModel converts a strongly typed struct into a model of the specified Model type.
// It uses a JSON round-trip; for nested or non-aligned structures, consider writing an explicit mapper instead.
func ConvertTypeToModel[Model ModelConstraint](typ interface{}) (*Model, error) {
	var result Model
	if err := convert(typ, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
