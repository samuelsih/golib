package oas

import (
	"encoding/json/jsontext"
	"encoding/json/v2"

	"github.com/guregu/null/v6"
)

// OpenAPI is the root OpenAPI Object of an OpenAPI 3.1 document.
type OpenAPI struct {
	OpenAPI           string                            `json:"openapi"`
	Info              Info                              `json:"info"`
	JSONSchemaDialect null.String                       `json:"jsonSchemaDialect,omitzero"`
	Servers           []Server                          `json:"servers,omitempty"`
	Paths             Paths                             `json:"paths,omitempty"`
	Webhooks          map[string]RefT[PathItem]         `json:"webhooks,omitempty"`
	Components        null.Value[Components]            `json:"components,omitzero"`
	Security          []SecurityRequirement             `json:"security,omitempty"`
	Tags              []Tag                             `json:"tags,omitempty"`
	ExternalDocs      null.Value[ExternalDocumentation] `json:"externalDocs,omitzero"`
	Extensions        map[string]any                    `json:"-"`
}

// Info describes the API metadata.
type Info struct {
	Title          string              `json:"title"`
	Summary        null.String         `json:"summary,omitzero"`
	Description    null.String         `json:"description,omitzero"`
	TermsOfService null.String         `json:"termsOfService,omitzero"`
	Contact        null.Value[Contact] `json:"contact,omitzero"`
	License        null.Value[License] `json:"license,omitzero"`
	Version        string              `json:"version"`
	Extensions     map[string]any      `json:"-"`
}

// Contact is the contact information for the API.
type Contact struct {
	Name       null.String    `json:"name,omitzero"`
	URL        null.String    `json:"url,omitzero"`
	Email      null.String    `json:"email,omitzero"`
	Extensions map[string]any `json:"-"`
}

// License is the license information for the API.
type License struct {
	Name       string         `json:"name"`
	Identifier null.String    `json:"identifier,omitzero"`
	URL        null.String    `json:"url,omitzero"`
	Extensions map[string]any `json:"-"`
}

// ExternalDocumentation points to additional external documentation.
type ExternalDocumentation struct {
	Description null.String    `json:"description,omitzero"`
	URL         string         `json:"url"`
	Extensions  map[string]any `json:"-"`
}

// Tag adds metadata to a group of operations.
type Tag struct {
	Name         string                            `json:"name"`
	Description  null.String                       `json:"description,omitzero"`
	ExternalDocs null.Value[ExternalDocumentation] `json:"externalDocs,omitzero"`
	Extensions   map[string]any                    `json:"-"`
}

// Server describes a server that hosts the API.
type Server struct {
	URL         string                    `json:"url"`
	Description null.String               `json:"description,omitzero"`
	Variables   map[string]ServerVariable `json:"variables,omitempty"`
	Extensions  map[string]any            `json:"-"`
}

// ServerVariable is a variable for server URL template substitution.
type ServerVariable struct {
	Enum        []string       `json:"enum,omitempty"`
	Default     string         `json:"default"`
	Description null.String    `json:"description,omitzero"`
	Extensions  map[string]any `json:"-"`
}

// Paths maps a relative path to a PathItem.
type Paths map[string]RefT[PathItem]

// Callback maps a runtime expression to a PathItem.
type Callback map[string]RefT[PathItem]

// PathItem describes the operations available on a single path.
type PathItem struct {
	Summary     null.String           `json:"summary,omitzero"`
	Description null.String           `json:"description,omitzero"`
	Get         null.Value[Operation] `json:"get,omitzero"`
	Put         null.Value[Operation] `json:"put,omitzero"`
	Post        null.Value[Operation] `json:"post,omitzero"`
	Delete      null.Value[Operation] `json:"delete,omitzero"`
	Options     null.Value[Operation] `json:"options,omitzero"`
	Head        null.Value[Operation] `json:"head,omitzero"`
	Patch       null.Value[Operation] `json:"patch,omitzero"`
	Trace       null.Value[Operation] `json:"trace,omitzero"`
	Servers     []Server              `json:"servers,omitempty"`
	Parameters  []RefT[Parameter]     `json:"parameters,omitempty"`
	Extensions  map[string]any        `json:"-"`
}

// Operation describes a single API operation on a path.
type Operation struct {
	Tags         []string                          `json:"tags,omitempty"`
	Summary      null.String                       `json:"summary,omitzero"`
	Description  null.String                       `json:"description,omitzero"`
	ExternalDocs null.Value[ExternalDocumentation] `json:"externalDocs,omitzero"`
	OperationID  null.String                       `json:"operationId,omitzero"`
	Parameters   []RefT[Parameter]                 `json:"parameters,omitempty"`
	RequestBody  RefT[RequestBody]                 `json:"requestBody,omitzero"`
	Responses    Responses                         `json:"responses,omitempty"`
	Callbacks    map[string]RefT[Callback]         `json:"callbacks,omitempty"`
	Deprecated   null.Bool                         `json:"deprecated,omitzero"`
	Security     []SecurityRequirement             `json:"security,omitempty"`
	Servers      []Server                          `json:"servers,omitempty"`
	Extensions   map[string]any                    `json:"-"`
}

// Components holds reusable objects for the API.
type Components struct {
	Schemas         map[string]RefT[Schema]         `json:"schemas,omitempty"`
	Responses       map[string]RefT[Response]       `json:"responses,omitempty"`
	Parameters      map[string]RefT[Parameter]      `json:"parameters,omitempty"`
	Examples        map[string]RefT[Example]        `json:"examples,omitempty"`
	RequestBodies   map[string]RefT[RequestBody]    `json:"requestBodies,omitempty"`
	Headers         map[string]RefT[Header]         `json:"headers,omitempty"`
	SecuritySchemes map[string]RefT[SecurityScheme] `json:"securitySchemes,omitempty"`
	Links           map[string]RefT[Link]           `json:"links,omitempty"`
	Callbacks       map[string]RefT[Callback]       `json:"callbacks,omitempty"`
	PathItems       map[string]RefT[PathItem]       `json:"pathItems,omitempty"`
	Extensions      map[string]any                  `json:"-"`
}

// ParameterIn is the location of an operation parameter.
type ParameterIn string

const (
	ParameterInQuery  ParameterIn = "query"
	ParameterInHeader ParameterIn = "header"
	ParameterInPath   ParameterIn = "path"
	ParameterInCookie ParameterIn = "cookie"
)

// ParameterStyle is the serialization style of a parameter.
type ParameterStyle string

const (
	ParameterStyleMatrix         ParameterStyle = "matrix"
	ParameterStyleLabel          ParameterStyle = "label"
	ParameterStyleForm           ParameterStyle = "form"
	ParameterStyleSimple         ParameterStyle = "simple"
	ParameterStyleSpaceDelimited ParameterStyle = "spaceDelimited"
	ParameterStylePipeDelimited  ParameterStyle = "pipeDelimited"
	ParameterStyleDeepObject     ParameterStyle = "deepObject"
)

// Parameter describes a single operation parameter.
type Parameter struct {
	Name            string                     `json:"name"`
	In              ParameterIn                `json:"in"`
	Description     null.String                `json:"description,omitzero"`
	Required        null.Bool                  `json:"required,omitzero"`
	Deprecated      null.Bool                  `json:"deprecated,omitzero"`
	AllowEmptyValue null.Bool                  `json:"allowEmptyValue,omitzero"`
	Style           null.Value[ParameterStyle] `json:"style,omitzero"`
	Explode         null.Bool                  `json:"explode,omitzero"`
	AllowReserved   null.Bool                  `json:"allowReserved,omitzero"`
	Schema          RefT[Schema]               `json:"schema,omitzero"`
	Example         any                        `json:"example,omitempty"`
	Examples        map[string]RefT[Example]   `json:"examples,omitempty"`
	Content         map[string]MediaType       `json:"content,omitempty"`
	Extensions      map[string]any             `json:"-"`
}

// Header describes a response header. It follows the Parameter object without
// name and in.
type Header struct {
	Description     null.String                `json:"description,omitzero"`
	Required        null.Bool                  `json:"required,omitzero"`
	Deprecated      null.Bool                  `json:"deprecated,omitzero"`
	AllowEmptyValue null.Bool                  `json:"allowEmptyValue,omitzero"`
	Style           null.Value[ParameterStyle] `json:"style,omitzero"`
	Explode         null.Bool                  `json:"explode,omitzero"`
	AllowReserved   null.Bool                  `json:"allowReserved,omitzero"`
	Schema          RefT[Schema]               `json:"schema,omitzero"`
	Example         any                        `json:"example,omitempty"`
	Examples        map[string]RefT[Example]   `json:"examples,omitempty"`
	Content         map[string]MediaType       `json:"content,omitempty"`
	Extensions      map[string]any             `json:"-"`
}

// RequestBody describes a single request body.
type RequestBody struct {
	Description null.String          `json:"description,omitzero"`
	Content     map[string]MediaType `json:"content"`
	Required    null.Bool            `json:"required,omitzero"`
	Extensions  map[string]any       `json:"-"`
}

// MediaType describes a request or response body for a given media type.
type MediaType struct {
	Schema     RefT[Schema]             `json:"schema,omitzero"`
	Example    any                      `json:"example,omitempty"`
	Examples   map[string]RefT[Example] `json:"examples,omitempty"`
	Encoding   map[string]Encoding      `json:"encoding,omitempty"`
	Extensions map[string]any           `json:"-"`
}

// Encoding describes a single encoding definition for a request body.
type Encoding struct {
	ContentType   null.String                `json:"contentType,omitzero"`
	Headers       map[string]RefT[Header]    `json:"headers,omitempty"`
	Style         null.Value[ParameterStyle] `json:"style,omitzero"`
	Explode       null.Bool                  `json:"explode,omitzero"`
	AllowReserved null.Bool                  `json:"allowReserved,omitzero"`
	Extensions    map[string]any             `json:"-"`
}

// Responses maps an HTTP status code, or "default", to a Response.
type Responses map[string]RefT[Response]

// Response describes a single response from an API operation.
type Response struct {
	Description string                  `json:"description"`
	Headers     map[string]RefT[Header] `json:"headers,omitempty"`
	Content     map[string]MediaType    `json:"content,omitempty"`
	Links       map[string]RefT[Link]   `json:"links,omitempty"`
	Extensions  map[string]any          `json:"-"`
}

// Link describes a design-time link for a response.
type Link struct {
	OperationRef null.String        `json:"operationRef,omitzero"`
	OperationID  null.String        `json:"operationId,omitzero"`
	Parameters   map[string]any     `json:"parameters,omitempty"`
	RequestBody  any                `json:"requestBody,omitempty"`
	Description  null.String        `json:"description,omitzero"`
	Server       null.Value[Server] `json:"server,omitzero"`
	Extensions   map[string]any     `json:"-"`
}

// Example is an example of a request or response payload.
type Example struct {
	Summary       null.String    `json:"summary,omitzero"`
	Description   null.String    `json:"description,omitzero"`
	Value         any            `json:"value,omitempty"`
	ExternalValue null.String    `json:"externalValue,omitzero"`
	Extensions    map[string]any `json:"-"`
}

// Schema is a JSON Schema 2020-12 schema with the OpenAPI additions.
type Schema struct {
	ID            null.String             `json:"$id,omitzero"`
	Schema        null.String             `json:"$schema,omitzero"`
	Anchor        null.String             `json:"$anchor,omitzero"`
	DynamicRef    null.String             `json:"$dynamicRef,omitzero"`
	DynamicAnchor null.String             `json:"$dynamicAnchor,omitzero"`
	Vocabulary    map[string]bool         `json:"$vocabulary,omitempty"`
	Comment       null.String             `json:"$comment,omitzero"`
	Defs          map[string]RefT[Schema] `json:"$defs,omitempty"`

	AllOf                 []RefT[Schema]          `json:"allOf,omitempty"`
	AnyOf                 []RefT[Schema]          `json:"anyOf,omitempty"`
	OneOf                 []RefT[Schema]          `json:"oneOf,omitempty"`
	Not                   RefT[Schema]            `json:"not,omitzero"`
	If                    RefT[Schema]            `json:"if,omitzero"`
	Then                  RefT[Schema]            `json:"then,omitzero"`
	Else                  RefT[Schema]            `json:"else,omitzero"`
	DependentSchemas      map[string]RefT[Schema] `json:"dependentSchemas,omitempty"`
	PrefixItems           []RefT[Schema]          `json:"prefixItems,omitempty"`
	Items                 RefT[Schema]            `json:"items,omitzero"`
	Contains              RefT[Schema]            `json:"contains,omitzero"`
	Properties            map[string]RefT[Schema] `json:"properties,omitempty"`
	PatternProperties     map[string]RefT[Schema] `json:"patternProperties,omitempty"`
	AdditionalProperties  BoolOrSchema            `json:"additionalProperties,omitzero"`
	PropertyNames         RefT[Schema]            `json:"propertyNames,omitzero"`
	UnevaluatedItems      RefT[Schema]            `json:"unevaluatedItems,omitzero"`
	UnevaluatedProperties BoolOrSchema            `json:"unevaluatedProperties,omitzero"`

	Type              Strings             `json:"type,omitempty"`
	Enum              []any               `json:"enum,omitempty"`
	Const             any                 `json:"const,omitempty"`
	MultipleOf        null.Float          `json:"multipleOf,omitzero"`
	Maximum           null.Float          `json:"maximum,omitzero"`
	ExclusiveMaximum  null.Float          `json:"exclusiveMaximum,omitzero"`
	Minimum           null.Float          `json:"minimum,omitzero"`
	ExclusiveMinimum  null.Float          `json:"exclusiveMinimum,omitzero"`
	MaxLength         null.Int            `json:"maxLength,omitzero"`
	MinLength         null.Int            `json:"minLength,omitzero"`
	Pattern           null.String         `json:"pattern,omitzero"`
	MaxItems          null.Int            `json:"maxItems,omitzero"`
	MinItems          null.Int            `json:"minItems,omitzero"`
	UniqueItems       null.Bool           `json:"uniqueItems,omitzero"`
	MaxContains       null.Int            `json:"maxContains,omitzero"`
	MinContains       null.Int            `json:"minContains,omitzero"`
	MaxProperties     null.Int            `json:"maxProperties,omitzero"`
	MinProperties     null.Int            `json:"minProperties,omitzero"`
	Required          []string            `json:"required,omitempty"`
	DependentRequired map[string][]string `json:"dependentRequired,omitempty"`

	Format           null.String  `json:"format,omitzero"`
	ContentEncoding  null.String  `json:"contentEncoding,omitzero"`
	ContentMediaType null.String  `json:"contentMediaType,omitzero"`
	ContentSchema    RefT[Schema] `json:"contentSchema,omitzero"`
	Title            null.String  `json:"title,omitzero"`
	Description      null.String  `json:"description,omitzero"`
	Default          any          `json:"default,omitempty"`
	Deprecated       null.Bool    `json:"deprecated,omitzero"`
	ReadOnly         null.Bool    `json:"readOnly,omitzero"`
	WriteOnly        null.Bool    `json:"writeOnly,omitzero"`
	Examples         []any        `json:"examples,omitempty"`

	Discriminator null.Value[Discriminator]         `json:"discriminator,omitzero"`
	ExternalDocs  null.Value[ExternalDocumentation] `json:"externalDocs,omitzero"`
	Example       any                               `json:"example,omitempty"` // Deprecated in favor of Examples.

	Extensions map[string]any `json:"-"`
}

// Discriminator helps distinguish between schemas during validation.
type Discriminator struct {
	PropertyName string            `json:"propertyName"`
	Mapping      map[string]string `json:"mapping,omitempty"`
	Extensions   map[string]any    `json:"-"`
}

// BoolOrSchema holds a JSON Schema that is either a boolean or a schema.
type BoolOrSchema struct {
	Bool   null.Bool
	Schema RefT[Schema]
}

// MarshalJSONTo writes the boolean if valid, otherwise the schema.
func (b BoolOrSchema) MarshalJSONTo(enc *jsontext.Encoder) error {
	switch {
	case b.Bool.Valid:
		return json.MarshalEncode(enc, b.Bool.Bool)
	case b.Schema.Ref != nil || b.Schema.Value != nil:
		return json.MarshalEncode(enc, b.Schema)
	default:
		return enc.WriteValue(jsontext.Value("null"))
	}
}

// UnmarshalJSONFrom decodes either a boolean or a schema.
func (b *BoolOrSchema) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	switch dec.PeekKind() {
	case jsontext.KindTrue, jsontext.KindFalse:
		var v bool
		if err := json.UnmarshalDecode(dec, &v); err != nil {
			return err
		}
		*b = BoolOrSchema{Bool: null.BoolFrom(v)}
		return nil
	case jsontext.KindNull:
		*b = BoolOrSchema{}
		return dec.SkipValue()
	default:
		var s RefT[Schema]
		if err := json.UnmarshalDecode(dec, &s); err != nil {
			return err
		}
		*b = BoolOrSchema{Schema: s}
		return nil
	}
}

// Strings holds a JSON Schema type: one type name or several.
type Strings []string

// MarshalJSONTo writes one name as a string and several as an array.
func (s Strings) MarshalJSONTo(enc *jsontext.Encoder) error {
	if len(s) == 1 {
		return json.MarshalEncode(enc, s[0])
	}
	return json.MarshalEncode(enc, []string(s))
}

// UnmarshalJSONFrom accepts either a single string or an array of strings.
func (s *Strings) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	if dec.PeekKind() == jsontext.KindString {
		var one string
		if err := json.UnmarshalDecode(dec, &one); err != nil {
			return err
		}
		*s = Strings{one}
		return nil
	}
	var many []string
	if err := json.UnmarshalDecode(dec, &many); err != nil {
		return err
	}
	*s = many
	return nil
}

// SecuritySchemeType is the type of a security scheme.
type SecuritySchemeType string

const (
	SecuritySchemeTypeAPIKey        SecuritySchemeType = "apiKey"
	SecuritySchemeTypeHTTP          SecuritySchemeType = "http"
	SecuritySchemeTypeMutualTLS     SecuritySchemeType = "mutualTLS"
	SecuritySchemeTypeOAuth2        SecuritySchemeType = "oauth2"
	SecuritySchemeTypeOpenIDConnect SecuritySchemeType = "openIdConnect"
)

// SecuritySchemeIn is the location of an API key.
type SecuritySchemeIn string

const (
	SecuritySchemeInQuery  SecuritySchemeIn = "query"
	SecuritySchemeInHeader SecuritySchemeIn = "header"
	SecuritySchemeInCookie SecuritySchemeIn = "cookie"
)

// SecurityScheme describes a security scheme for the API.
type SecurityScheme struct {
	Type             SecuritySchemeType           `json:"type"`
	Description      null.String                  `json:"description,omitzero"`
	Name             null.String                  `json:"name,omitzero"`
	In               null.Value[SecuritySchemeIn] `json:"in,omitzero"`
	Scheme           null.String                  `json:"scheme,omitzero"`
	BearerFormat     null.String                  `json:"bearerFormat,omitzero"`
	Flows            null.Value[OAuthFlows]       `json:"flows,omitzero"`
	OpenIDConnectURL null.String                  `json:"openIdConnectUrl,omitzero"`
	Extensions       map[string]any               `json:"-"`
}

// OAuthFlows lists the supported OAuth 2.0 flows.
type OAuthFlows struct {
	Implicit          null.Value[OAuthFlow] `json:"implicit,omitzero"`
	Password          null.Value[OAuthFlow] `json:"password,omitzero"`
	ClientCredentials null.Value[OAuthFlow] `json:"clientCredentials,omitzero"`
	AuthorizationCode null.Value[OAuthFlow] `json:"authorizationCode,omitzero"`
	Extensions        map[string]any        `json:"-"`
}

// OAuthFlow is the configuration of a single OAuth 2.0 flow.
type OAuthFlow struct {
	AuthorizationURL null.String       `json:"authorizationUrl,omitzero"`
	TokenURL         null.String       `json:"tokenUrl,omitzero"`
	RefreshURL       null.String       `json:"refreshUrl,omitzero"`
	Scopes           map[string]string `json:"scopes"`
	Extensions       map[string]any    `json:"-"`
}

// SecurityRequirement maps a security scheme name to the scopes required.
type SecurityRequirement map[string][]string

// Reference is an OpenAPI Reference Object.
type Reference struct {
	Ref         string      `json:"$ref"`
	Summary     null.String `json:"summary,omitzero"`
	Description null.String `json:"description,omitzero"`
}

// RefT holds either an inline value of type T or a Reference.
type RefT[T any] struct {
	Ref   *Reference
	Value *T
}

// MarshalJSONTo writes the reference if set, otherwise the inline value.
func (r RefT[T]) MarshalJSONTo(enc *jsontext.Encoder) error {
	switch {
	case r.Ref != nil:
		return json.MarshalEncode(enc, r.Ref)
	case r.Value != nil:
		return json.MarshalEncode(enc, r.Value)
	default:
		return enc.WriteValue(jsontext.Value("null"))
	}
}

// UnmarshalJSONFrom decodes a Reference when a $ref member is present.
func (r *RefT[T]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	if dec.PeekKind() == jsontext.KindNull {
		*r = RefT[T]{}
		return dec.SkipValue()
	}
	if dec.PeekKind() != jsontext.KindBeginObject {
		var v T
		if err := json.UnmarshalDecode(dec, &v); err != nil {
			return err
		}
		*r = RefT[T]{Value: &v}
		return nil
	}

	raw, err := dec.ReadValue()
	if err != nil {
		return err
	}

	var probe struct {
		Ref jsontext.Value `json:"$ref"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return err
	}
	if probe.Ref.Kind() == jsontext.KindString {
		var ref Reference
		if err := json.Unmarshal(raw, &ref); err != nil {
			return err
		}
		*r = RefT[T]{Ref: &ref}
		return nil
	}

	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		return err
	}
	*r = RefT[T]{Value: &v}
	return nil
}
