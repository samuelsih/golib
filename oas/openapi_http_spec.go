package oas

import (
	"encoding"
	"encoding/json/jsontext"
	"fmt"
	"path"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/guregu/null/v6"
	"github.com/samuelsih/golib/reflectx"
)

var (
	timeType          = reflect.TypeFor[time.Time]()
	durationType      = reflect.TypeFor[time.Duration]()
	jsonValueType     = reflect.TypeFor[jsontext.Value]()
	textMarshalerType = reflect.TypeFor[encoding.TextMarshaler]()
)

// schemaBuilder turns Go types into JSON Schema and stores named structs in
// components/schemas.
type schemaBuilder struct {
	refs   map[reflect.Type]string
	byName map[string]reflect.Type
	comps  map[string]RefT[Schema]
	errs   []error
}

func newSchemaBuilder() *schemaBuilder {
	return &schemaBuilder{
		refs:   map[reflect.Type]string{},
		byName: map[string]reflect.Type{},
		comps:  map[string]RefT[Schema]{},
	}
}

func (b *schemaBuilder) addErr(err error) {
	if err != nil {
		b.errs = append(b.errs, err)
	}
}

func (b *schemaBuilder) build(t reflect.Type) RefT[Schema] {
	if t == nil {
		return schemaOf(&Schema{})
	}

	if t.Kind() == reflect.Pointer {
		return nullableSchema(b.build(t.Elem()))
	}
	if inner, ok := nullableBase(t); ok {
		return nullableSchema(b.build(inner))
	}
	if schema, ok := b.buildSpecial(t); ok {
		return schema
	}
	if t.Kind() == reflect.Struct && t.Name() != "" {
		return b.buildRef(t)
	}

	return b.buildInline(t)
}

func (b *schemaBuilder) buildSpecial(t reflect.Type) (RefT[Schema], bool) {
	switch t {
	case timeType:
		return schemaOf(&Schema{Type: Strings{"string"}, Format: null.StringFrom("date-time")}), true
	case durationType:
		return schemaOf(&Schema{Type: Strings{"integer"}, Format: null.StringFrom("int64")}), true
	case jsonValueType:
		return schemaOf(&Schema{}), true
	}

	if t.Implements(textMarshalerType) || reflect.PointerTo(t).Implements(textMarshalerType) {
		return schemaOf(&Schema{Type: Strings{"string"}}), true
	}

	return RefT[Schema]{}, false
}

func (b *schemaBuilder) buildRef(t reflect.Type) RefT[Schema] {
	if name, ok := b.refs[t]; ok {
		return refSchema(name)
	}

	name := b.componentName(t)
	b.refs[t] = name
	b.byName[name] = t
	b.comps[name] = schemaOf(b.objectSchema(t))

	return refSchema(name)
}

func (b *schemaBuilder) buildInline(t reflect.Type) RefT[Schema] {
	switch t.Kind() {
	case reflect.Bool:
		return schemaOf(&Schema{Type: Strings{"boolean"}})
	case reflect.String:
		return schemaOf(&Schema{Type: Strings{"string"}})
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		schema := &Schema{Type: Strings{"integer"}}
		if t.Kind() == reflect.Int64 || t.Kind() == reflect.Uint64 {
			schema.Format = null.StringFrom("int64")
		}
		return schemaOf(schema)
	case reflect.Float32:
		return schemaOf(&Schema{Type: Strings{"number"}, Format: null.StringFrom("float")})
	case reflect.Float64:
		return schemaOf(&Schema{Type: Strings{"number"}, Format: null.StringFrom("double")})
	case reflect.Slice, reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 {
			return schemaOf(&Schema{Type: Strings{"string"}, Format: null.StringFrom("byte")})
		}
		return schemaOf(&Schema{Type: Strings{"array"}, Items: b.build(t.Elem())})
	case reflect.Map:
		return schemaOf(&Schema{
			Type:                 Strings{"object"},
			AdditionalProperties: BoolOrSchema{Schema: b.build(t.Elem())},
		})
	case reflect.Struct:
		return schemaOf(b.objectSchema(t))
	case reflect.Interface:
		return schemaOf(&Schema{})
	default:
		b.addErr(fmt.Errorf("oas: unsupported schema type %s", t))
		return schemaOf(&Schema{})
	}
}

func (b *schemaBuilder) objectSchema(t reflect.Type) *Schema {
	properties := map[string]RefT[Schema]{}
	var required []string

	reflectx.WalkFields(t, shouldFlattenJSON, func(sf reflect.StructField) {
		field := reflectx.JSONFieldOf(sf)
		if field.Skip {
			return
		}

		properties[field.Name] = applyFieldTags(b.build(sf.Type), sf, false)
		if fieldRequired(sf, field.Omit) && !slices.Contains(required, field.Name) {
			required = append(required, field.Name)
		}
	})

	schema := &Schema{Type: Strings{"object"}}
	if len(properties) > 0 {
		schema.Properties = properties
	}
	if len(required) > 0 {
		schema.Required = required
	}

	return schema
}

// shouldFlattenJSON reports whether an anonymous field is embedded in its
// parent object instead of becoming a nested property.
func shouldFlattenJSON(sf reflect.StructField) bool {
	field := reflectx.JSONFieldOf(sf)

	return !field.Skip && field.Name == sf.Name &&
		reflectx.Indirect(sf.Type).Kind() == reflect.Struct
}

func (b *schemaBuilder) componentName(t reflect.Type) string {
	name := sanitizeComponentName(t.Name())
	if name == "" {
		name = "Schema"
	}
	if _, ok := b.byName[name]; !ok {
		return name
	}

	if pkg := path.Base(t.PkgPath()); pkg != "." && pkg != "" {
		candidate := sanitizeComponentName(pkg + "_" + t.Name())
		if _, ok := b.byName[candidate]; !ok {
			return candidate
		}
	}

	candidate := sanitizeComponentName(t.PkgPath() + "_" + t.Name())

	for i := 2; ; i++ {
		name := candidate + "_" + strconv.Itoa(i)
		if _, ok := b.byName[name]; !ok {
			return name
		}
	}
}

func sanitizeComponentName(name string) string {
	var out strings.Builder

	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '_', r == '-', r == '.':
			out.WriteRune(r)
		default:
			out.WriteByte('_')
		}
	}

	return strings.Trim(out.String(), "_")
}

func refSchema(name string) RefT[Schema] {
	return RefT[Schema]{Ref: &Reference{Ref: "#/components/schemas/" + name}}
}

func schemaOf(schema *Schema) RefT[Schema] {
	return RefT[Schema]{Value: schema}
}

func nullableSchema(base RefT[Schema]) RefT[Schema] {
	if base.Value != nil && len(base.Value.Type) > 0 &&
		base.Value.AnyOf == nil && base.Value.OneOf == nil && base.Value.AllOf == nil {
		clone := *base.Value
		if !slices.Contains(clone.Type, "null") {
			clone.Type = append(slices.Clone(clone.Type), "null")
		}
		return schemaOf(&clone)
	}

	return schemaOf(&Schema{AnyOf: []RefT[Schema]{
		base,
		{Value: &Schema{Type: Strings{"null"}}},
	}})
}

// nullableBase reports the value type wrapped by database/sql and guregu/null
// null containers such as null.String or null.Value[T].
func nullableBase(t reflect.Type) (reflect.Type, bool) {
	switch t.PkgPath() {
	case "github.com/guregu/null/v6", "database/sql":
	default:
		return nil, false
	}

	for current := t; current.Kind() == reflect.Struct; {
		var (
			value    reflect.Type
			embedded reflect.Type
			valid    bool
		)

		for f := range current.Fields() {
			switch {
			case f.Name == "Valid" && f.Type.Kind() == reflect.Bool:
				valid = true
			case f.Anonymous:
				embedded = f.Type
			default:
				value = f.Type
			}
		}

		if valid && value != nil {
			return value, true
		}
		if embedded == nil {
			return nil, false
		}

		current = embedded
	}

	return nil, false
}

func fieldRequired(sf reflect.StructField, omit bool) bool {
	if v, ok := reflectx.GetTagValueAs[bool](sf, "required"); ok {
		return v
	}
	if omit {
		return false
	}

	return sf.Type.Kind() != reflect.Pointer
}

func applyFieldTags(base RefT[Schema], sf reflect.StructField, forParameter bool) RefT[Schema] {
	if !hasSchemaTags(sf.Tag, forParameter) {
		return base
	}

	var schema *Schema
	if base.Value != nil {
		clone := *base.Value
		schema = &clone
	} else {
		schema = &Schema{AllOf: []RefT[Schema]{base}}
	}

	applySchemaTags(schema, sf, forParameter)

	return schemaOf(schema)
}

var schemaTagKeys = [...]string{
	"title", "description", "format", "pattern",
	"minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum", "multipleOf",
	"minLength", "maxLength", "minItems", "maxItems", "uniqueItems",
	"minProperties", "maxProperties", "enum", "default", "example",
	"deprecated", "readOnly", "writeOnly",
}

func hasSchemaTags(tag reflect.StructTag, forParameter bool) bool {
	for key := range slices.Values(schemaTagKeys[:]) {
		if forParameter && (key == "description" || key == "example") {
			continue
		}
		if _, ok := tag.Lookup(key); ok {
			return true
		}
	}

	return false
}

func applySchemaTags(schema *Schema, sf reflect.StructField, forParameter bool) {
	setFloat := func(key string, dst *null.Float) {
		if v, ok := reflectx.GetTagValueAs[float64](sf, key); ok {
			*dst = null.FloatFrom(v)
		}
	}
	setInt := func(key string, dst *null.Int) {
		if v, ok := reflectx.GetTagValueAs[int64](sf, key); ok {
			*dst = null.IntFrom(v)
		}
	}
	setBool := func(key string, dst *null.Bool) {
		if v, ok := reflectx.GetTagValueAs[bool](sf, key); ok {
			*dst = null.BoolFrom(v)
		}
	}

	if v, ok := reflectx.GetTagValueAs[string](sf, "title"); ok && v != "" {
		schema.Title = null.StringFrom(v)
	}
	if !forParameter {
		if v, ok := reflectx.GetTagValueAs[string](sf, "description"); ok && v != "" {
			schema.Description = null.StringFrom(v)
		}
	}
	if v, ok := reflectx.GetTagValueAs[string](sf, "format"); ok && v != "" {
		schema.Format = null.StringFrom(v)
	}
	if v, ok := reflectx.GetTagValueAs[string](sf, "pattern"); ok && v != "" {
		schema.Pattern = null.StringFrom(v)
	}

	setFloat("minimum", &schema.Minimum)
	setFloat("maximum", &schema.Maximum)
	setFloat("exclusiveMinimum", &schema.ExclusiveMinimum)
	setFloat("exclusiveMaximum", &schema.ExclusiveMaximum)
	setFloat("multipleOf", &schema.MultipleOf)

	setInt("minLength", &schema.MinLength)
	setInt("maxLength", &schema.MaxLength)
	setInt("minItems", &schema.MinItems)
	setInt("maxItems", &schema.MaxItems)
	setInt("minProperties", &schema.MinProperties)
	setInt("maxProperties", &schema.MaxProperties)

	setBool("uniqueItems", &schema.UniqueItems)
	setBool("deprecated", &schema.Deprecated)
	setBool("readOnly", &schema.ReadOnly)
	setBool("writeOnly", &schema.WriteOnly)

	valueType := schemaValueType(schema)

	if v, ok := reflectx.GetTagValueAs[string](sf, "enum"); ok {
		parts := strings.Split(v, ",")
		schema.Enum = make([]any, 0, len(parts))
		for part := range slices.Values(parts) {
			schema.Enum = append(schema.Enum, convertScalar(strings.TrimSpace(part), valueType))
		}
	}
	if v, ok := typedTagValue(sf, "default", valueType); ok {
		schema.Default = v
	}
	if !forParameter {
		if v, ok := typedTagValue(sf, "example", valueType); ok {
			schema.Examples = []any{v}
		}
	}
}

func schemaValueType(schema *Schema) string {
	if schema == nil {
		return ""
	}

	for t := range slices.Values(schema.Type) {
		if t != "null" {
			return t
		}
	}

	return ""
}

func typedTagValue(sf reflect.StructField, key, schemaType string) (any, bool) {
	switch schemaType {
	case "boolean":
		v, ok := reflectx.GetTagValueAs[bool](sf, key)
		return v, ok
	case "integer":
		v, ok := reflectx.GetTagValueAs[int64](sf, key)
		return v, ok
	case "number":
		v, ok := reflectx.GetTagValueAs[float64](sf, key)
		return v, ok
	default:
		v, ok := reflectx.GetTagValueAs[string](sf, key)
		return v, ok
	}
}

func convertScalar(raw, schemaType string) any {
	switch schemaType {
	case "boolean":
		if v, err := strconv.ParseBool(raw); err == nil {
			return v
		}
	case "integer":
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
			return v
		}
	case "number":
		if v, err := strconv.ParseFloat(raw, 64); err == nil {
			return v
		}
	}

	return raw
}
