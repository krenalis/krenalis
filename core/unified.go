// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"time"

	"github.com/krenalis/krenalis/core/internal/schemas"
	"github.com/krenalis/krenalis/core/internal/unified"
)

// unifiedRecordQuery is a programmatic service request, not a wire format.
type unifiedRecordQuery struct {
	Scope unified.Scope
	Where *Filter
	Test  unified.CountTest
	At    time.Time
	Page  unified.Page
}

func (this *Workspace) unifiedRecordProfiles(ctx context.Context, query unifiedRecordQuery) (*unified.Result, error) {
	this.core.mustBeOpen()
	rule, err := prepareUnifiedRecordCount(query.Scope, query.Where, query.Test)
	if err != nil {
		return nil, err
	}
	return this.store.UnifiedProfiles(ctx, &rule, unified.Bindings{}, query.At, query.Page)
}

// prepareUnifiedRecordCount admits a Count with the existing public Record
// filter on its occurrences. It is an internal entry, not a new FilterRule wire
// tag. The store validates the remaining canonical scope and terminal.
func prepareUnifiedRecordCount(scope unified.Scope, filter *Filter, test unified.CountTest) (unified.Rule, error) {
	where, _, err := prepareFilter(filter, schemas.Event)
	if err != nil {
		return unified.Rule{}, err
	}
	return unified.Rule{Count: &unified.Count{Scope: scope, Where: where, Test: test}}, nil
}
