//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/meergo/meergo/core/types"
	"github.com/meergo/meergo/test/meergotester"
)

func TestParquetTestApacheFiles(t *testing.T) {

	// TODO: test all files within 'test/testdata/apache/parquet-testing'.
	// TODO: test column values in addition to column names and types.
	// See the issue https://github.com/meergo/meergo/issues/1418.

	// Retrieve the storage directory that contains the Parquet file to import.
	storageDir, err := filepath.Abs("testdata/apache/parquet-testing/data")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(storageDir); err != nil {
		t.Fatal(err)
	}

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.NewMeergoInstance(t)
	c.PopulateUserSchema(false)
	c.SetFilesystemRoot(storageDir)
	c.Start()
	defer c.Stop()

	tests := []struct {
		path               string
		expectedProperties []types.Property
	}{
		{
			path: "alltypes_plain.parquet",
			expectedProperties: []types.Property{
				{Name: "id", Type: types.Int(32), Nullable: true},
				{Name: "bool_col", Type: types.Boolean(), Nullable: true},
				{Name: "tinyint_col", Type: types.Int(32), Nullable: true},
				{Name: "smallint_col", Type: types.Int(32), Nullable: true},
				{Name: "int_col", Type: types.Int(32), Nullable: true},
				{Name: "bigint_col", Type: types.Int(64), Nullable: true},
				{Name: "float_col", Type: types.Float(32), Nullable: true},
				{Name: "double_col", Type: types.Float(64), Nullable: true},
				{Name: "date_string_col", Type: types.Text(), Nullable: true},
				{Name: "string_col", Type: types.Text(), Nullable: true},
				{Name: "timestamp_col", Type: types.DateTime(), Nullable: true},
			},
		},
		{
			path:               "byte_array_decimal.parquet",
			expectedProperties: []types.Property{{Name: "value", Type: types.Decimal(4, 2), Nullable: true}},
		},
		{
			path:               "binary.parquet",
			expectedProperties: []types.Property{{Name: "foo", Type: types.Text(), Nullable: true}},
		},
		{
			path:               "concatenated_gzip_members.parquet",
			expectedProperties: []types.Property{{Name: "long_col", Type: types.Uint(64), Nullable: true}},
		},
		{
			path:               "delta_length_byte_array.parquet",
			expectedProperties: []types.Property{{Name: "FRUIT", Type: types.Text(), Nullable: true}},
		},
		{
			path:               "int32_decimal.parquet",
			expectedProperties: []types.Property{{Name: "value", Type: types.Decimal(4, 2), Nullable: true}},
		},
		{
			path:               "fixed_length_byte_array.parquet",
			expectedProperties: []types.Property{{Name: "flba_field", Type: types.Text(), Nullable: false}},
		},
		{
			path: "sort_columns.parquet",
			expectedProperties: []types.Property{
				{Name: "a", Type: types.Int(64), Nullable: true},
				{Name: "b", Type: types.Text(), Nullable: true},
			},
		},
		{
			path: "byte_array_decimal.parquet",
			expectedProperties: []types.Property{
				{Name: "value", Type: types.Decimal(4, 2), Nullable: true},
			},
		},
		{
			path: "data_index_bloom_encoding_stats.parquet",
			expectedProperties: []types.Property{
				{Name: "String", Type: types.Text(), Nullable: true},
			},
		},
		{
			path: "lz4_raw_compressed.parquet",
			expectedProperties: []types.Property{
				{Name: "c0", Type: types.Int(64), Nullable: false},
				{Name: "c1", Type: types.Text(), Nullable: false},
				{Name: "v11", Type: types.Float(64), Nullable: true},
			},
		},
		{
			path: "nan_in_stats.parquet",
			expectedProperties: []types.Property{
				{Name: "x", Type: types.Float(64), Nullable: true},
			},
		},
		{
			path: "single_nan.parquet",
			expectedProperties: []types.Property{
				{Name: "mycol", Type: types.Float(64), Nullable: true},
			},
		},
		{
			path: "byte_stream_split.zstd.parquet",
			expectedProperties: []types.Property{
				{Name: "f32", Type: types.Float(32), Nullable: true},
				{Name: "f64", Type: types.Float(64), Nullable: true},
			},
		},
	}

	fs := c.CreateSourceFilesystem()

	for _, test := range tests {

		t.Run(test.path, func(t *testing.T) {

			// Read the file.
			_, gotSchema := c.File(fs, test.path, "Parquet", "", meergotester.NoCompression, nil, 0)
			gotProperties := types.Properties(gotSchema)

			// Validate the properties.
			if len(gotProperties) != len(test.expectedProperties) {
				t.Errorf("expected properties: %#v", test.expectedProperties)
				t.Errorf("got properties:      %#v", gotProperties)
				t.Fatalf("expected %d properties, got %d", len(test.expectedProperties), len(gotProperties))
			}
			for i := range gotProperties {
				gotProperty := gotProperties[i]
				expectedProperty := test.expectedProperties[i]
				equal := gotProperty.Name == expectedProperty.Name &&
					types.Equal(gotProperty.Type, expectedProperty.Type) &&
					gotProperty.Nullable == expectedProperty.Nullable
				if !equal {
					t.Errorf("expected property name:     %q", expectedProperty.Name)
					t.Errorf("got property name:          %q", gotProperty.Name)
					t.Errorf("expected property type:     %s", expectedProperty.Type)
					t.Errorf("got property type:          %s", gotProperty.Type)
					t.Errorf("expected property nullable: %t", expectedProperty.Nullable)
					t.Errorf("got property nullable:      %t", gotProperty.Nullable)
					t.Fatalf("expected property[%d] do not match with read property [%d]", i, i)
				}
			}

		})

	}

}
