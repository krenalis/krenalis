//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"chichi/pkg/open2b/sql"
	"errors"
	"fmt"
	"io"

	"github.com/open2b/scriggo"
	"github.com/open2b/scriggo/native"
)

type Transformations struct {
	*WorkspaceAPI
}

type Transformation struct {
	ID         int
	SourceCode string
	// Connection is the connection for this transformation; it matches the
	// connection of every input property.
	Connection      int
	InputProperties []InputProperty
	// GRProperty is the Golden Record property.
	GRProperty string
}

type TransformationToCreate struct {
	SourceCode      string
	InputProperties []InputProperty
	GRProperty      string
}

type TransformationToUpdate = TransformationToCreate

type InputProperty struct {
	Connection int
	Name       string
}

func (this *Transformations) Create(t TransformationToCreate) (int, error) {
	err := this.validate(t)
	if err != nil {
		return 0, err
	}
	var id int
	err = this.myDB.Transaction(func(tx *sql.Tx) error {
		var err error
		id, err = tx.Table("Transformations").Add(map[string]any{
			"sourceCode":       t.SourceCode,
			"goldenRecordName": t.GRProperty,
		}, nil)
		if err != nil {
			return err
		}
		for _, prop := range t.InputProperties {
			_, err := tx.Table("TransformationsConnections").Add(map[string]any{
				"connection":     prop.Connection,
				"property":       prop.Name,
				"transformation": id,
			}, nil)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (this *Transformations) Update(id int, t TransformationToUpdate) error {
	err := this.validate(t)
	if err != nil {
		return err
	}
	err = this.myDB.Transaction(func(tx *sql.Tx) error {
		var err error
		_, err = tx.Table("Transformations").Update(map[string]any{
			"sourceCode":       t.SourceCode,
			"goldenRecordName": t.GRProperty,
		}, sql.Where{"id": id})
		if err != nil {
			return err
		}
		_, err = tx.Table("TransformationsConnections").Delete(sql.Where{"transformation": id})
		if err != nil {
			return err
		}
		for _, prop := range t.InputProperties {
			_, err := tx.Table("TransformationsConnections").Add(map[string]any{
				"connection":     prop.Connection,
				"property":       prop.Name,
				"transformation": id,
			}, nil)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

// List lists the transformations for the given connection.
func (this *Transformations) List(connection int) ([]Transformation, error) {
	var transformations []Transformation
	err := this.myDB.Transaction(func(tx *sql.Tx) error {
		var transfIDs []int
		transfProps := map[int][]InputProperty{}
		rows, err := tx.Table("TransformationsConnections").Select(
			[]any{"property", "transformation"},
			sql.Where{"connection": connection},
			[]any{"connection", "property"},
			0, 0,
		).Rows()
		if err != nil {
			return err
		}
		for _, row := range rows {
			tID := row["transformation"].(int)
			transfIDs = append(transfIDs, tID)
			transfProps[tID] = append(transfProps[tID], InputProperty{
				Connection: connection,
				Name:       row["property"].(string),
			})
		}
		if len(transfIDs) == 0 {
			transformations = []Transformation{}
			return nil
		}
		rows, err = tx.Table("Transformations").Select(
			[]any{"id", "goldenRecordName", "sourceCode"},
			sql.Where{"id": transfIDs},
			nil, 0, 0,
		).Rows()
		if err != nil {
			return err
		}
		for _, row := range rows {
			id := row["id"].(int)
			transformations = append(transformations, Transformation{
				ID:              id,
				SourceCode:      row["sourceCode"].(string),
				Connection:      connection,
				InputProperties: transfProps[id],
				GRProperty:      row["goldenRecordName"].(string),
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return transformations, nil
}

func (this *Transformations) validate(t TransformationToCreate) error {
	_, err := buildTransfFunc(t.SourceCode)
	if err != nil {
		return fmt.Errorf("source code is invalid: %s", err)
	}
	if len(t.InputProperties) == 0 {
		return errors.New("should have at least one input property")
	}
	c := t.InputProperties[0].Connection
	for _, p := range t.InputProperties[1:] {
		if p.Connection != c {
			return errors.New("every input property should refer to the same connection")
		}
	}
	if t.GRProperty == "" {
		return errors.New("output property is mandatory")
	}
	return nil
}

// TransformationFuncType is the type of a transformation function.
type TransformationFuncType = func(props map[string]any) (prop any, ok bool, err error)

// buildTransfFunc builds a transformation function from its source code and
// returns it.
func buildTransfFunc(source string) (TransformationFuncType, error) {
	if source == "" {
		return nil, errors.New("transformation function source cannot be empty")
	}
	src := `{% Fn = ` + source + ` %}`
	opts := &scriggo.BuildOptions{
		Globals: native.Declarations{
			"Fn": (*TransformationFuncType)(nil),
		},
	}
	fs := scriggo.Files{"transform.txt": []byte(src)}
	template, err := scriggo.BuildTemplate(fs, "transform.txt", opts)
	if err != nil {
		return nil, err
	}
	var fn TransformationFuncType
	vars := map[string]interface{}{
		"Fn": &fn,
	}
	err = template.Run(io.Discard, vars, nil)
	if err != nil {
		return nil, err
	}
	return fn, nil
}
