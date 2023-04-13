//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package transformations_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"chichi/apis/transformations"
)

func BenchmarkRun(b *testing.B) {
	for i := 0; i < b.N; i++ {
		pool := transformations.NewPool()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		src := `
def transform(user):
	return user`
		got, err := pool.Run(ctx, src, map[string]any{"Name": "John", "LastName": "Lennon"})
		if err != nil {
			b.Fatal(err)
		}
		if !reflect.DeepEqual(got, map[string]any{"Name": "John", "LastName": "Lennon"}) {
			b.Fatalf("bad")
		}
	}
}
