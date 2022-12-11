//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"errors"
	"fmt"
	"strings"

	"chichi/apis/postgres"
)

type Transformations struct {
	*Workspace
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

// createTransformation creates the transformation t on the tx SQL transaction.
// If the transformation is created successfully, its ID is returned.
func createTransformation(tx *postgres.Tx, t TransformationToCreate) (int, error) {
	var id int
	err := tx.QueryRow("INSERT INTO transformations (source_code, golden_record_name) VALUES ($1, $2) RETURNING id",
		t.SourceCode, t.GRProperty).Scan(&id)
	if err != nil {
		return 0, err
	}
	for _, prop := range t.InputProperties {
		_, err = tx.Exec("INSERT INTO transformations_connections (connection, property, transformation) VALUES ($1, $2, $3)",
			prop.Connection, prop.Name, id)
		if err != nil {
			return 0, err
		}
	}
	return int(id), nil
}

// Create creates the transformation t. If the transformation is created
// successfully, its ID is returned.
//
// The transformation function must have at least one non-empty input property,
// a not-empty source code and a not-empty output property.
func (this *Transformations) Create(t TransformationToCreate) (int, error) {
	err := this.validate(t)
	if err != nil {
		return 0, err
	}
	var id int
	err = this.db.Transaction(func(tx *postgres.Tx) error {
		var err error
		id, err = createTransformation(tx, t)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return id, nil
}

// Update updates the transformation function with the given ID.
//
// The updated transformation function must have at least one non-empty input
// property, a not-empty source code and a not-empty output property.
func (this *Transformations) Update(id int, t TransformationToUpdate) error {
	err := this.validate(t)
	if err != nil {
		return err
	}
	err = this.db.Transaction(func(tx *postgres.Tx) error {
		_, err = tx.Exec("UPDATE transformations\nSET source_code = $1, golden_record_name = $2 WHERE id = $3",
			t.SourceCode, t.GRProperty, id)
		if err != nil {
			return err
		}
		_, err = tx.Exec("DELETE FROM transformations_connections WHERE transformation = $1", id)
		if err != nil {
			return err
		}
		for _, prop := range t.InputProperties {
			_, err = tx.Exec("INSERT INTO transformations_connections (connection, property, transformation)\n"+
				"VALUES($1, $2, $3)", prop.Connection, prop.Name, id)
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

// SaveAll saves all the transformations for the given connection.
// Every transformation function must have at least one non-empty input
// property, a not-empty source code and a not-empty output property.
func (this *Transformations) SaveAll(connection int, transformations []TransformationToUpdate) error {

	// Validate all the transformations.
	for _, t := range transformations {
		err := this.validate(t)
		if err != nil {
			return err
		}
	}

	err := this.db.Transaction(func(tx *postgres.Tx) error {

		// Retrieve the IDs of the transformations to delete.
		var toDelete []int
		err := tx.QueryScan("SELECT transformation FROM transformations_connections WHERE connection = $1", connection,
			func(rows *postgres.Rows) error {
				for rows.Next() {
					var transformation int
					if err := rows.Scan(&transformation); err != nil {
						return err
					}
					toDelete = append(toDelete, transformation)
				}
				return nil
			})
		if err != nil {
			return err
		}

		// Delete the transformations and their connections.
		if len(toDelete) > 0 {
			_, err = tx.Exec("DELETE FROM transformations WHERE id IN " + postgres.QuoteValue(toDelete))
			if err != nil {
				return fmt.Errorf("cannot delete transformations: %s", err)
			}
			_, err = tx.Exec("DELETE FROM transformations_connections WHERE connection IN " + postgres.QuoteValue(toDelete))
			if err != nil {
				return fmt.Errorf("cannot delete connections: %s", err)
			}
		}

		// Create the transformations.
		for _, t := range transformations {
			_, err = createTransformation(tx, t)
			if err != nil {
				return fmt.Errorf("cannot create transformation: %s", err)
			}
		}

		return nil
	})
	return err
}

// List lists the transformations for the given connection.
func (this *Transformations) List(connection int) ([]Transformation, error) {
	var transformations []Transformation
	err := this.db.Transaction(func(tx *postgres.Tx) error {
		var transfIDs []int
		transfProps := map[int][]InputProperty{}
		stmt := "SELECT property, transformation FROM transformations_connections WHERE connection = $1 ORDER BY connection, property"
		err := tx.QueryScan(stmt, connection, func(rows *postgres.Rows) error {
			for rows.Next() {
				var property string
				var transformation int
				if err := rows.Scan(&property, &transformation); err != nil {
					return err
				}
				transfIDs = append(transfIDs, transformation)
				transfProps[transformation] = append(transfProps[transformation], InputProperty{
					Connection: connection,
					Name:       property,
				})
			}
			return nil
		})
		if err != nil {
			return err
		}
		if transfIDs == nil {
			transformations = []Transformation{}
			return nil
		}
		stmt = "SELECT id, golden_record_name, source_code FROM Transformations WHERE id IN " + postgres.QuoteValue(transfIDs)
		err = tx.QueryScan(stmt, func(rows *postgres.Rows) error {
			for rows.Next() {
				var id int
				var record, source string
				if err := rows.Scan(&id, &record, &source); err != nil {
					return err
				}
				transformations = append(transformations, Transformation{
					ID:              id,
					SourceCode:      source,
					Connection:      connection,
					InputProperties: transfProps[id],
					GRProperty:      record})
			}
			return nil
		})
		return err
	})
	if err != nil {
		return nil, err
	}
	return transformations, nil
}

func (this *Transformations) validate(t TransformationToCreate) error {
	if strings.TrimSpace(t.SourceCode) == "" {
		return errors.New("transformation function cannot be empty")
	}
	// TODO(Gianluca): validate the Python function here.
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
