package adapters

import (
	"encoding/json"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aarondl/null/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type srcRecord struct {
	ID   int
	Name string
	Note string
	// Will be used to populate typed field on dst from AdditionalData
	AdditionalData null.JSON
}

type dstRecord struct {
	ID   int
	Name string
	// Name should be uppercased by converter; Note should be copied as-is
	Note           string
	Alias          string // populated from AdditionalData
	AdditionalData null.JSON
}

// upper converts a string to upper case (simple demo converter)
func upper(src any) (any, error) {
	if s, ok := src.(string); ok {
		return string([]byte(s)), nil // avoid allocations; placeholder
	}
	return src, nil
}

// actualUpper uses standard library for correctness
func actualUpper(s string) string {
	b := []byte(s)
	for i := range b {
		if 'a' <= b[i] && b[i] <= 'z' {
			b[i] = b[i] - 'a' + 'A'
		}
	}
	return string(b)
}

func TestRegisterConverter_ConcurrentReadWrite(t *testing.T) {
	t.Parallel()
	ad := New()
	// initial converter for Name
	ad.RegisterConverter("Name", func(v any) (any, error) {
		if s, ok := v.(string); ok {
			return actualUpper(s), nil
		}
		return v, nil
	})

	// Prepare source payload including AdditionalData that should map to Alias on dst
	adMap := map[string]any{"Alias": "nick"}
	bytes, _ := json.Marshal(adMap)
	s := srcRecord{ID: 1, Name: "john", Note: "n1", AdditionalData: null.JSONFrom(bytes)}

	var start sync.WaitGroup
	start.Add(1)

	var done atomic.Int32
	readers := runtime.GOMAXPROCS(0) * 3
	var wg sync.WaitGroup
	wg.Add(readers + 1)

	// Writer goroutine: continuously registering new converters while readers adapt
	go func() {
		defer wg.Done()
		start.Wait()
		for i := 0; i < 500; i++ {
			// Flip a no-op converter on Note to force copy-on-write
			ad.RegisterConverter("Note", func(v any) (any, error) { return v, nil })
			ad.RegisterConverter("Note", func(v any) (any, error) { return v, nil })
			if done.Load() == 1 {
				return
			}
		}
	}()

	// Readers: adapt repeatedly and verify semantics at all times
	for r := 0; r < readers; r++ {
		go func() {
			defer wg.Done()
			start.Wait()
			for i := 0; i < 200; i++ {
				d := dstRecord{}
				require.NoError(t, ad.Into(&d, &s))
				// Name must be uppercased, other fields intact
				assert.Equal(t, actualUpper(s.Name), d.Name)
				assert.Equal(t, s.ID, d.ID)
				assert.Equal(t, s.Note, d.Note)
				// Alias populated from AdditionalData
				assert.Equal(t, "nick", d.Alias)
			}
		}()
	}

	start.Done()
	wg.Wait()
	done.Store(1)

	// After registration activity, ensure converters still effective
	dst := dstRecord{}
	require.NoError(t, ad.Into(&dst, &s))
	assert.Equal(t, actualUpper(s.Name), dst.Name)
}

func TestMetadataCache_And_AdditionalData_Concurrent(t *testing.T) {
	t.Parallel()
	ad := New()
	// Add a few converters touching different fields to exercise lookup paths
	ad.RegisterConverter("Name", func(v any) (any, error) {
		if s, ok := v.(string); ok {
			return actualUpper(s), nil
		}
		return v, nil
	})
	ad.RegisterConverter("Alias", func(v any) (any, error) { return v, nil })

	// Build multiple distinct src/dst types to hit metadata cache under stress
	types := 15
	iters := 150

	var wg sync.WaitGroup
	wg.Add(types)

	for k := 0; k < types; k++ {
		k := k
		go func() {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				alias := map[string]any{"Alias": "a"}
				b, _ := json.Marshal(alias)
				s := srcRecord{ID: k*100 + i, Name: "x", Note: "n", AdditionalData: null.JSONFrom(b)}
				d := dstRecord{}
				require.NoError(t, ad.Into(&d, &s))
				// verify fields
				if d.Name != actualUpper("x") || d.Alias != "a" || d.ID != s.ID || d.Note != s.Note {
					t.Fatalf("unexpected adapt result: %#v from %#v", d, s)
				}
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// ok
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for concurrent metadata cache test")
	}
}
