// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/krenalis/krenalis/tools/json"
)

const maxInternationalDocumentBytes = 8 << 20

// internationalDecoder reads only the closed transport types in
// international_types.go. Reflection shares required-field, exact-integer and
// null handling without maintaining a second, potentially divergent schema.
// It never decodes through interface{} or floating point.
type internationalDecoder struct {
	decoder *json.Decoder
}

func (d *internationalDecoder) read(out reflect.Value, maximum, valueMaximum int, nullable bool) error {

	if out.Kind() == reflect.Pointer {
		if d.decoder.PeekKind() == json.Null && nullable {
			return d.token(json.Null)
		}
		out.Set(reflect.New(out.Type().Elem()))
		return d.read(out.Elem(), maximum, valueMaximum, false)
	}

	switch out.Kind() {

	case reflect.Struct:
		err := d.token(json.Object)
		if err != nil {
			return err
		}

		typ := out.Type()
		seen := make([]bool, typ.NumField())
		for d.decoder.PeekKind() != '}' {

			token, err := d.decoder.ReadToken()
			if err != nil {
				return err
			}
			field := -1
			for i := range typ.NumField() {
				name, _, _ := strings.Cut(typ.Field(i).Tag.Get("json"), ",")
				if token.Kind() == json.String && token.String() == name {
					field = i
					break
				}
			}
			if field < 0 || seen[field] {
				return fmt.Errorf("unknown or duplicate field at byte %d", d.decoder.ByteOffset())
			}

			seen[field] = true
			definition := typ.Field(field)
			limit := 0
			if value := definition.Tag.Get("max"); value != "" {
				limit, err = strconv.Atoi(value)
				if err != nil {
					return fmt.Errorf("invalid internal collection bound")
				}
			}
			valueLimit := 8
			if value := definition.Tag.Get("maxvalue"); value != "" {
				valueLimit, err = strconv.Atoi(value)
				if err != nil {
					return fmt.Errorf("invalid internal map value bound")
				}
			}
			err = d.read(out.Field(field), limit, valueLimit, definition.Tag.Get("nullable") == "true")
			if err != nil {
				return fmt.Errorf("%s: %w", definition.Name, err)
			}

		}

		for i, present := range seen {
			if !present && !strings.HasSuffix(typ.Field(i).Tag.Get("json"), ",omitempty") {
				return fmt.Errorf("missing field %s", typ.Field(i).Name)
			}
		}
		return d.token('}')

	case reflect.Slice:
		err := d.token(json.Array)
		if err != nil {
			return err
		}
		if maximum <= 0 {
			return fmt.Errorf("missing internal collection bound")
		}
		out.Set(reflect.MakeSlice(out.Type(), 0, min(maximum, 16)))
		for d.decoder.PeekKind() != ']' {
			if out.Len() == maximum {
				return fmt.Errorf("array exceeds %d entries", maximum)
			}
			entry := reflect.New(out.Type().Elem()).Elem()
			err = d.read(entry, 0, 0, nullable)
			if err != nil {
				return fmt.Errorf("entry %d: %w", out.Len(), err)
			}
			out.Set(reflect.Append(out, entry))
		}
		return d.token(']')

	case reflect.Map:
		// The only free-key object is the typed postal-code map.
		err := d.token(json.Object)
		if err != nil {
			return err
		}
		out.Set(reflect.MakeMap(out.Type()))
		for d.decoder.PeekKind() != '}' {
			if out.Len() == maximum {
				return fmt.Errorf("postal map exceeds %d places", maximum)
			}
			key := reflect.New(out.Type().Key()).Elem()
			err = d.read(key, 0, 0, false)
			if err != nil {
				return err
			}
			if out.MapIndex(key).IsValid() {
				return fmt.Errorf("duplicate postal place")
			}
			entry := reflect.New(out.Type().Elem()).Elem()
			err = d.read(entry, valueMaximum, 0, false)
			if err != nil {
				return err
			}
			out.SetMapIndex(key, entry)
		}
		return d.token('}')

	case reflect.String:
		token, err := d.decoder.ReadToken()
		if err != nil {
			return err
		}
		if token.Kind() != json.String || !internationalText(token.String()) {
			return fmt.Errorf("expected nonempty Text at byte %d", d.decoder.ByteOffset())
		}
		out.SetString(token.String())
		return nil

	case reflect.Uint32, reflect.Uint64:
		token, err := d.decoder.ReadToken()
		if err != nil {
			return err
		}
		if token.Kind() != json.Number {
			return fmt.Errorf("expected unsigned integer at byte %d", d.decoder.ByteOffset())
		}
		value, err := strconv.ParseUint(token.String(), 10, out.Type().Bits())
		if err != nil {
			return fmt.Errorf("invalid unsigned integer at byte %d", d.decoder.ByteOffset())
		}
		out.SetUint(value)
		return nil

	default:
		return fmt.Errorf("unsupported internal transport type")

	}

}

func (d *internationalDecoder) token(kind json.Kind) error {
	token, err := d.decoder.ReadToken()
	if err != nil {
		return err
	}
	if token.Kind() != kind {
		return fmt.Errorf("expected %s at byte %d", kind, d.decoder.ByteOffset())
	}
	return nil
}

type internationalReader struct {
	ctx    context.Context
	reader io.Reader
}

// Read checks cancellation before reading the next input chunk.
func (r *internationalReader) Read(p []byte) (int, error) {
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	default:
		return r.reader.Read(p)
	}
}

// appendInternationalCanonical implements C1, not RFC 8785: exact U64 tokens,
// scalar-order keys, literal non-ASCII, prescribed escapes and array order.
// UTF-8 byte order equals scalar order for valid Unicode strings.
func appendInternationalCanonical(dst []byte, decoder *json.Decoder, depth int) ([]byte, error) {

	if depth > 32 {
		return nil, fmt.Errorf("JSON nesting exceeds 32 levels")
	}
	token, err := decoder.ReadToken()
	if err != nil {
		return nil, err
	}

	switch token.Kind() {
	case json.Object:
		fields := map[string][]byte{}
		var keys []string
		for decoder.PeekKind() != '}' {
			key, err := decoder.ReadToken()
			if err != nil {
				return nil, err
			}
			name := key.String()
			value, err := appendInternationalCanonical(nil, decoder, depth+1)
			if err != nil {
				return nil, err
			}
			keys = append(keys, name)
			fields[name] = value
		}
		_, err = decoder.ReadToken()
		if err != nil {
			return nil, err
		}
		slices.Sort(keys)
		dst = append(dst, '{')
		for i, key := range keys {
			if i > 0 {
				dst = append(dst, ',')
			}
			dst = appendInternationalString(dst, key)
			dst = append(dst, ':')
			dst = append(dst, fields[key]...)
		}
		return append(dst, '}'), nil
	case json.Array:
		dst = append(dst, '[')
		for i := 0; decoder.PeekKind() != ']'; i++ {
			if i > 0 {
				dst = append(dst, ',')
			}
			dst, err = appendInternationalCanonical(dst, decoder, depth+1)
			if err != nil {
				return nil, err
			}
		}
		_, err = decoder.ReadToken()
		return append(dst, ']'), err
	case json.String:
		return appendInternationalString(dst, token.String()), nil
	case json.Number:
		n, err := strconv.ParseUint(token.String(), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("C1 requires unsigned integers")
		}
		return strconv.AppendUint(dst, n, 10), nil
	case json.Null:
		return append(dst, "null"...), nil
	default:
		return nil, fmt.Errorf("unsupported C1 value")
	}

}

func appendInternationalString(dst []byte, value string) []byte {
	dst = append(dst, '"')
	for _, c := range value {
		switch c {
		case '"', '\\':
			dst = append(dst, '\\', byte(c))
		case '\b':
			dst = append(dst, `\b`...)
		case '\t':
			dst = append(dst, `\t`...)
		case '\n':
			dst = append(dst, `\n`...)
		case '\f':
			dst = append(dst, `\f`...)
		case '\r':
			dst = append(dst, `\r`...)
		default:
			if c < 0x20 {
				const hex = "0123456789abcdef"
				dst = append(dst, '\\', 'u', '0', '0', hex[c>>4], hex[c&15])
			} else {
				dst = append(dst, string(c)...)
			}
		}
	}
	return append(dst, '"')
}

func canonicalInternationalJSON(data []byte) ([]byte, error) {
	if len(data) > maxInternationalDocumentBytes {
		return nil, fmt.Errorf("international document exceeds byte limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	if decoder.PeekKind() != json.Object {
		return nil, fmt.Errorf("C1 document must be an object")
	}
	canonical, err := appendInternationalCanonical(nil, decoder, 0)
	if err != nil {
		return nil, err
	}
	_, err = decoder.ReadToken()
	if err != nil {
		if err == io.EOF {
			return append(canonical, '\n'), nil
		}
		return nil, err
	}
	return nil, fmt.Errorf("multiple JSON documents")
}

func decodeInternationalJSON(ctx context.Context, reader io.Reader, out any) error {
	if ctx == nil || reader == nil {
		return fmt.Errorf("context and reader are required")
	}
	source := &internationalReader{ctx: ctx, reader: reader}
	bounded := &io.LimitedReader{R: source, N: maxInternationalDocumentBytes + 1}
	d := internationalDecoder{decoder: json.NewDecoder(bounded)}
	err := d.read(reflect.ValueOf(out).Elem(), 0, 0, false)
	if err != nil {
		return err
	}
	_, err = d.decoder.ReadToken()
	if err != nil {
		if err != io.EOF {
			return err
		}
	}
	if err == nil || bounded.N == 0 {
		return fmt.Errorf("multiple or oversized international document")
	}
	return nil
}

func internationalCanonical(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return canonicalInternationalJSON(data)
}

func internationalDigest(domain string, document any, components ...Component) ([32]byte, error) {
	canonical, err := internationalCanonical(document)
	if err != nil {
		return [32]byte{}, err
	}
	name := "document"
	if len(components) != 0 {
		name = "data"
	}
	component, err := BytesComponent(name, canonical)
	if err != nil {
		return [32]byte{}, err
	}
	return canonicalDigest(domain, append(components, component))
}

func internationalText(value string) bool {
	if value == "" || len(value) > 4096 {
		return false
	}
	for _, c := range value {
		if c < 0x20 || c == 0x7f {
			return false
		}
	}
	return true
}

func internationalToken(value string) bool {
	if len(value) == 0 || len(value) > 128 {
		return false
	}
	for i, c := range []byte(value) {
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' {
			continue
		}
		if i == 0 || !strings.ContainsRune("._:-", rune(c)) {
			return false
		}
	}
	return true
}
