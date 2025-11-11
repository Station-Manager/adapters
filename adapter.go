package adapters

import (
	"fmt"
	"github.com/goccy/go-json"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/aarondl/null/v8"
	boilertypes "github.com/aarondl/sqlboiler/v4/types"
)

// ConverterFunc is a function that converts a source field value to a destination field value.
// It is registered by field name and applies to any source/destination struct pair.
type ConverterFunc func(src interface{}) (interface{}, error)

// Composition helpers
// ComposeConverters chains multiple ConverterFunc instances left-to-right.
// If any converter returns an error it aborts.
// Nil output propagates immediately.
func ComposeConverters(fns ...ConverterFunc) ConverterFunc {
	return func(src interface{}) (interface{}, error) {
		cur := src
		for _, fn := range fns {
			out, err := fn(cur)
			if err != nil {
				return nil, err
			}
			if out == nil {
				return nil, nil
			}
			cur = out
		}
		return cur, nil
	}
}

// MapString returns a ConverterFunc applying f when src is a string; otherwise returns src unchanged.
func MapString(f func(string) string) ConverterFunc {
	return func(src interface{}) (interface{}, error) {
		if s, ok := src.(string); ok {
			return f(s), nil
		}
		return src, nil
	}
}

// OverwritePolicy controls how AdditionalData values interact with already-set fields
type OverwritePolicy int

const (
	PreferFields         OverwritePolicy = iota // default: do not overwrite fields set from direct mapping
	PreferAdditionalData                        // overwrite fields with values from AdditionalData if present
)

type Options struct {
	IncludeZeroValues             bool            // when true, include zero-valued fields in marshaled AdditionalData
	CaseInsensitiveAdditionalData bool            // when true, AdditionalData keys are matched case-insensitively
	OverwritePolicy               OverwritePolicy // controls if AdditionalData overwrites direct fields
}

type Option func(*Options)

func WithIncludeZeroValues(v bool) Option { return func(o *Options) { o.IncludeZeroValues = v } }
func WithCaseInsensitiveAdditionalData(v bool) Option {
	return func(o *Options) { o.CaseInsensitiveAdditionalData = v }
}
func WithOverwritePolicy(p OverwritePolicy) Option { return func(o *Options) { o.OverwritePolicy = p } }

// converterRegistry stores converters at multiple scopes and is swapped atomically (copy-on-write)
type converterRegistry struct {
	global map[string]ConverterFunc
	byDst  map[reflect.Type]map[string]ConverterFunc
	byPair map[[2]reflect.Type]map[string]ConverterFunc // [srcType, dstType]
}

type fieldInfo struct {
	index            []int
	name             string
	jsonName         string
	typ              reflect.Type
	canSet           bool
	isAdditionalData bool
	ignore           bool
}

type structMetadata struct {
	fields              []fieldInfo
	fieldsByName        map[string]*fieldInfo
	fieldsByJSONName    map[string]*fieldInfo
	additionalDataField *fieldInfo
}

// Adapter performs struct adaptation with optional converters & AdditionalData handling.
// See README for usage and option guidelines.
type Adapter struct {
	converters    atomic.Value // holds *converterRegistry
	metadataCache sync.Map     // map[reflect.Type]*structMetadata
	boolMapPool   sync.Pool    // Pool for map[string]bool reuse
	options       Options
}

// New creates an Adapter with default options.
func New() *Adapter { return NewWithOptions() }

// NewWithOptions creates a new Adapter with provided options.
func NewWithOptions(opts ...Option) *Adapter {
	a := &Adapter{}
	optsState := Options{IncludeZeroValues: false, CaseInsensitiveAdditionalData: false, OverwritePolicy: PreferFields}
	for _, f := range opts {
		f(&optsState)
	}
	a.options = optsState
	reg := &converterRegistry{global: make(map[string]ConverterFunc), byDst: make(map[reflect.Type]map[string]ConverterFunc), byPair: make(map[[2]reflect.Type]map[string]ConverterFunc)}
	a.converters.Store(reg)
	a.boolMapPool = sync.Pool{New: func() interface{} { return (map[string]bool)(nil) }}
	return a
}

// RegisterConverter adds a global field converter (applies to any src/dst containing fieldName).
func (a *Adapter) RegisterConverter(fieldName string, fn ConverterFunc) {
	old := a.converters.Load().(*converterRegistry)
	newReg := &converterRegistry{
		global: make(map[string]ConverterFunc, len(old.global)+1),
		byDst:  make(map[reflect.Type]map[string]ConverterFunc, len(old.byDst)),
		byPair: make(map[[2]reflect.Type]map[string]ConverterFunc, len(old.byPair)),
	}
	for k, v := range old.global {
		newReg.global[k] = v
	}
	for k, v := range old.byDst {
		m := make(map[string]ConverterFunc, len(v))
		for fk, fv := range v {
			m[fk] = fv
		}
		newReg.byDst[k] = m
	}
	for k, v := range old.byPair {
		m := make(map[string]ConverterFunc, len(v))
		for fk, fv := range v {
			m[fk] = fv
		}
		newReg.byPair[k] = m
	}
	newReg.global[fieldName] = fn
	a.converters.Store(newReg)
}

// RegisterConverterFor scope: destination type + fieldName.
func (a *Adapter) RegisterConverterFor(dstType any, fieldName string, fn ConverterFunc) {
	old := a.converters.Load().(*converterRegistry)
	newReg := &converterRegistry{
		global: make(map[string]ConverterFunc, len(old.global)),
		byDst:  make(map[reflect.Type]map[string]ConverterFunc, len(old.byDst)+1),
		byPair: make(map[[2]reflect.Type]map[string]ConverterFunc, len(old.byPair)),
	}
	for k, v := range old.global {
		newReg.global[k] = v
	}
	for k, v := range old.byDst {
		m := make(map[string]ConverterFunc, len(v))
		for fk, fv := range v {
			m[fk] = fv
		}
		newReg.byDst[k] = m
	}
	for k, v := range old.byPair {
		m := make(map[string]ConverterFunc, len(v))
		for fk, fv := range v {
			m[fk] = fv
		}
		newReg.byPair[k] = m
	}
	dt := reflect.TypeOf(dstType)
	if dt.Kind() == reflect.Ptr {
		dt = dt.Elem()
	}
	m := newReg.byDst[dt]
	if m == nil {
		m = make(map[string]ConverterFunc)
		newReg.byDst[dt] = m
	}
	m[fieldName] = fn
	a.converters.Store(newReg)
}

// RegisterConverterForPair scope: (srcType,dstType)+fieldName highest precedence.
func (a *Adapter) RegisterConverterForPair(srcType, dstType any, fieldName string, fn ConverterFunc) {
	old := a.converters.Load().(*converterRegistry)
	newReg := &converterRegistry{
		global: make(map[string]ConverterFunc, len(old.global)),
		byDst:  make(map[reflect.Type]map[string]ConverterFunc, len(old.byDst)),
		byPair: make(map[[2]reflect.Type]map[string]ConverterFunc, len(old.byPair)+1),
	}
	for k, v := range old.global {
		newReg.global[k] = v
	}
	for k, v := range old.byDst {
		m := make(map[string]ConverterFunc, len(v))
		for fk, fv := range v {
			m[fk] = fv
		}
		newReg.byDst[k] = m
	}
	for k, v := range old.byPair {
		m := make(map[string]ConverterFunc, len(v))
		for fk, fv := range v {
			m[fk] = fv
		}
		newReg.byPair[k] = m
	}
	st := reflect.TypeOf(srcType)
	if st.Kind() == reflect.Ptr {
		st = st.Elem()
	}
	dt := reflect.TypeOf(dstType)
	if dt.Kind() == reflect.Ptr {
		dt = dt.Elem()
	}
	key := [2]reflect.Type{st, dt}
	m := newReg.byPair[key]
	if m == nil {
		m = make(map[string]ConverterFunc)
		newReg.byPair[key] = m
	}
	m[fieldName] = fn
	a.converters.Store(newReg)
}

// Adapt performs adaptation from src -> dst.
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

// --- metadata helpers ---
func (a *Adapter) getBoolMap(capHint int) map[string]bool {
	pooled := a.boolMapPool.Get().(map[string]bool)
	if pooled == nil {
		return make(map[string]bool, capHint)
	}
	for k := range pooled {
		delete(pooled, k)
	}
	return pooled
}
func (a *Adapter) putBoolMap(m map[string]bool) {
	if m != nil && len(m) <= 128 {
		a.boolMapPool.Put(m)
	}
}

func (a *Adapter) getOrBuildMetadata(typ reflect.Type) *structMetadata {
	if cached, ok := a.metadataCache.Load(typ); ok {
		return cached.(*structMetadata)
	}
	fc := a.countFields(typ)
	meta := &structMetadata{fields: make([]fieldInfo, 0, fc), fieldsByName: make(map[string]*fieldInfo, fc), fieldsByJSONName: make(map[string]*fieldInfo, fc)}
	a.buildFieldMetadata(typ, meta, nil)
	for i := range meta.fields {
		fi := &meta.fields[i]
		meta.fieldsByName[fi.name] = fi
		if fi.jsonName != "" {
			meta.fieldsByJSONName[fi.jsonName] = fi
		}
	}
	if ad, ok := meta.fieldsByName["AdditionalData"]; ok && ad.isAdditionalData {
		meta.additionalDataField = ad
	}
	actual, _ := a.metadataCache.LoadOrStore(typ, meta)
	return actual.(*structMetadata)
}

func (a *Adapter) safeFieldByIndex(val reflect.Value, index []int) (reflect.Value, bool) {
	for i, x := range index {
		if i > 0 && val.Kind() == reflect.Ptr {
			if val.IsNil() {
				return reflect.Value{}, false
			}
			val = val.Elem()
		}
		val = val.Field(x)
	}
	return val, true
}

func (a *Adapter) countFields(typ reflect.Type) int {
	c := 0
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.PkgPath != "" {
			continue
		}
		if f.Anonymous {
			ft := f.Type
			if ft.Kind() == reflect.Ptr {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				c += a.countFields(ft)
				continue
			}
		}
		c++
	}
	return c
}

func (a *Adapter) buildFieldMetadata(typ reflect.Type, meta *structMetadata, prefix []int) {
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		idx := append(append([]int(nil), prefix...), i)
		if f.Anonymous {
			ft := f.Type
			if ft.Kind() == reflect.Ptr {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				a.buildFieldMetadata(ft, meta, idx)
				continue
			}
		}
		if f.PkgPath != "" {
			continue
		}
		adapterTag := f.Tag.Get("adapter")
		ignore := adapterTag == "ignore" || adapterTag == "-"
		jsonName := ""
		if jt, ok := f.Tag.Lookup("json"); ok {
			for j := 0; j < len(jt); j++ {
				if jt[j] == ',' {
					jt = jt[:j]
					break
				}
			}
			if jt != "-" {
				jsonName = jt
			}
		}
		isAD := f.Name == "AdditionalData" && (f.Type == reflect.TypeOf(null.JSON{}) || f.Type == reflect.TypeOf(boilertypes.JSON{}))
		meta.fields = append(meta.fields, fieldInfo{index: idx, name: f.Name, jsonName: jsonName, typ: f.Type, canSet: true, isAdditionalData: isAD, ignore: ignore})
	}
}

// --- core adaptation ---
func (a *Adapter) adaptStruct(dstVal, srcVal reflect.Value) error {
	dt := dstVal.Type()
	st := srcVal.Type()
	dstMeta := a.getOrBuildMetadata(dt)
	srcMeta := a.getOrBuildMetadata(st)
	hasAD := srcMeta.additionalDataField != nil || dstMeta.additionalDataField != nil
	var processed, dstSet map[string]bool
	if hasAD {
		capHint := len(srcMeta.fields)
		if len(dstMeta.fields) > capHint {
			capHint = len(dstMeta.fields)
		}
		processed = a.getBoolMap(capHint)
		dstSet = a.getBoolMap(capHint)
		defer func() { a.putBoolMap(processed); a.putBoolMap(dstSet) }()
	}
	for i := range dstMeta.fields {
		df := &dstMeta.fields[i]
		if !df.canSet || df.isAdditionalData || df.ignore {
			continue
		}
		sf, found := srcMeta.fieldsByName[df.name]
		if !found && df.jsonName != "" {
			sf, found = srcMeta.fieldsByJSONName[df.jsonName]
		}
		if !found {
			continue
		}
		if sf.isAdditionalData || sf.ignore {
			if hasAD {
				processed[sf.name] = true
			}
			continue
		}
		srcField, ok := a.safeFieldByIndex(srcVal, sf.index)
		if !ok {
			continue
		}
		dstField := dstVal.FieldByIndex(df.index)
		if err := a.adaptField(dstField, srcField, df.name, st, dt); err != nil {
			return fmt.Errorf("adapting field %s: %w", df.name, err)
		}
		if hasAD {
			processed[sf.name] = true
			dstSet[df.name] = true
		}
	}
	if srcMeta.additionalDataField != nil {
		srcAD := srcVal.FieldByIndex(srcMeta.additionalDataField.index)
		if err := a.unmarshalAdditionalData(dstVal, dstMeta, srcAD, dstSet); err != nil {
			return fmt.Errorf("unmarshaling AdditionalData: %w", err)
		}
		if hasAD {
			processed["AdditionalData"] = true
		}
	}
	if dstMeta.additionalDataField != nil {
		dstAD := dstVal.FieldByIndex(dstMeta.additionalDataField.index)
		if err := a.marshalRemainingFields(dstAD, srcVal, st, processed); err != nil {
			return fmt.Errorf("marshaling remaining fields to AdditionalData: %w", err)
		}
	}
	return nil
}

func (a *Adapter) adaptField(dstField, srcField reflect.Value, fieldName string, srcRoot, dstRoot reflect.Type) error {
	if !dstField.CanSet() {
		return fmt.Errorf("cannot set field %s (unexported or unsettable)", fieldName)
	}
	reg := a.converters.Load().(*converterRegistry)
	// precedence pair > dst > global
	if fn := reg.byPair[[2]reflect.Type{srcRoot, dstRoot}][fieldName]; fn != nil {
		return a.applyConverter(dstField, fn, srcField, fieldName)
	}
	if fn := reg.byDst[dstRoot][fieldName]; fn != nil {
		return a.applyConverter(dstField, fn, srcField, fieldName)
	}
	if fn := reg.global[fieldName]; fn != nil {
		return a.applyConverter(dstField, fn, srcField, fieldName)
	}
	srcType := srcField.Type()
	dstType := dstField.Type()
	if srcType == dstType {
		dstField.Set(srcField)
		return nil
	}
	if srcType.AssignableTo(dstType) {
		dstField.Set(srcField)
		return nil
	}
	if srcType.ConvertibleTo(dstType) {
		dstField.Set(srcField.Convert(dstType))
		return nil
	}
	return nil
}

func (a *Adapter) applyConverter(dstField reflect.Value, fn ConverterFunc, srcField reflect.Value, fieldName string) error {
	converted, err := fn(srcField.Interface())
	if err != nil {
		return err
	}
	if converted == nil {
		dstField.Set(reflect.Zero(dstField.Type()))
		return nil
	}
	cv := reflect.ValueOf(converted)
	if !cv.IsValid() {
		return fmt.Errorf("converter returned invalid value for field %s", fieldName)
	}
	if !cv.Type().AssignableTo(dstField.Type()) {
		return fmt.Errorf("converter returned type %s, expected %s", cv.Type(), dstField.Type())
	}
	dstField.Set(cv)
	return nil
}

func (a *Adapter) unmarshalAdditionalData(dstVal reflect.Value, dstMeta *structMetadata, srcAdditionalData reflect.Value, dstFieldsSet map[string]bool) error {
	var rawBytes []byte
	if nj, ok := srcAdditionalData.Interface().(null.JSON); ok {
		if !nj.Valid {
			return nil
		}
		rawBytes = nj.JSON
	} else if bj, ok := srcAdditionalData.Interface().(boilertypes.JSON); ok {
		if len(bj) == 0 {
			return nil
		}
		rawBytes = bj
	} else {
		return nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(rawBytes, &fields); err != nil {
		return err
	}
	reg := a.converters.Load().(*converterRegistry)
	lookupInsensitive := a.options.CaseInsensitiveAdditionalData
	lookup := func(key string) (*fieldInfo, bool, string) {
		if !lookupInsensitive {
			if fi, ok := dstMeta.fieldsByName[key]; ok {
				return fi, true, fi.name
			}
			if fi, ok := dstMeta.fieldsByJSONName[key]; ok {
				return fi, true, fi.name
			}
			return nil, false, ""
		}
		lk := strings.ToLower(key)
		if fi, ok := dstMeta.fieldsByName[key]; ok {
			return fi, true, fi.name
		}
		if fi, ok := dstMeta.fieldsByJSONName[key]; ok {
			return fi, true, fi.name
		}
		for n, fi := range dstMeta.fieldsByName {
			if strings.ToLower(n) == lk {
				return fi, true, fi.name
			}
		}
		for jn, fi := range dstMeta.fieldsByJSONName {
			if strings.ToLower(jn) == lk {
				return fi, true, fi.name
			}
		}
		return nil, false, ""
	}
	for k, raw := range fields {
		fi, ok, canon := lookup(k)
		if !ok || !fi.canSet || fi.ignore {
			continue
		}
		if a.options.OverwritePolicy == PreferFields && dstFieldsSet[canon] {
			continue
		}
		dstField := dstVal.FieldByIndex(fi.index)
		if fn := reg.global[fi.name]; fn != nil { // converter path
			var anyVal interface{}
			if err := json.Unmarshal(raw, &anyVal); err == nil {
				converted, err := fn(anyVal)
				if err == nil && converted != nil {
					cv := reflect.ValueOf(converted)
					if cv.IsValid() && cv.Type().AssignableTo(dstField.Type()) {
						dstField.Set(cv)
						dstFieldsSet[canon] = true
					}
				}
			}
			// Do not fallback to direct unmarshal when a converter is registered, regardless of outcome
			continue
		}
		ptr := reflect.New(dstField.Type())
		if err := json.Unmarshal(raw, ptr.Interface()); err != nil {
			continue
		}
		dstField.Set(ptr.Elem())
		dstFieldsSet[canon] = true
	}
	return nil
}

func (a *Adapter) marshalRemainingFields(dstAdditionalData reflect.Value, srcVal reflect.Value, srcType reflect.Type, processed map[string]bool) error {
	remaining := make(map[string]interface{})
	srcMeta := a.getOrBuildMetadata(srcType)
	for i := range srcMeta.fields {
		sf := &srcMeta.fields[i]
		if sf.isAdditionalData || sf.ignore {
			continue
		}
		if processed[sf.name] {
			continue
		}
		srcField, ok := a.safeFieldByIndex(srcVal, sf.index)
		if !ok || !srcField.CanInterface() {
			continue
		}
		if !a.options.IncludeZeroValues && srcField.IsZero() {
			continue
		}
		remaining[sf.name] = srcField.Interface()
	}
	bytes, err := json.Marshal(remaining)
	if err != nil {
		return err
	}
	t := dstAdditionalData.Type()
	if t == reflect.TypeOf(null.JSON{}) {
		if len(remaining) == 0 {
			dstAdditionalData.Set(reflect.ValueOf(null.JSON{}))
		} else {
			dstAdditionalData.Set(reflect.ValueOf(null.JSONFrom(bytes)))
		}
	} else if t == reflect.TypeOf(boilertypes.JSON{}) {
		if len(remaining) == 0 {
			dstAdditionalData.Set(reflect.ValueOf(boilertypes.JSON(nil)))
		} else {
			dstAdditionalData.Set(reflect.ValueOf(boilertypes.JSON(bytes)))
		}
	}
	return nil
}
