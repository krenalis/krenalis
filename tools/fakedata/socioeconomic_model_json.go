// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"fmt"
	"io"
	"slices"
	"strconv"

	"github.com/krenalis/krenalis/tools/json"
)

var socioeconomicAgeNames = [...]string{
	"Education", "PrimaryActivity", "Employee.JobLevel",
	"Income[s]", "Income[u]", "Income[t]", "Income[r]", "Income[i]",
}

var socioeconomicBranchNames = [...]string{"e", "s", "u", "t", "r", "i"}

// The transport uses the closed logical-v0 schema. Dimensions are positional
// in D2-A order, not category strings or source-publication metadata.
type socioeconomicModelJSON struct {
	companyCount          uint64
	ages                  [8]socioeconomicAgeJSON
	companyIndustry       []uint32
	employeeCount         []socioeconomicLawJSON
	education             []uint32
	primaryActivity       []uint32
	employeeJobFamily     []uint32
	employeeJobLevel      []uint32
	selfEmployedJobFamily []uint32
	selfEmployedIndustry  []uint32
	employer              socioeconomicEmployerParameters
	income                [6]socioeconomicIncomeJSON
}

type socioeconomicAgeJSON struct {
	lowerBounds []uint32
	compiled    []uint32
}

type socioeconomicIncomeJSON struct {
	tupleMap []uint32
	laws     []socioeconomicLawJSON
}

type socioeconomicLawJSON struct {
	values  []uint32
	weights []uint32
}

// Streaming bounds every collection before allocation and retains no raw JSON.
// Even simultaneous schema maxima fit D2-A's 8 MiB decoded-work limit:
// 12,194 categorical weights, 1,792 count points, 524,288 Income points,
// 76,160 tuple entries, age maps, factors and law descriptors total 4,749,760
// bytes including slice capacity/overhead. A further 64 KiB covers bounded
// tokens, decoder state and fixed structures (4,815,296 bytes altogether).
type socioeconomicDecoder struct {
	decoder *json.Decoder
}

func (d *socioeconomicDecoder) age(out *socioeconomicAgeJSON, maximum int) error {
	return d.object([]string{"lowerBounds", "compiled"}, func(field int) error {
		var err error
		if field == 0 {
			out.lowerBounds, err = d.vector(maximum, 75)
		} else {
			out.compiled, err = d.vector(55, uint64(maximum-1))
		}
		return err
	})
}

func (d *socioeconomicDecoder) laws(maximum, points int) ([]socioeconomicLawJSON, error) {

	err := d.token(json.Array)
	if err != nil {
		return nil, err
	}
	laws := make([]socioeconomicLawJSON, 0, maximum)
	for d.decoder.PeekKind() != ']' {

		if len(laws) == maximum {
			return nil, fmt.Errorf("law count exceeds %d", maximum)
		}

		var law socioeconomicLawJSON
		err = d.object([]string{"values", "weights"}, func(field int) error {
			var err error
			if field == 0 {
				law.values, err = d.vector(points, 1<<32-1)
			} else {
				law.weights, err = d.vector(points, 1<<32-1)
			}
			return err
		})
		if err != nil {
			return nil, fmt.Errorf("law %d: %w", len(laws), err)
		}
		laws = append(laws, law)

	}
	err = d.token(']')

	return laws, err
}

// object enforces the complete field set before semantic validation. Its field
// names are fixed schema literals; untrusted names are never copied into errors.
func (d *socioeconomicDecoder) object(fields []string, read func(int) error) error {

	err := d.token(json.Object)
	if err != nil {
		return err
	}
	var seen uint32
	for d.decoder.PeekKind() != '}' {

		token, err := d.decoder.ReadToken()
		if err != nil {
			return fmt.Errorf("invalid JSON field at byte %d", d.decoder.ByteOffset())
		}
		field := slices.Index(fields, token.String())
		if token.Kind() != json.String || field < 0 {
			return fmt.Errorf("unknown field at byte %d", d.decoder.ByteOffset())
		}

		seen |= uint32(1) << field
		err = read(field)
		if err != nil {
			return fmt.Errorf("%s: %w", fields[field], err)
		}

	}
	err = d.token('}')
	if err != nil {
		return err
	}
	for field, name := range fields {
		if seen&(uint32(1)<<field) == 0 {
			return fmt.Errorf("missing field %s", name)
		}
	}

	return nil
}

func (d *socioeconomicDecoder) token(kind json.Kind) error {
	token, err := d.decoder.ReadToken()
	if err != nil {
		return fmt.Errorf("invalid JSON at byte %d", d.decoder.ByteOffset())
	}
	if token.Kind() != kind {
		return fmt.Errorf("expected %s at byte %d", kind, d.decoder.ByteOffset())
	}
	return nil
}

func (d *socioeconomicDecoder) unsigned(maximum uint64) (uint64, error) {

	token, err := d.decoder.ReadToken()
	if err != nil {
		return 0, fmt.Errorf("invalid JSON number at byte %d", d.decoder.ByteOffset())
	}
	if token.Kind() != json.Number {
		return 0, fmt.Errorf("expected unsigned integer at byte %d", d.decoder.ByteOffset())
	}

	// ParseUint deliberately rejects decimal/exponent spellings and negatives.
	value, err := strconv.ParseUint(token.String(), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid unsigned integer at byte %d", d.decoder.ByteOffset())
	}
	if value > maximum {
		return 0, fmt.Errorf("integer %d exceeds %d", value, maximum)
	}

	return value, nil
}

func (d *socioeconomicDecoder) vector(maximum int, limit uint64) ([]uint32, error) {

	err := d.token(json.Array)
	if err != nil {
		return nil, err
	}

	values := make([]uint32, 0, maximum)
	for d.decoder.PeekKind() != ']' {
		if len(values) == maximum {
			return nil, fmt.Errorf("array exceeds %d entries", maximum)
		}
		value, err := d.unsigned(limit)
		if err != nil {
			return nil, fmt.Errorf("entry %d: %w", len(values), err)
		}
		values = append(values, uint32(value))
	}
	err = d.token(']')

	return values, err
}

func (d *socioeconomicDecoder) weights(out *[]uint32, maximum int) error {
	return d.object([]string{"weights"}, func(int) error {
		var err error
		*out, err = d.vector(maximum, 1<<32-1)
		return err
	})
}

// Bound tokens and whitespace runs before the decoder can buffer them. The
// canonical format is compact and has no free text; 192 bytes accommodates
// escaped spellings of every legal key and ordinary pretty-printed variants.
// Syntax, escaping, duplicate names and UTF-8 remain tools/json's responsibility.
type socioeconomicJSONReader struct {
	reader  io.Reader
	err     error
	quoted  bool
	escaped bool
	length  int
	spaces  int
}

// Read bounds lexical buffering without retaining the document.
func (r *socioeconomicJSONReader) Read(p []byte) (int, error) {

	if r.err != nil {
		return 0, r.err
	}
	n, err := r.reader.Read(p)
	r.err = err
	for i, b := range p[:n] {

		if r.quoted {
			r.length++
			if r.escaped {
				r.escaped = false
			} else if b == '\\' {
				r.escaped = true
			} else if b == '"' {
				r.quoted = false
			}
		} else {
			switch b {
			case '"':
				r.quoted = true
				r.length = 0
				r.spaces = 0
			case ' ', '\t', '\r', '\n':
				r.length = 0
				r.spaces++
			case '{', '}', '[', ']', ':', ',':
				r.length = 0
				r.spaces = 0
			default:
				r.length++
				r.spaces = 0
			}
		}

		if r.length > 192 || r.spaces > 192 {
			r.err = fmt.Errorf("model JSON token or whitespace run exceeds 192 bytes")
			return i, r.err
		}

	}

	return n, err
}

// decodeSocioeconomicModel accepts only the canonical logical-v0 representation,
// not publication evidence or an authored pre-canonicalization model.
func decodeSocioeconomicModel(reader io.Reader) (*socioeconomicModel, error) {

	bounded := &io.LimitedReader{R: reader, N: maxSocioeconomicModelBytes + 1}
	d := socioeconomicDecoder{decoder: json.NewDecoder(&socioeconomicJSONReader{reader: bounded})}
	var document socioeconomicModelJSON
	fields := []string{
		"schema", "companyCount", "ages", "companyIndustry", "employeeCount", "education", "primaryActivity",
		"employeeJobFamily", "employeeJobLevel", "selfEmployedJobFamily", "selfEmployedIndustry", "employer", "income",
	}
	err := d.object(fields, func(field int) error {

		switch fields[field] {

		case "schema":
			token, err := d.decoder.ReadToken()
			if err != nil {
				return fmt.Errorf("invalid schema token")
			}
			if token.Kind() != json.String || token.String() != "SW-L2-D2B2-logical-v0" {
				return fmt.Errorf("unsupported model schema")
			}
		case "companyCount":
			var err error
			document.companyCount, err = d.unsigned(104855)
			return err
		case "ages":
			return d.object(socioeconomicAgeNames[:], func(i int) error {
				maximum := 6
				if i == 1 {
					maximum = 8
				}
				return d.age(&document.ages[i], maximum)
			})
		case "companyIndustry":
			return d.weights(&document.companyIndustry, 14)
		case "employeeCount":
			return d.object([]string{"laws"}, func(int) error {
				var err error
				document.employeeCount, err = d.laws(14, 128)
				return err
			})
		case "education":
			return d.weights(&document.education, 840)
		case "primaryActivity":
			return d.weights(&document.primaryActivity, 6720)
		case "employeeJobFamily":
			return d.weights(&document.employeeJobFamily, 1400)
		case "employeeJobLevel":
			return d.weights(&document.employeeJobLevel, 1680)
		case "selfEmployedJobFamily":
			return d.weights(&document.selfEmployedJobFamily, 1400)
		case "selfEmployedIndustry":
			return d.weights(&document.selfEmployedIndustry, 140)
		case "employer":
			return d.object([]string{"cap", "industryFactors", "sizeFactors"}, func(i int) error {

				if i == 0 {
					value, err := d.unsigned(65535)
					if err != nil {
						return err
					}
					document.employer.cap = uint16(value)
					return nil
				}

				out := document.employer.industryFactors[:]
				if i == 2 {
					out = document.employer.sizeFactors[:]
				}
				values, err := d.vector(len(out), 4095)
				if err != nil {
					return err
				}
				if len(values) != len(out) {
					return fmt.Errorf("expected %d factors, got %d", len(out), len(values))
				}
				for j, value := range values {
					out[j] = uint16(value)
				}

				return nil
			})
		case "income":
			return d.object(socioeconomicBranchNames[:], func(branch int) error {
				return d.object([]string{"tupleMap", "laws"}, func(i int) error {
					var err error
					if i == 0 {
						maximum := [...]int{56000, 16800, 840, 840, 840, 840}
						document.income[branch].tupleMap, err = d.vector(maximum[branch], 65535)
					} else {
						maximum := [...]int{512, 256, 64, 64, 64, 64}
						document.income[branch].laws, err = d.laws(maximum[branch], 512)
					}
					return err
				})
			})

		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("it-2026a model: %w", err)
	}
	_, err = d.decoder.ReadToken()
	if err != nil {
		if err != io.EOF {
			return nil, fmt.Errorf("invalid JSON after model at byte %d", d.decoder.ByteOffset())
		}
	}
	if err == nil {
		return nil, fmt.Errorf("multiple JSON values in model")
	}
	if bounded.N == 0 {
		return nil, fmt.Errorf("model exceeds %d decoded bytes", maxSocioeconomicModelBytes)
	}

	return compileSocioeconomicModel(&document)
}
