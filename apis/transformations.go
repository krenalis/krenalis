//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"encoding/json"
	"strings"

	"chichi/apis/errors"
	"chichi/apis/postgres"
	"chichi/apis/types"
)

type Transformations struct {
	*Workspace
	state *transformationsState
}

// newTransformations returns a new Transformations for the given workspace.
func newTransformations(ws *Workspace, state *transformationsState) *Transformations {
	return &Transformations{
		Workspace: ws,
		state:     state,
	}
}

// Transformation represents a transformation from a kind of properties to
// another.
//
// In particular, if the transformation refers to a source connection, it is a
// transformation from the connection properties to a property of the Golden
// Record; otherwise, if it refers to a destination connection, it is a
// transformation from the Golden Record properties to a connection property.
type Transformation struct {

	// ID is the identifier of the transformation. When representing a
	// transformation to create, this is zero.
	ID int

	// Connection is the connection.
	Connection int

	// In is the schema of the input properties of the transformation. If the
	// connection is a source then the properties are the properties of the
	// connection, otherwise, if it is a destination, it contains the properties
	// of the Golden Record.
	//
	// This is the schema of the transformation.
	//
	In types.Schema

	// SourceCode is the source code of the transformation function, which
	// should be something like:
	//
	//   def transform(user):
	//     return user["first_name"]
	//
	SourceCode string

	// Out is the output property of the transformation. If the connection is a
	// source then this is a Golden Record property, otherwise, if it is a
	// destination, it is one of the properties of the connection.
	Out string
}

// sets sets the transformations for the given connection.
// If transformations is nil, then every transformation associated to connection
// is removed.
func (this *Transformations) set(connection int, transformations []*Transformation) {
	this.state.Lock()
	if transformations == nil {
		delete(this.state.ofConnection, connection)
	} else {
		this.state.ofConnection[connection] = transformations
	}
	this.state.Unlock()
}

// Set sets the transformations for the given connection.
// TODO(marco): document errors.
func (this *Transformations) Set(connection int, transformations []*Transformation) error {

	if connection < 1 || connection > maxInt32 {
		return errors.BadRequest("connection identifier %d is not valid", connection)
	}

	// Validate the transformations.
	for _, t := range transformations {
		// TODO(Gianluca): validate the Python function here.
		if strings.TrimSpace(t.SourceCode) == "" {
			return errors.BadRequest("a transformation source code is empty")
		}
		if !t.In.Valid() {
			return errors.BadRequest("schema is invalid")
		}
		props := t.In.PropertiesNames()
		if len(props) == 0 {
			return errors.BadRequest("should have at least one input property")
		}
		for _, in := range props {
			if !types.IsValidPropertyName(in) {
				return errors.BadRequest("input property name %q is not valid", in)
			}
		}
		if !types.IsValidPropertyName(t.Out) {
			return errors.BadRequest("output property name %q is not valid", t.Out)
		}
	}

	n := setConnectionTransformations{
		Connection:      connection,
		Transformations: transformations,
	}

	// Marshal the schemas into JSON.
	schemas := make([][]byte, len(transformations))
	for i, t := range transformations {
		var err error
		schemas[i], err = json.Marshal(t.In)
		if err != nil {
			return err
		}
	}

	err := this.db.Transaction(func(tx *postgres.Tx) error {
		_, err := tx.Exec("DELETE FROM transformations WHERE connection = $1", n.Connection)
		if err != nil {
			return err
		}
		query, err := tx.Prepare("INSERT INTO transformations\n" +
			"(connection, \"in\", source_code, out)\n" +
			"VALUES ($1, $2, $3, $4) RETURNING id")
		if err != nil {
			if err != nil {
				if postgres.IsForeignKeyViolation(err) {
					if postgres.ErrConstraintName(err) == "transformations_connection_fkey" {
						err = errors.NotFound("connection %d does not exist", connection)
					}
				}
			}
			return err
		}
		for i, t := range n.Transformations {
			var id int
			err := query.QueryRow(t.Connection, schemas[i], t.SourceCode, t.Out).Scan(&id)
			if err != nil {
				return err
			}
			t.ID = id
		}
		return tx.Notify(n)
	})

	return err
}

// List lists the transformations for the given connection.
// If the connection does not exist, it returns an errors.NotFoundError error.
func (this *Transformations) List(connection int) ([]*Transformation, error) {
	if connection < 1 || connection > maxInt32 {
		return nil, errors.BadRequest("connection identifier %d is not valid", connection)
	}
	if _, ok := this.Workspace.Connections.state.Get(connection); !ok {
		return nil, errors.NotFound("connection %d does not exist", connection)
	}
	ts := this.state.List(connection)
	if ts == nil {
		// No transformations associated to this connection.
		return []*Transformation{}, nil
	}
	return ts, nil
}
