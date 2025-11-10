// Package adapters provides struct-to-struct adaptation with field conversion and AdditionalData handling.
//
// The Adapter type manages field conversions and performs struct-to-struct adaptation
// with special handling for AdditionalData fields of type null.JSON.
//
// # Basic Usage
//
// Create an adapter and use it to copy fields between structs:
//
//	adapter := adapters.New()
//	err := adapter.Adapt(sourceStruct, destStruct)
//
// # Adaptation Rules
//
// The Adapt method follows these rules in order:
//  1. Copy fields with the same name and type directly
//  2. Copy and convert fields with the same name using registered converters
//  3. Marshal remaining source fields to dst.AdditionalData (null.JSON), if present
//  4. Unmarshal src.AdditionalData (null.JSON) to populate dst fields
//
// # Field Converters
//
// Register custom converters for specific field names:
//
//	adapter.RegisterConverter("Temperature", func(src interface{}) (interface{}, error) {
//	    temp := src.(float64)
//	    return int(temp), nil
//	})
//
// # Ignoring Fields
//
// Fields can be excluded from adaptation using struct tags:
//
//	type User struct {
//	    Name     string
//	    Password string `adapter:"ignore"`  // Not copied or marshaled
//	    Email    string
//	    Token    string `adapter:"-"`       // Alternative syntax
//	}
//
// Ignored fields are:
//   - Not copied between source and destination structs
//   - Not marshaled to AdditionalData
//   - Not unmarshaled from AdditionalData
//
// # AdditionalData
//
// The AdditionalData field (type null.JSON or boilertypes.JSON) has special handling:
//   - Fields present in source but not in destination are marshaled to dst.AdditionalData
//   - Fields in src.AdditionalData that match dst field names are unmarshaled to dst
//   - Direct field copying takes precedence over AdditionalData unmarshaling
//
// # Embedded Structs
//
// Embedded struct fields (including pointer-to-struct) are flattened and treated
// as if they were defined directly in the parent struct.
//
// # Thread Safety
//
// The Adapter is safe for concurrent use. Multiple goroutines can call Adapt
// and RegisterConverter simultaneously.
package adapters
