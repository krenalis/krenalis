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
	"sync"

	"chichi/apis/postgres"
	"chichi/apis/types"
)

type Transformations struct {
	*Workspace
	state transformationsState
}

// newTransformations returns a new Transformations for the given workspace.
func newTransformations(ws *Workspace) *Transformations {
	return &Transformations{
		Workspace: ws,
		state: transformationsState{
			ofConnection: map[int][]*Transformation{},
		},
	}
}

type transformationsState struct {
	sync.Mutex
	ofConnection map[int][]*Transformation
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

	// In contains the input properties of the transformation. If the connection
	// is a source then the properties are the properties of the connection,
	// otherwise, if it is a destination, it contains the properties of the
	// Golden Record.
	In InputProperties

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

// InputProperties represents the input properties of a transformation.
type InputProperties []string

// Scan implements the sql.Scanner interface.
// TODO(Gianluca): this is just a stub of the implementation and may not be
// correct, for example, when some array's element contains commas. Please refer
// to https://www.postgresql.org/docs/current/arrays.html to get more details.
func (props *InputProperties) Scan(src any) error {
	s, ok := src.(string)
	if !ok {
		return fmt.Errorf("cannot scan a %T value into an InputProperties value", src)
	}
	s = s[1 : len(s)-1] // trim leading and trailing "{" ... "}"
	parts := strings.Split(s, ",")
	*props = InputProperties(parts)
	return nil
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
func (this *Transformations) Set(connection int, transformations []*Transformation) error {

	if connection < 1 || connection > maxInt32 {
		return errors.New("invalid connection identifier")
	}

	// Validate the transformations.
	for _, t := range transformations {
		// TODO(Gianluca): validate the Python function here.
		if strings.TrimSpace(t.SourceCode) == "" {
			return errors.New("transformation function cannot be empty")
		}
		if len(t.In) == 0 {
			return errors.New("should have at least one input property")
		}
		for _, in := range t.In {
			if !types.IsValidPropertyName(in) {
				return errors.New("invalid property name")
			}
		}
		if !types.IsValidPropertyName(t.Out) {
			return errors.New("invalid property name")
		}
	}

	n := setConnectionTransformations{
		Connection:      connection,
		Transformations: transformations,
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
			return err
		}
		for _, t := range n.Transformations {
			var id int
			err := query.QueryRow(t.Connection, t.In, t.SourceCode, t.Out).Scan(&id)
			if err != nil {
				return err
			}
			t.ID = id
		}
		return tx.Notify(n)
	})
	if err != nil {
		if postgres.IsForeignKeyViolation(err) {
			switch postgres.ErrConstraintName(err) {
			case "transformations_connection_fkey":
				err = ConnectionNotFoundError{}
			}
		}
		return err
	}

	return nil
}

// List lists the transformations for the given connection.
//
// If there are no transformations associated to the given connection, this
// method returns nil.
func (this *Transformations) list(connection int) []*Transformation {
	this.state.Lock()
	ts := this.state.ofConnection[connection]
	this.state.Unlock()
	return ts
}

// List lists the transformations for the given connection.
// Returns a ConnectionNotFoundError error if the connection does not exist.
func (this *Transformations) List(connection int) ([]*Transformation, error) {
	if connection < 1 || connection > maxInt32 {
		return nil, errors.New("invalid connection identifier")
	}
	_, err := this.Workspace.Connections.Get(connection)
	if err != nil {
		return nil, err
	}
	ts := this.list(connection)
	if ts == nil {
		// No transformations associated to this connection.
		return []*Transformation{}, nil
	}
	return ts, nil
}
