// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package snowflaketester

import (
	"context"

	"github.com/snowflakedb/gosnowflake/v2"
)

// ServiceContext tags a public Profiles service read for the campaign observer.
func ServiceContext(ctx context.Context, tag string) context.Context {
	if ctx == nil || tag != ServiceQueryTag && tag != ServiceExplanationTag && tag != ServiceRecordTag {
		panic("test/snowflaketester: invalid service application tag")
	}
	return gosnowflake.WithQueryTag(ctx, tag)
}
