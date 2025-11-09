package adapters

import (
	"fmt"
	"github.com/goccy/go-json"
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/aarondl/null/v8"
)

// ConverterFunc is a function that converts a source field value to a destination field value.
// It is registered by field name and applies to any source/destination struct pair.
type ConverterFunc func(src interface{}) (interface{}, error)

type fieldInfo struct {
	index            []int
	name             string
	typ              reflect.Type
	canSet           bool
	isAdditionalData bool
}

type structMetadata struct {
	fields              []fieldInfo
	fieldsByName        map[string]*fieldInfo
	additionalDataField *fieldInfo // Cached AdditionalData field info, nil if not present
}

// Adapter manages field conversions and performs struct-to-struct adaptation
// with special handling for AdditionalData fields of type null.JSON from github.com/aarondl/null/v8.
type Adapter struct {
	//	mu            sync.RWMutex
	//	converters    map[string]ConverterFunc // Maps field name -> converter function
	converters    atomic.Value
	metadataCache sync.Map  // map[reflect.Type]*structMetadata
	boolMapPool   sync.Pool // Pool for map[string]bool reuse
}

// New creates a new Adapter instance
func New() *Adapter {
	a := &Adapter{}
	a.converters.Store(make(map[string]ConverterFunc))
	a.boolMapPool = sync.Pool{
		New: func() interface{} {
			// Return nil so getBoolMap can distinguish between fresh and reused maps
			return (map[string]bool)(nil)
		},
	}
	return a
}

// RegisterConverter registers a converter function for a specific field name.
// The converter will be applied to any field with this name during adaptation.
func (a *Adapter) RegisterConverter(fieldName string, fn ConverterFunc) {
	// Copy-on-write pattern for thread-safe map updates
	oldMap := a.converters.Load().(map[string]ConverterFunc)
	newMap := make(map[string]ConverterFunc, len(oldMap)+1)

	// Copy existing converters
	for k, v := range oldMap {
		newMap[k] = v
	}
	newMap[fieldName] = fn

	a.converters.Store(newMap)
}

// Adapt copies and converts fields from src to dst according to the adaptation rules:
//  1. Copy fields with the same name and type directly
//  2. Copy and convert fields with the same name using a registered converter.
//  3. Marshal remaining source fields to dst.AdditionalData (null.JSON), if present.
//  4. Unmarshal src.AdditionalData (null.JSON) to populate dst fields
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

// getBoolMap retrieves a map from the pool and clears it
// desiredCapacity is used to allocate properly-sized maps on first use from the pool
func (a *Adapter) getBoolMap(desiredCapacity int) map[string]bool {
	pooledMap := a.boolMapPool.Get().(map[string]bool)

	// If this is a nil map from the pool (brand new from sync.Pool.New),
	// allocate with proper capacity
	if pooledMap == nil {
		return make(map[string]bool, desiredCapacity)
	}

	// Clear the map for reuse (fast even for large maps - just resets internal buckets)
	for k := range pooledMap {
		delete(pooledMap, k)
	}

	return pooledMap
}

// putBoolMap returns a map to the pool
func (a *Adapter) putBoolMap(m map[string]bool) {
	if m == nil {
		return
	}
	// Don't return excessively large maps to the pool (prevents memory bloat)
	if len(m) > 128 {
		return
	}
	a.boolMapPool.Put(m)
}

// getOrBuildMetadata retrieves or builds cached metadata for a struct type
func (a *Adapter) getOrBuildMetadata(typ reflect.Type) *structMetadata {
	if cached, ok := a.metadataCache.Load(typ); ok {
		return cached.(*structMetadata)
	}

	// Count total fields including embedded structs for accurate preallocation
	fieldCount := a.countFields(typ)

	meta := &structMetadata{
		fields:              make([]fieldInfo, 0, fieldCount),
		fieldsByName:        make(map[string]*fieldInfo, fieldCount),
		additionalDataField: nil,
	}

	// Build field metadata, handling embedded structs
	a.buildFieldMetadata(typ, meta, nil)

	// Build fieldsByName map AFTER all fields are added to prevent stale pointers
	// (slice may have reallocated during buildFieldMetadata, though unlikely with correct capacity)
	for i := range meta.fields {
		meta.fieldsByName[meta.fields[i].name] = &meta.fields[i]
	}

	// Cache AdditionalData field lookup if it exists
	if adField, ok := meta.fieldsByName["AdditionalData"]; ok && adField.isAdditionalData {
		meta.additionalDataField = adField
	}

	// Store and return (handle race condition gracefully)
	actual, _ := a.metadataCache.LoadOrStore(typ, meta)
	return actual.(*structMetadata)
}

// countFields recursively counts all fields including those in embedded structs
func (a *Adapter) countFields(typ reflect.Type) int {
	count := 0
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		// Handle embedded structs - recurse into them
		if field.Anonymous {
			fieldType := field.Type
			// Dereference pointer if it's a pointer-to-struct
			if fieldType.Kind() == reflect.Ptr {
				fieldType = fieldType.Elem()
			}
			if fieldType.Kind() == reflect.Struct {
				count += a.countFields(fieldType)
				continue
			}
		}

		count++
	}
	return count
}

// buildFieldMetadata recursively builds field metadata including embedded structs
func (a *Adapter) buildFieldMetadata(typ reflect.Type, meta *structMetadata, indexPrefix []int) {
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Build index path (for nested access)
		fieldIndex := append(append([]int(nil), indexPrefix...), i)

		// Handle embedded structs - recurse into them
		// Support both direct embedded structs and pointer-to-struct embedded fields
		if field.Anonymous {
			fieldType := field.Type
			// Dereference pointer if it's a pointer-to-struct
			if fieldType.Kind() == reflect.Ptr {
				fieldType = fieldType.Elem()
			}
			if fieldType.Kind() == reflect.Struct {
				a.buildFieldMetadata(fieldType, meta, fieldIndex)
				continue
			}
		}

		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		info := fieldInfo{
			index:            fieldIndex,
			name:             field.Name,
			typ:              field.Type,
			canSet:           true, // already checked it's exported
			isAdditionalData: field.Name == "AdditionalData" && field.Type == reflect.TypeOf(null.JSON{}),
		}
		meta.fields = append(meta.fields, info)
	}
}

// adaptStruct performs the struct-to-struct adaptation
func (a *Adapter) adaptStruct(dstVal, srcVal reflect.Value) error {
	dstType := dstVal.Type()
	srcType := srcVal.Type()

	// Get cached metadata
	dstMeta := a.getOrBuildMetadata(dstType)
	srcMeta := a.getOrBuildMetadata(srcType)

	// Only allocate tracking maps if we have AdditionalData fields to process
	hasAdditionalDataProcessing := srcMeta.additionalDataField != nil || dstMeta.additionalDataField != nil
	var processedSrcFields map[string]bool
	var dstFieldsSet map[string]bool

	if hasAdditionalDataProcessing {
		// Get maps from pool for reuse, sized based on actual field counts
		// Use the larger of src/dst field counts as capacity hint
		mapCapacity := len(srcMeta.fields)
		if len(dstMeta.fields) > mapCapacity {
			mapCapacity = len(dstMeta.fields)
		}

		processedSrcFields = a.getBoolMap(mapCapacity)
		dstFieldsSet = a.getBoolMap(mapCapacity)

		// Ensure cleanup happens even on error
		defer func() {
			a.putBoolMap(processedSrcFields)
			a.putBoolMap(dstFieldsSet)
		}()
	}

	// Step 1 & 2: Copy fields with the same name (with or without conversion)
	for i := range dstMeta.fields {
		dstFieldInfo := &dstMeta.fields[i]

		// Skip unexported or AdditionalData fields
		if !dstFieldInfo.canSet || dstFieldInfo.isAdditionalData {
			continue
		}

		// Find matching source field using cached metadata
		srcFieldInfo, found := srcMeta.fieldsByName[dstFieldInfo.name]
		if !found {
			continue
		}

		// Skip source AdditionalData if not null.JSON
		if srcFieldInfo.isAdditionalData {
			if hasAdditionalDataProcessing {
				processedSrcFields[srcFieldInfo.name] = true
			}
			continue
		}

		// Get field values by index (faster than FieldByName)
		dstField := dstVal.FieldByIndex(dstFieldInfo.index)
		srcField := srcVal.FieldByIndex(srcFieldInfo.index)

		// Try to copy/convert the field
		if err := a.adaptField(dstField, srcField, dstFieldInfo.name); err != nil {
			return fmt.Errorf("adapting field %s: %w", dstFieldInfo.name, err)
		}

		if hasAdditionalDataProcessing {
			processedSrcFields[srcFieldInfo.name] = true
			dstFieldsSet[dstFieldInfo.name] = true
		}
	}

	// Step 4: Unmarshal src.AdditionalData (null.JSON) to populate dst fields
	if srcMeta.additionalDataField != nil {
		srcAdditionalData := srcVal.FieldByIndex(srcMeta.additionalDataField.index)
		if err := a.unmarshalAdditionalData(dstVal, dstMeta, srcAdditionalData, dstFieldsSet); err != nil {
			return fmt.Errorf("unmarshaling AdditionalData: %w", err)
		}
		if hasAdditionalDataProcessing {
			processedSrcFields["AdditionalData"] = true
		}
	}

	// Step 3: Marshal remaining source fields to dst.AdditionalData (null.JSON)
	if dstMeta.additionalDataField != nil {
		dstAdditionalData := dstVal.FieldByIndex(dstMeta.additionalDataField.index)
		// Marshal any remaining unprocessed source fields to AdditionalData
		if err := a.marshalRemainingFields(dstAdditionalData, srcVal, srcType, processedSrcFields); err != nil {
			return fmt.Errorf("marshaling remaining fields to AdditionalData: %w", err)
		}
	}

	return nil
}

// adaptField copies or converts a single field value
func (a *Adapter) adaptField(dstField, srcField reflect.Value, fieldName string) error {
	// Check if destination field can be set
	if !dstField.CanSet() {
		return fmt.Errorf("cannot set field %s (unexported or unsettable)", fieldName)
	}

	srcType := srcField.Type()
	dstType := dstField.Type()

	// Load converter map once (lock-free read) - CHANGED
	converters := a.converters.Load().(map[string]ConverterFunc)
	converter, exists := converters[fieldName]

	if exists {
		// Use registered converter
		converted, err := converter(srcField.Interface())
		if err != nil {
			return err
		}
		// Handle nil converter result
		if converted == nil {
			// Set zero value for destination type
			dstField.Set(reflect.Zero(dstType))
			return nil
		}
		convertedVal := reflect.ValueOf(converted)
		if !convertedVal.IsValid() {
			return fmt.Errorf("converter returned invalid value for field %s", fieldName)
		}
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
func (a *Adapter) unmarshalAdditionalData(dstVal reflect.Value, dstMeta *structMetadata, srcAdditionalData reflect.Value, dstFieldsSet map[string]bool) error {
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

	// Load converter map once (lock-free read) - CHANGED
	converters := a.converters.Load().(map[string]ConverterFunc)

	// Populate destination fields from AdditionalData
	for fieldName, rawValue := range additionalFields {
		// Skip if field was already set (rule: don't overwrite)
		if dstFieldsSet[fieldName] {
			continue
		}

		// Look up field in metadata (fast map lookup)
		dstFieldInfo, found := dstMeta.fieldsByName[fieldName]
		if !found || !dstFieldInfo.canSet {
			continue // Field doesn't exist or isn't settable
		}

		// Use cached index for fast field access
		dstField := dstVal.FieldByIndex(dstFieldInfo.index)

		// Check if converter exists for this field - CHANGED
		converter, exists := converters[fieldName]

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
			// Handle nil converter result
			if converted == nil {
				dstField.Set(reflect.Zero(dstField.Type()))
				dstFieldsSet[fieldName] = true
				continue
			}
			convertedVal := reflect.ValueOf(converted)
			if !convertedVal.IsValid() {
				continue // Skip invalid values
			}
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
		// Support both direct embedded structs and pointer-to-struct embedded fields
		if srcFieldType.Anonymous {
			fieldVal := srcField
			fieldType := srcField.Type()
			// Dereference pointer if it's a pointer-to-struct
			if fieldVal.Kind() == reflect.Ptr {
				if fieldVal.IsNil() {
					continue // Skip nil pointers
				}
				fieldVal = fieldVal.Elem()
				fieldType = fieldVal.Type()
			}
			if fieldVal.Kind() == reflect.Struct {
				a.collectRemainingFields(fieldVal, fieldType, processedSrcFields, remainingFields)
				continue
			}
		}

		// Skip if already processed
		if processedSrcFields[srcFieldType.Name] {
			continue
		}

		remainingFields[srcFieldType.Name] = srcField.Interface()
	}
}
