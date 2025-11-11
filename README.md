# Adapters Package

A high-performance, thread-safe Go library for struct-to-struct field adaptation with intelligent handling of `AdditionalData` fields, pluggable converters, and validation hooks.

## Features

- Direct Field Copying using name or JSON tag match
- Field Conversion via scoped converter functions (global, destination-type, (src,dst) pair)
- Validation Hooks (same scoping precedence as converters) for enforcing invariants post-conversion
- Smart `AdditionalData` marshal/unmarshal (opt-in/opt-out flags)
- Copy-on-Write atomic registries (no locks on fast path)
- Metadata caching & optional warming (zero allocations per field after warmup)
- Builder API for ergonomic configuration
- High test coverage (85%+) & concurrency tests

## Installation

```bash
go get github.com/Station-Manager/adapters
```

## Quick Start

```go
// Basic usage
adapter := adapters.New()
err := adapter.Into(&dst, &src)
```

### Generics helpers

```go
// Allocate and adapt in one step
out, err := adapter.AdaptTo[Dest](&src)
// or
var d Dest
_ = adapter.Copy(&d, &src)
```

### Batch registration

```go
adapter.Batch(func(r *adapters.RegistryBatch){
    r.GlobalConverter("Temperature", conv)
    r.GlobalValidator("Name", notEmpty)
})
```

### AdditionalData field rename

```go
type D struct {
    Extras null.JSON `adapter:"additional" json:"extras"`
}
```

### Converters

```go
adapter.RegisterConverter("Temperature", func(v any) (any, error) {
    c := v.(float64)
    return (c*9/5)+32, nil // Celsius -> Fahrenheit
})
```

### Validators

Validators run after setting a field (and after any converter). Return an error to abort adaptation.

```go
adapter.RegisterValidator("Name", func(v any) error {
    if len(v.(string)) == 0 { return errors.New("name required") }
    return nil
})
```

### Builder API

```go
ad := adapters.NewBuilder().
    WithOptions(adapters.WithCaseInsensitiveAdditionalData(true)).
    AddConverter("Name", adapters.MapString(strings.ToUpper)).
    AddValidator("Name", func(v any) error { if v.(string)=="" { return errors.New("empty") }; return nil }).
    Build()
```

### AdditionalData Controls

Options:
- `WithDisableUnmarshalAdditionalData(true)` skip source AdditionalData expansion
- `WithDisableMarshalAdditionalData(true)` skip marshaling remaining fields into destination AdditionalData
- `WithIncludeZeroValues(true)` include zero values when marshaling
- `WithCaseInsensitiveAdditionalData(true)` case-insensitive key matching
- `WithOverwritePolicy(PreferAdditionalData)` allow AdditionalData to overwrite direct fields

### JSON Tag Precedence

Field matching order:
1. Exact field name match
2. JSON tag name match
3. (If case-insensitive option on) case-insensitive variations

Adapter-specific struct tags (`adapter:"ignore"`) are minimal and only used to ignore fields. Prefer JSON tags for naming.

### Validation + Conversion Precedence

For both converters and validators: pair > destination-type > global.

### Opting Out of AdditionalData

Use the disable options (above). If disabled, no JSON marshal/unmarshal occurs.

### Metadata Warmup

```go
adapter.WarmMetadata(ExampleSrc{}, ExampleDst{})
```

Pre-builds metadata to reduce first-call latency in hot paths.

## Error Handling

Adapt returns an error if:
- src or dst nil
- Arguments not pointers to structs
- Converter returns error
- Validator returns error
- AdditionalData contains invalid JSON

## Concurrency

Registries use atomic pointer swaps with copy-on-write maps; Adapt performs only reads (no locks). Registering converters/validators is safe concurrently with adaptations.

## Performance

Fast path operations avoid reflection map lookups by cached metadata.

## License

See LICENSE.
