//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"chichi/apis/postgres"
)

type Transformations struct {
	*Workspace
}

type Transformation struct {
	Connection int
	Source     string
	Inputs     []string
	Output     string
}

// SaveAll saves all the transformations for the given connection.
// Every transformation function must have at least one non-empty input
// property, a not-empty source code and a not-empty output property.
func (this *Transformations) SaveAll(connection int, transformations []Transformation) error {

	// Validate all the transformations.
	for _, t := range transformations {
		err := this.validate(t)
		if err != nil {
			return err
		}
	}

	// TODO(Gianluca):
	// exists, err := this.myDB.Table("Connections").Exists(sql.Where{"id": connection})
	// if err != nil {
	// 	return err
	// }
	// if !exists {
	// 	return ConnectionNotFoundError{}
	// }

	err := this.db.Transaction(func(tx *postgres.Tx) error {

		// Delete the transformations relative to this connection.
		_, err := tx.Exec("DELETE FROM transformations WHERE connection = $1", connection)
		if err != nil {
			return fmt.Errorf("cannot delete transformations for connection %d: %s", connection, err)
		}

		// Create the transformations.
		for _, t := range transformations {
			inputs, err := json.Marshal(t.Inputs)
			if err != nil {
				return err
			}
			_, err = tx.Exec("INSERT INTO transformations (connection, source, inputs, output) VALUES ($1, $2, $3, $4)",
				t.Connection, t.Source, string(inputs), t.Output,
			)
			if err != nil {
				return err
			}
		}

		return nil
	})
	return err
}

// List lists the transformations for the given connection.
func (this *Transformations) List(connection int) ([]Transformation, error) {

	// TODO(Gianluca):
	// exists, err := this.myDB.Table("Connections").Exists(sql.Where{"id": connection})
	// if err != nil {
	// 	return nil, err
	// }
	// if !exists {
	// 	return nil, ConnectionNotFoundError{}
	// }

	rows, err := this.db.Query("SELECT source, inputs, output FROM transformations WHERE connection = $1", connection)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ts := []Transformation{}
	for rows.Next() {
		var source, inputs, output string
		err := rows.Scan(&source, &inputs, &output)
		if err != nil {
			return nil, err
		}
		t := Transformation{
			Source:     source,
			Connection: connection,
			Output:     output,
		}
		err = json.Unmarshal([]byte(inputs), &t.Inputs)
		if err != nil {
			return nil, err
		}
		ts = append(ts, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ts, nil
}

func (this *Transformations) validate(t Transformation) error {
	if strings.TrimSpace(t.Source) == "" {
		return errors.New("transformation function cannot be empty")
	}
	// TODO(Gianluca): validate the Python function here.
	if len(t.Inputs) == 0 {
		return errors.New("should have at least one input property")
	}
	if t.Output == "" {
		return errors.New("output property is mandatory")
	}
	return nil
}
