# Adapters Package

A high-performance, thread-safe Go library for struct-to-struct field adaptation with intelligent handling of `AdditionalData` fields.

## Features

- **Direct Field Copying**: Copy fields with matching names and types
- **Field Conversion**: Register custom converters for field-specific transformations
- **Smart AdditionalData Handling**: Automatically marshal/unmarshal extra fields to/from `null.JSON` fields
- **Thread-Safe**: Concurrent-safe converter registration and adaptation
- **High Performance**: Optimized for speed with minimal allocations
- **Well Tested**: 90%+ code coverage with comprehensive tests and benchmarks

## Installation

```bash
go get github.com/Station-Manager/adapters
```

## Quick Start

### Basic Field Copying

```go
package main

import (
    "fmt"
    "github.com/Station-Manager/adapters"
)

type Source struct {
    Name  string
    Age   int
    Email string
}

type Dest struct {
    Name  string
    Age   int
    Email string
}

func main() {
    adapter := adapters.New()

    src := &Source{
        Name:  "John Doe",
        Age:   30,
        Email: "john@example.com",
    }

    dst := &Dest{}

    err := adapter.Adapt(dst, src)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Name: %s, Age: %d, Email: %s\n", dst.Name, dst.Age, dst.Email)
}
```

### Field Conversion

Register custom converters for specific fields:

```go
package main

import (
    "fmt"
    "github.com/Station-Manager/adapters"
)

type Source struct {
    Temperature float64 // Celsius
}

type Dest struct {
    Temperature float64 // Fahrenheit
}

func main() {
    adapter := adapters.New()

    // Register converter for Temperature field
    adapter.RegisterConverter("Temperature", func(src interface{}) (interface{}, error) {
        celsius, ok := src.(float64)
        if !ok {
            return nil, fmt.Errorf("expected float64")
        }
        fahrenheit := (celsius * 9 / 5) + 32
        return fahrenheit, nil
    })

    src := &Source{Temperature: 25.0}
    dst := &Dest{}

    err := adapter.Adapt(dst, src)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Temperature: %.2f°F\n", dst.Temperature) // Output: 77.00°F
}
```

### Marshal to AdditionalData

When all destination fields are mapped and there are extra source fields, they're automatically marshaled to `AdditionalData`:

```go
package main

import (
    "encoding/json"
    "fmt"
    "github.com/Station-Manager/adapters"
    "github.com/aarondl/null/v8"
)

type FullSource struct {
    Name   string
    Age    int
    Email  string
    Phone  string
    City   string
    Active bool
}

type CompactDest struct {
    Name            string
    Age             int
    AdditionalData null.JSON
}

func main() {
    adapter := adapters.New()

    src := &FullSource{
        Name:   "Jane Doe",
        Age:    25,
        Email:  "jane@example.com",
        Phone:  "555-1234",
        City:   "Boston",
        Active: true,
    }

    dst := &CompactDest{}

    err := adapter.Adapt(dst, src)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Name: %s, Age: %d\n", dst.Name, dst.Age)

    // Access AdditionalData
    var extras map[string]interface{}
    json.Unmarshal(dst.AdditionalData.JSON, &extras)
    fmt.Printf("Email: %s, Phone: %s, City: %s, Active: %v\n",
        extras["Email"], extras["Phone"], extras["City"], extras["Active"])
}
```

### Unmarshal from AdditionalData

Fields stored in `AdditionalData` can be automatically unpacked to destination struct fields:

```go
package main

import (
    "encoding/json"
    "fmt"
    "github.com/Station-Manager/adapters"
    "github.com/aarondl/null/v8"
)

type CompactSource struct {
    Name            string
    Age             int
    AdditionalData null.JSON
}

type ExpandedDest struct {
    Name   string
    Age    int
    Email  string
    Phone  string
    City   string
    Active bool
}

func main() {
    adapter := adapters.New()

    // Prepare source with AdditionalData
    extraFields := map[string]interface{}{
        "Email":  "bob@example.com",
        "Phone":  "555-9876",
        "City":   "Chicago",
        "Active": true,
    }
    jsonData, _ := json.Marshal(extraFields)

    src := &CompactSource{
        Name:            "Bob Smith",
        Age:             40,
        AdditionalData: null.JSONFrom(jsonData),
    }

    dst := &ExpandedDest{}

    err := adapter.Adapt(dst, src)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Name: %s, Age: %d\n", dst.Name, dst.Age)
    fmt.Printf("Email: %s, Phone: %s, City: %s, Active: %v\n",
        dst.Email, dst.Phone, dst.City, dst.Active)
}
```

### Round-Trip Adaptation

Seamlessly convert between expanded and compact representations:

```go
package main

import (
    "fmt"
    "github.com/Station-Manager/adapters"
    "github.com/aarondl/null/v8"
)

type FullData struct {
    Name   string
    Age    int
    Email  string
    Phone  string
    City   string
}

type CompactData struct {
    Name            string
    Age             int
    AdditionalData null.JSON
}

func main() {
    adapter := adapters.New()

    // Original full data
    original := &FullData{
        Name:  "Alice",
        Age:   35,
        Email: "alice@example.com",
        Phone: "555-0000",
        City:  "Seattle",
    }

    // Compact it
    compact := &CompactData{}
    adapter.Adapt(compact, original)

    // Expand it back
    expanded := &FullData{}
    adapter.Adapt(expanded, compact)

    fmt.Printf("Original == Expanded: %v\n", *original == *expanded) // true
}
```

## Adaptation Rules

The adapter follows these rules in order:

1. **Direct Copy**: Fields with the same name and compatible types are copied directly
2. **Converter**: If a converter is registered for a field name, it's applied during copying
3. **Marshal to AdditionalData**: If all destination fields (except `AdditionalData`) are set AND the destination has an `AdditionalData` field of type `null.JSON`, remaining source fields are marshaled to JSON
4. **Unmarshal from AdditionalData**: If the source has an `AdditionalData` field of type `null.JSON`, its contents are unmarshaled to populate destination fields (without overwriting already-set fields)
5. **Ignore non-JSON AdditionalData**: Source fields named `AdditionalData` that aren't `null.JSON` are ignored

## Field Precedence

When both direct fields and `AdditionalData` contain the same field name:
- **Direct fields take precedence** over data in `AdditionalData`
- Fields populated from `AdditionalData` won't overwrite already-set fields
- Converters are applied to both direct fields and fields from `AdditionalData`

## Performance

The adapter is optimized for performance:

```bash
# Run benchmarks
cd adapters
go test -bench=. -benchmem

# Typical results (YMMV):
BenchmarkAdapter_BasicFieldCopy-8              1000000    1200 ns/op    256 B/op    8 allocs/op
BenchmarkAdapter_WithConverter-8                500000    2500 ns/op    384 B/op   12 allocs/op
BenchmarkAdapter_MarshalToAdditionalData-8      200000    8000 ns/op   1024 B/op   18 allocs/op
BenchmarkAdapter_UnmarshalFromAdditionalData-8  150000   10000 ns/op   1536 B/op   24 allocs/op
BenchmarkAdapter_Concurrent-8                  2000000     800 ns/op    256 B/op    8 allocs/op
```

## Thread Safety

The `Adapter` type is safe for concurrent use:
- Converter registration uses a `sync.RWMutex`
- Multiple goroutines can call `Adapt()` simultaneously
- Converter lookups are optimized with read locks

## Testing

Run the test suite:

```bash
cd adapters
go test -v -cover

# Expected: 90%+ coverage
```

## Error Handling

The adapter returns errors for:
- Nil source or destination
- Non-pointer arguments
- Non-struct arguments
- Converter function errors
- Invalid JSON in `AdditionalData`

```go
err := adapter.Adapt(dst, src)
if err != nil {
    log.Printf("Adaptation failed: %v", err)
    // Handle error appropriately
}
```

## Best Practices

1. **Reuse Adapter Instances**: Create one `Adapter` and reuse it for better performance
2. **Register Converters Once**: Register all converters during initialization
3. **Validate Inputs**: Check that source and destination are appropriate types
4. **Handle Errors**: Always check the error return value
5. **Use Pointers**: Both source and destination must be pointers to structs
6. **Document Converters**: Clearly document expected input/output types for converters

## Examples

See the `adapter_test.go` file for comprehensive examples covering:
- Basic field copying
- Type conversion
- Converter registration
- AdditionalData marshaling/unmarshaling
- Field precedence rules
- Error handling
- Concurrent usage
- Round-trip adaptation

## License

See LICENSE file for details.

## Contributing

Contributions are welcome! Please ensure:
- Tests pass: `go test -v`
- Coverage remains high: `go test -cover`
- Benchmarks don't regress: `go test -bench=.`
- Code is formatted: `go fmt`
