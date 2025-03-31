//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package meergo

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"time"

	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"
)

// validateAppConnector validates the passed app connector, performing checks to
// detect errors that could cause panic or errors in the Meergo code that uses
// the connectors.
//
// In case of a validation error, this function panics.
func validateAppConnector(app AppInfo) {

	// TODO(Gianluca): Groups are currently not supported, see
	// https://github.com/meergo/meergo/issues/895.
	if (app.AsSource != nil && app.AsSource.Targets&Groups != 0) ||
		(app.AsDestination != nil && app.AsDestination.Targets&Groups != 0) {
		panic("target Groups is not supported by this installation of Meergo (see https://github.com/meergo/meergo/issues/895)")
	}

	if app.AsSource != nil {
		if app.AsSource.Targets&Users != 0 {
			iface := reflect.TypeFor[interface {
				Schema(ctx context.Context, target Targets, role Role, eventType string) (types.Type, error)
				Records(ctx context.Context, target Targets, lastChangeTime time.Time, ids, properties []string, cursor string, schema types.Type) ([]Record, string, error)
			}]()
			if !app.ct.Implements(iface) {
				panic(fmt.Sprintf("inconsistency between the declared functionalities for the %s connector and the methods it actually implements", app.Name))
			}
		}
	}

	if app.AsDestination != nil {
		if app.AsDestination.Targets&Users != 0 {
			iface := reflect.TypeFor[interface {
				Schema(ctx context.Context, target Targets, role Role, eventType string) (types.Type, error)
				Upsert(ctx context.Context, target Targets, records Records) error
				Records(ctx context.Context, target Targets, lastChangeTime time.Time, ids, properties []string, cursor string, schema types.Type) ([]Record, string, error)
			}]()
			if !app.ct.Implements(iface) {
				panic(fmt.Sprintf("inconsistency between the declared functionalities for the %s connector and the methods it actually implements", app.Name))
			}
		}
		if app.AsDestination.Targets&Events != 0 {
			iface := reflect.TypeFor[interface {
				EventRequest(ctx context.Context, event Event, eventType string, schema types.Type, properties map[string]any, redacted bool) (*EventRequest, error)
				EventTypes(ctx context.Context) ([]*EventType, error)
				Schema(ctx context.Context, target Targets, role Role, eventType string) (types.Type, error)
			}]()
			if !app.ct.Implements(iface) {
				panic(fmt.Sprintf("inconsistency between the declared functionalities for the %s connector and the methods it actually implements", app.Name))
			}
			if app.AsDestination.SendingMode == None {
				panic(fmt.Sprintf("connector %s is declared to support Events as destination, but it does not specify a sending mode", app.Name))
			}
		}
	}

	if app.Terms.User != "" || app.Terms.Users != "" {
		if (app.AsSource == nil || app.AsSource.Targets&Users == 0) &&
			(app.AsDestination == nil || app.AsDestination.Targets&Users == 0) {
			panic(fmt.Sprintf("connector %s cannot specify a term for user and/or users"+
				" if it does not support the Users target neither as source nor as destination", app.Name))
		}
	}

	if app.Terms.Group != "" || app.Terms.Groups != "" {
		if (app.AsSource == nil || app.AsSource.Targets&Groups == 0) &&
			(app.AsDestination == nil || app.AsDestination.Targets&Groups == 0) {
			panic(fmt.Sprintf("connector %s cannot specify a term for group and/or groups"+
				" if it does not support the Groups target neither as source nor as destination", app.Name))
		}
	}

	if (app.AsSource != nil && app.AsSource.HasSettings) ||
		(app.AsDestination != nil && app.AsDestination.HasSettings) {
		iface := reflect.TypeFor[interface {
			ServeUI(ctx context.Context, event string, settings json.Value, role Role) (*UI, error)
		}]()
		if !app.ct.Implements(iface) {
			panic(fmt.Sprintf("inconsistency between the declared functionalities for the %s connector and the methods it actually implements", app.Name))
		}
	}

	if app.OAuth.AuthURL != "" {
		iface := reflect.TypeFor[interface {
			OAuthAccount(ctx context.Context) (string, error)
		}]()
		if !app.ct.Implements(iface) {
			panic(fmt.Sprintf("inconsistency between the declared functionalities for the %s connector and the methods it actually implements", app.Name))
		}
	}

	if app.WebhooksPer != WebhooksPerNone {
		iface := reflect.TypeFor[interface {
			ReceiveWebhook(r *http.Request, role Role) ([]WebhookPayload, error)
		}]()
		if !app.ct.Implements(iface) {
			panic(fmt.Sprintf("inconsistency between the declared functionalities for the %s connector and the methods it actually implements", app.Name))
		}
	}

}

// validateDatabaseConnector validates the passed database connector, performing
// checks to detect errors that could cause panic or errors in the Meergo code
// that uses the connectors.
//
// In case of a validation error, this function panics.
func validateDatabaseConnector(database DatabaseInfo) {
	iface := reflect.TypeFor[interface {
		Close() error
		Columns(ctx context.Context, table string) ([]Column, error)
		Merge(ctx context.Context, table Table, rows [][]any) error
		Query(ctx context.Context, query string) (Rows, []Column, error)
		QuoteTime(value any, typ types.Type) string
		ServeUI(ctx context.Context, event string, settings json.Value, role Role) (*UI, error)
	}]()
	if !database.ct.Implements(iface) {
		panic(fmt.Sprintf("the Database connector %s does not implement the required methods", database.Name))
	}
}

// validateFileConnector validates the passed file connector, performing checks
// to detect errors that could cause panic or errors in the Meergo code that
// uses the connectors.
//
// In case of a validation error, this function panics.
func validateFileConnector(file FileInfo) {

	if file.AsSource != nil {
		iface := reflect.TypeFor[interface {
			Read(ctx context.Context, r io.Reader, sheet string, records RecordWriter) error
		}]()
		if !file.ct.Implements(iface) {
			panic(fmt.Sprintf("inconsistency between the declared functionalities for the %s connector and the methods it actually implements", file.Name))
		}
	}

	if file.AsDestination != nil {
		iface := reflect.TypeFor[interface {
			Write(ctx context.Context, w io.Writer, sheet string, records RecordReader) error
			ContentType(ctx context.Context) string
		}]()
		if !file.ct.Implements(iface) {
			panic(fmt.Sprintf("inconsistency between the declared functionalities for the %s connector and the methods it actually implements", file.Name))
		}
	}

	if file.HasSheets {
		iface := reflect.TypeFor[interface {
			Sheets(ctx context.Context, r io.Reader) ([]string, error)
		}]()
		if !file.ct.Implements(iface) {
			panic(fmt.Sprintf("inconsistency between the declared functionalities for the %s connector and the methods it actually implements", file.Name))
		}
	}

	if (file.AsSource != nil && file.AsSource.HasSettings) ||
		(file.AsDestination != nil && file.AsDestination.HasSettings) {
		iface := reflect.TypeFor[interface {
			ServeUI(ctx context.Context, event string, settings json.Value, role Role) (*UI, error)
		}]()
		if !file.ct.Implements(iface) {
			panic(fmt.Sprintf("inconsistency between the declared functionalities for the %s connector and the methods it actually implements", file.Name))
		}
	}

}

// validateFileStorageConnector validates the passed file storage connector,
// performing checks to detect errors that could cause panic or errors in the
// Meergo code that uses the connectors.
//
// In case of a validation error, this function panics.
func validateFileStorageConnector(fileStorage FileStorageInfo) {

	iface := reflect.TypeFor[interface {
		AbsolutePath(ctx context.Context, name string) (string, error)
		ServeUI(ctx context.Context, event string, settings json.Value, role Role) (*UI, error)
	}]()
	if !fileStorage.ct.Implements(iface) {
		panic(fmt.Sprintf("the FileStorage connector %s does not implement the minimum required methods", fileStorage.Name))
	}

	if fileStorage.AsSource {
		iface := reflect.TypeFor[interface {
			Reader(ctx context.Context, name string) (io.ReadCloser, time.Time, error)
		}]()
		if !fileStorage.ct.Implements(iface) {
			panic(fmt.Sprintf("inconsistency between the declared functionalities for the %s connector and the methods it actually implements", fileStorage.Name))
		}
	}

	if fileStorage.AsDestination {
		iface := reflect.TypeFor[interface {
			Write(ctx context.Context, r io.Reader, name, contentType string) error
		}]()
		if !fileStorage.ct.Implements(iface) {
			panic(fmt.Sprintf("inconsistency between the declared functionalities for the %s connector and the methods it actually implements", fileStorage.Name))
		}
	}

}

// validateStreamConnector validates the passed stream connector, performing
// checks to detect errors that could cause panic or errors in the Meergo code
// that uses the connectors.
//
// In case of a validation error, this function panics.
func validateStreamConnector(stream StreamInfo) {
	iface := reflect.TypeFor[interface {
		Close() error
		Receive(ctx context.Context) (event []byte, ack func(), err error)
		Send(ctx context.Context, event []byte, options SendOptions, ack func(err error)) error
	}]()
	if !stream.ct.Implements(iface) {
		panic(fmt.Sprintf("the Stream connector %s does not implement the required methods", stream.Name))
	}
}
