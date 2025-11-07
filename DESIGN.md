1. Copy exported struct fields if they are the same name and type
2. Copy and convert exported struct fields if they have the same name but also have a converter function registered for that field (registration is based on field name only)
3. If the destination struct has had all its exported fields assigned a value (using point 1 above), and also has field `AdditionalData` on type `null.JSON` from `github.com/aarondl/null/v8`, then the remaining fields from the source struct are marshaled to JSON and stored in the `AdditionalData` field.
4. If the source struct has an exported field with the name `AdditionalData` and the type is `null.JSON` from `github.com/aarondl/null/v8`, then it should be it should be unmarshalled (JSON) and the data used to populate the destination struct. However, exported fields fulfilling the criteria in point 1 above should not be overwritten, and data convertion is point 2 above should also be applied.
5. If the source struct has an exported field with the name `AdditionalData` and the type is not `null.JSON` from `github.com/aarondl/null/v8`, then it should be ignored.
6. The code should be built for speed, thread safety and memory safety.
7. The code should be tested to 85%+
8. The code should be documented, with examples.
9. The code should be optimized for performance, with benchmarks and profiling used to identify and fix bottlenecks.

Example structs to be copied are:

    `database/postgres/models/Qso` <-> `types/Qso`
    `database/sqlite/models/Qso` <-> `types/Qso`
