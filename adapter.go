package adapters

import (
	"fmt"
	"github.com/goccy/go-json"
	"reflect"
	"sync"

	"github.com/aarondl/null/v8"
)

// ConverterFunc is a function that converts a source field value to a destination field value.
// It is registered by field name and applies to any source/destination struct pair.
type ConverterFunc func(src interface{}) (interface{}, error)

// Adapter manages field conversions and performs struct-to-struct adaptation
// with special handling for AdditionalData fields of type null.JSON from github.com/aarondl/null/v8.
type Adapter struct {
	mu         sync.RWMutex
	converters map[string]ConverterFunc // Maps field name -> converter function
}

// New creates a new Adapter instance
func New() *Adapter {
	return &Adapter{
		converters: make(map[string]ConverterFunc),
	}
}

// RegisterConverter registers a converter function for a specific field name.
// The converter will be applied to any field with this name during adaptation.
func (a *Adapter) RegisterConverter(fieldName string, fn ConverterFunc) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.converters[fieldName] = fn
}

// Adapt copies and converts fields from src to dst according to the adaptation rules:
// 1. Copy fields with same name and type directly
// 2. Copy and convert fields with same name using registered converter
// 3. Marshal remaining source fields to dst.AdditionalData (null.JSON) if present
// 4. Unmarshal src.AdditionalData (null.JSON) to populate dst fields
//
// Both src and dst must be pointers to structs.
func (a *Adapter) Adapt(src, dst interface{}) error {
	if src == nil || dst == nil {
		return fmt.Errorf("src and dst must not be nil")
	}

	srcVal := reflect.ValueOf(src)
	dstVal := reflect.ValueOf(dst)

	if srcVal.Kind() != reflect.Ptr || dstVal.Kind() != reflect.Ptr {
		return fmt.Errorf("src and dst must be pointers")
	}

	srcVal = srcVal.Elem()
	dstVal = dstVal.Elem()

	if srcVal.Kind() != reflect.Struct || dstVal.Kind() != reflect.Struct {
		return fmt.Errorf("src and dst must point to structs")
	}

	return a.adaptStruct(dstVal, srcVal)
}

// adaptStruct performs the struct-to-struct adaptation
func (a *Adapter) adaptStruct(dstVal, srcVal reflect.Value) error {
	dstType := dstVal.Type()
	srcType := srcVal.Type()

	// Track which source fields have been processed
	processedSrcFields := make(map[string]bool)

	// Track which destination fields have been set
	dstFieldsSet := make(map[string]bool)

	// Step 1 & 2: Copy fields with same name (with or without conversion)
	for i := 0; i < dstType.NumField(); i++ {
		dstField := dstVal.Field(i)
		dstFieldType := dstType.Field(i)

		// Skip unexported fields
		if !dstField.CanSet() {
			continue
		}

		// Skip AdditionalData field in first pass
		if dstFieldType.Name == "AdditionalData" {
			continue
		}

		// Find matching source field by name
		srcField := srcVal.FieldByName(dstFieldType.Name)
		if !srcField.IsValid() {
			continue
		}

		srcFieldType, found := srcType.FieldByName(dstFieldType.Name)
		if !found {
			continue
		}

		// Skip source AdditionalData if it's not null.JSON
		if srcFieldType.Name == "AdditionalData" && srcFieldType.Type != reflect.TypeOf(null.JSON{}) {
			processedSrcFields[srcFieldType.Name] = true
			continue
		}

		// Try to copy/convert the field
		if err := a.adaptField(dstField, srcField, dstFieldType.Name); err != nil {
			return fmt.Errorf("adapting field %s: %w", dstFieldType.Name, err)
		}

		processedSrcFields[srcFieldType.Name] = true
		dstFieldsSet[dstFieldType.Name] = true
	}

	// Step 4: Unmarshal src.AdditionalData (null.JSON) to populate dst fields
	srcAdditionalData := srcVal.FieldByName("AdditionalData")
	if srcAdditionalData.IsValid() {
		srcAdditionalDataType, found := srcType.FieldByName("AdditionalData")
		if found && srcAdditionalDataType.Type == reflect.TypeOf(null.JSON{}) {
			if err := a.unmarshalAdditionalData(dstVal, dstType, srcAdditionalData, dstFieldsSet); err != nil {
				return fmt.Errorf("unmarshaling AdditionalData: %w", err)
			}
			processedSrcFields["AdditionalData"] = true
		}
	}

	// Step 3: Marshal remaining source fields to dst.AdditionalData (null.JSON)
	dstAdditionalData := dstVal.FieldByName("AdditionalData")
	if dstAdditionalData.IsValid() && dstAdditionalData.CanSet() {
		dstAdditionalDataType, found := dstType.FieldByName("AdditionalData")
		if found && dstAdditionalDataType.Type == reflect.TypeOf(null.JSON{}) {
			// Marshal any remaining unprocessed source fields to AdditionalData
			if err := a.marshalRemainingFields(dstAdditionalData, srcVal, srcType, processedSrcFields); err != nil {
				return fmt.Errorf("marshaling remaining fields to AdditionalData: %w", err)
			}
		}
	}

	return nil
}

// adaptField copies or converts a single field value
func (a *Adapter) adaptField(dstField, srcField reflect.Value, fieldName string) error {
	srcType := srcField.Type()
	dstType := dstField.Type()

	// Check if a converter is registered for this field name
	a.mu.RLock()
	converter, exists := a.converters[fieldName]
	a.mu.RUnlock()

	if exists {
		// Use registered converter
		converted, err := converter(srcField.Interface())
		if err != nil {
			return err
		}
		convertedVal := reflect.ValueOf(converted)
		if !convertedVal.Type().AssignableTo(dstType) {
			return fmt.Errorf("converter returned type %s, expected %s", convertedVal.Type(), dstType)
		}
		dstField.Set(convertedVal)
		return nil
	}

	// If types are identical, direct assignment
	if srcType == dstType {
		dstField.Set(srcField)
		return nil
	}

	// If types are assignable, assign directly
	if srcType.AssignableTo(dstType) {
		dstField.Set(srcField)
		return nil
	}

	// If types are convertible, convert and assign
	if srcType.ConvertibleTo(dstType) {
		dstField.Set(srcField.Convert(dstType))
		return nil
	}

	// Cannot copy this field - skip it silently
	return nil
}

// unmarshalAdditionalData unmarshals src.AdditionalData to populate dst fields
func (a *Adapter) unmarshalAdditionalData(dstVal reflect.Value, dstType reflect.Type, srcAdditionalData reflect.Value, dstFieldsSet map[string]bool) error {
	// Get the null.JSON value
	nullJSON, ok := srcAdditionalData.Interface().(null.JSON)
	if !ok || !nullJSON.Valid {
		return nil // No data to unmarshal
	}

	// Get the JSON data
	jsonData := nullJSON.JSON

	// Unmarshal to a map
	var additionalFields map[string]json.RawMessage
	if err := json.Unmarshal(jsonData, &additionalFields); err != nil {
		return err
	}

	// Populate destination fields from AdditionalData
	for fieldName, rawValue := range additionalFields {
		// Skip if field was already set (rule: don't overwrite)
		if dstFieldsSet[fieldName] {
			continue
		}

		dstField := dstVal.FieldByName(fieldName)
		if !dstField.IsValid() || !dstField.CanSet() {
			continue
		}

		// Check if converter exists for this field
		a.mu.RLock()
		converter, exists := a.converters[fieldName]
		a.mu.RUnlock()

		if exists {
			// Unmarshal to interface{} first to preserve JSON types (e.g. float64 for numbers)
			var rawVal interface{}
			if err := json.Unmarshal(rawValue, &rawVal); err != nil {
				continue // Skip fields that can't be unmarshaled
			}

			// Apply converter
			converted, err := converter(rawVal)
			if err != nil {
				continue // Skip on conversion error
			}
			convertedVal := reflect.ValueOf(converted)
			if convertedVal.Type().AssignableTo(dstField.Type()) {
				dstField.Set(convertedVal)
				dstFieldsSet[fieldName] = true
			}
		} else {
			// Create a pointer to the field's type for unmarshaling
			fieldPtr := reflect.New(dstField.Type())
			if err := json.Unmarshal(rawValue, fieldPtr.Interface()); err != nil {
				continue // Skip fields that can't be unmarshaled
			}
			// Direct assignment
			dstField.Set(fieldPtr.Elem())
			dstFieldsSet[fieldName] = true
		}
	}

	return nil
}

// marshalRemainingFields marshals unprocessed source fields to dst.AdditionalData
func (a *Adapter) marshalRemainingFields(dstAdditionalData reflect.Value, srcVal reflect.Value, srcType reflect.Type, processedSrcFields map[string]bool) error {
	remainingFields := make(map[string]interface{})

	// Collect all fields including those from embedded structs
	a.collectRemainingFields(srcVal, srcType, processedSrcFields, remainingFields)

	// If there are no remaining fields, set AdditionalData to null
	if len(remainingFields) == 0 {
		dstAdditionalData.Set(reflect.ValueOf(null.JSON{}))
		return nil
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(remainingFields)
	if err != nil {
		return err
	}

	// Set the null.JSON field using JSONFrom
	nullJSON := null.JSONFrom(jsonData)
	dstAdditionalData.Set(reflect.ValueOf(nullJSON))

	return nil
}

// collectRemainingFields recursively collects fields from structs including embedded structs
func (a *Adapter) collectRemainingFields(srcVal reflect.Value, srcType reflect.Type, processedSrcFields map[string]bool, remainingFields map[string]interface{}) {
	for i := 0; i < srcType.NumField(); i++ {
		srcFieldType := srcType.Field(i)
		srcField := srcVal.Field(i)

		// Skip unexported fields
		if !srcField.CanInterface() {
			continue
		}

		// Skip AdditionalData field
		if srcFieldType.Name == "AdditionalData" {
			continue
		}

		// If this is an embedded struct, recurse into it
		if srcFieldType.Anonymous && srcField.Kind() == reflect.Struct {
			a.collectRemainingFields(srcField, srcField.Type(), processedSrcFields, remainingFields)
			continue
		}

		// Skip if already processed
		if processedSrcFields[srcFieldType.Name] {
			continue
		}

		remainingFields[srcFieldType.Name] = srcField.Interface()
	}
}
