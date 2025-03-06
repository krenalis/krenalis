//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package parquet

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/util"
	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"

	"github.com/fraugster/parquet-go/parquet"
)

func TestExportAndImportParquet(t *testing.T) {

	ctx := context.Background()

	// Instantiate the Parquet connector.
	config := meergo.FileConfig{}
	connector, err := New(&config)
	if err != nil {
		t.Fatal(err)
	}

	// Defines the content that will be exported to the Parquet file, as if they
	// were users reading from the data warehouse.
	exportedColumns := []types.Property{
		{Name: "subscribed", Type: types.Boolean(), ReadOptional: true},
		{Name: "rank_int8", Type: types.Int(8), ReadOptional: true},
		{Name: "rank_int16", Type: types.Int(16), ReadOptional: true},
		// This cannot be tested as Parquet does not support 24-bit integers,
		// which are therefore exported as 32-bit ints.
		// {Name: "rank_int24", Type: types.Int(24), ReadOptional: true},
		{Name: "rank_int32", Type: types.Int(32), ReadOptional: true},
		{Name: "rank_int64", Type: types.Int(64), ReadOptional: true},
		{Name: "first_name", Type: types.Text(), ReadOptional: true},
		{Name: "rank_uint8", Type: types.Uint(8), ReadOptional: true},
		{Name: "rank_uint16", Type: types.Uint(16), ReadOptional: true},
		// This cannot be tested as Parquet does not support 24-bit integers,
		// which are therefore exported as 32-bit ints.
		// {Name: "rank_uint24", Type: types.Uint(24), ReadOptional: true},
		{Name: "rank_uint32", Type: types.Uint(32), ReadOptional: true},
		{Name: "rank_uint64", Type: types.Uint(64), ReadOptional: true},
		{Name: "last_name", Type: types.Text(), ReadOptional: true},
		{Name: "score32", Type: types.Float(32), ReadOptional: true},
		{Name: "score64", Type: types.Float(64), ReadOptional: true},
		{Name: "decimal_1_0", Type: types.Decimal(1, 0), ReadOptional: true},
		{Name: "decimal_3_3", Type: types.Decimal(3, 3), ReadOptional: true},
		{Name: "decimal_10_3", Type: types.Decimal(10, 3), ReadOptional: true},
		{Name: "decimal_20_0", Type: types.Decimal(20, 0), ReadOptional: true},
		{Name: "decimal_20_20", Type: types.Decimal(20, 20), ReadOptional: true},
		{Name: "decimal_32_32", Type: types.Decimal(32, 32), ReadOptional: true},
		{Name: "decimal_37_37", Type: types.Decimal(37, 37), ReadOptional: true},
		{Name: "decimal_50_37", Type: types.Decimal(50, 37), ReadOptional: true},
		{Name: "decimal_76_37", Type: types.Decimal(76, 37), ReadOptional: true},
		{Name: "my_datetime", Type: types.DateTime(), ReadOptional: true},
		{Name: "my_date", Type: types.Date(), ReadOptional: true},
		{Name: "my_time", Type: types.Time(), ReadOptional: true},
		// This cannot be tested as Parquet does not support years.
		// {Name: "my_year", Type: types.Year(), ReadOptional: true},
		{Name: "my_uuid", Type: types.UUID(), ReadOptional: true},
		{Name: "my_json", Type: types.JSON(), ReadOptional: true},
	}
	// Values here must have the format documented in the Meergo doc about
	// export values and types (/developers/extend/connectors/data-values).
	exportedRecords := []map[string]any{
		{
			"subscribed": true,
			"first_name": "John",
			"last_name":  "Lemon",
		},
		{
			"first_name": "Ringo",
			"last_name":  "Planett",
		},
		{
			"first_name": "Ringo",
			"last_name":  "Planett",
			"score32":    float64(1234),
			"score64":    float64(5678),
		},
		{
			"rank_int8":   -80,
			"rank_int16":  -160,
			"rank_int32":  -320,
			"rank_int64":  -640,
			"rank_uint8":  uint(80),
			"rank_uint16": uint(160),
			"rank_uint32": uint(320),
			"rank_uint64": uint(640),
		},
		// Positive decimals.
		{
			"decimal_1_0":   decimal.MustParse("4"),
			"decimal_3_3":   decimal.MustParse("0.431"),
			"decimal_10_3":  decimal.MustParse("43298.432"),
			"decimal_20_0":  decimal.MustParse("12345678910111213141"),
			"decimal_20_20": decimal.MustParse("0.43274891578423975289"),
			"decimal_32_32": decimal.MustParse("0.43274891578423975289432789473289"),
			"decimal_37_37": decimal.MustParse("0.4327489157842397528943278947328943289"),
			"decimal_50_37": decimal.MustParse("1443328948239.4327489157842397528943278947328943289"),
			"decimal_76_37": decimal.MustParse("114328928398432438294823981443328948239.4327489157842397528943278947328943289"),
		},
		// Negative decimals.
		{
			"decimal_1_0":   decimal.MustParse("-4"),
			"decimal_3_3":   decimal.MustParse("-0.431"),
			"decimal_10_3":  decimal.MustParse("-43298.432"),
			"decimal_20_0":  decimal.MustParse("-12345678910111213141"),
			"decimal_20_20": decimal.MustParse("-0.43274891578423975289"),
			"decimal_32_32": decimal.MustParse("-0.43274891578423975289432789473289"),
			"decimal_37_37": decimal.MustParse("-0.4327489157842397528943278947328943289"),
			"decimal_50_37": decimal.MustParse("-1443328948239.4327489157842397528943278947328943289"),
			"decimal_76_37": decimal.MustParse("-114328928398432438294823981443328948239.4327489157842397528943278947328943289"),
		},
		{
			"decimal_1_0": decimal.MustParse("0"),
		},
		{
			"my_datetime": time.Date(2012, 12, 21, 15, 30, 2, 123456789, time.UTC),
			"my_date":     time.Date(2012, 12, 21, 0, 0, 0, 0, time.UTC),
			// Note that the nanoseconds part is truncated to zero in this test;
			// this is due to the issue: https://github.com/meergo/meergo/issues/1392.
			"my_time": time.Date(1970, 1, 1, 15, 30, 2, 123456000, time.UTC),
		},
		{
			"my_date": time.Date(1900, 12, 21, 0, 0, 0, 0, time.UTC), // before epoch.
		},
		{
			"my_uuid": "6cc9d700-57dc-48f7-81ab-2f3a13df8ea5",
		},
		{
			"my_json": json.Value(`{"a":10}`),
		},
	}

	// Create a temporary Parquet file to export to.
	//
	// It's useful that this is a physical file on disk (rather than a file in
	// memory) because this allows you to read the file with external tools to
	// debug it.
	parquetFile, err := os.CreateTemp("", "meergo-parquet-test-*.parquet")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := parquetFile.Close()
		if err != nil {
			t.Logf("cannot close temporary Parquet file: %s", err)
		}
	}()
	parquetFileName := parquetFile.Name()
	t.Logf("create temporary Parquet file with name: %s", parquetFileName)

	// Export the Parquet file.
	recordReader := &testRecordReader{
		t:       t,
		columns: exportedColumns,
		records: exportedRecords,
	}
	err = connector.Write(ctx, parquetFile, "", recordReader)
	if err != nil {
		t.Fatal(err)
	}
	err = parquetFile.Close()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("export completed (%d record(s) should have been written)", len(exportedRecords))

	// Check that all acks have been received.
	if recordReader.acksReceived != len(exportedRecords) {
		t.Fatalf("expected to receive %d ack(s), got %v", len(exportedRecords), recordReader.acksReceived)
	}
	t.Logf("correctly received %d ack(s)", recordReader.acksReceived)

	// Check the exported file with Pandas.
	pyExec, err := lookupPythonExecPath()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(pyExec, "test_parquet_file.py", parquetFileName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		t.Fatalf("validation of Parquet file using Pandas failed: %s. The test's output may contain some additional information", err)
	}
	t.Logf("the exported Parquet file has been validated correctly by Pandas")

	// Import the Parquet file.
	recordWriter := &testRecordWriter{
		t:           t,
		readRecords: []map[string]any{},
	}
	parquetFile, err = os.Open(parquetFileName)
	if err != nil {
		t.Fatal(err)
	}
	err = connector.Read(ctx, parquetFile, "", recordWriter)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("import from Parquet file completed")

	// Check if the read columns match with the exported columns.
	if len(exportedColumns) != len(recordWriter.readColumns) {
		t.Fatalf("%d column(s) expected, but %d have been read",
			len(exportedColumns), len(recordWriter.readColumns))
	}
	t.Logf("%d column(s) read from the Parquet file", len(recordWriter.readColumns))
	fail := false
	for i := range exportedColumns {
		expected := exportedColumns[i]
		got := recordWriter.readColumns[i]
		if expected.Name != got.Name {
			t.Logf("column [%d]: expected name %q, got %q", i, expected.Name, got.Name)
			fail = true
			continue
		}
		if !types.Equal(expected.Type, got.Type) {
			t.Logf("column %q: expected type %v, got %v", expected.Name, expected.Type, got.Type)
			fail = true
			continue
		}
		t.Logf("name and type of column %q match with the exported one", expected.Name)
	}
	if fail {
		t.Fatal("read columns do not match with exported columns")
	}

	// Check if the read records match with the exported columns.
	if len(exportedRecords) != len(recordWriter.readRecords) {
		t.Fatalf("%d record(s) expected, but only %d have been read",
			len(exportedRecords), len(recordWriter.readRecords))
	}
	t.Logf("%d record(s) read from the Parquet file", len(recordWriter.readRecords))
	for i := range exportedRecords {
		expectedRecord := exportedRecords[i]
		gotRecords := recordWriter.readRecords[i]
		if len(expectedRecord) != len(gotRecords) {
			t.Fatalf("expected %v properties, got %v", len(expectedRecord), len(gotRecords))
		}
		for _, c := range exportedColumns {
			expectedProperty := expectedRecord[c.Name]
			gotProperty := gotRecords[c.Name]
			var equalProperty bool
			if c.Type.Kind() == types.DecimalKind {
				expectedDec, ok1 := expectedProperty.(decimal.Decimal)
				gotDec, ok2 := gotProperty.(decimal.Decimal)
				if ok1 && ok2 {
					equalProperty = expectedDec.Equal(gotDec)
				} else {
					equalProperty = reflect.DeepEqual(expectedProperty, gotProperty)
				}
			} else {
				equalProperty = reflect.DeepEqual(expectedProperty, gotProperty)
			}
			if !equalProperty {
				t.Fatalf("%q: expected property value %q, got %q", c.Name, expectedProperty, gotProperty)
			}
		}
		t.Logf("imported record [%d] matches with exported record", i)
	}
	t.Logf("record value(s) match with expected values")

}

// TestExport tests all those cases that cannot be tested in
// TestExportAndImportParquet, perhaps because they involve values that cannot
// be read back from Parquet without losing information.
func TestExport(t *testing.T) {

	ctx := context.Background()

	// Instantiate the Parquet connector.
	config := meergo.FileConfig{}
	connector, err := New(&config)
	if err != nil {
		t.Fatal(err)
	}

	exportedColumns := []types.Property{
		{Name: "rank_int24", Type: types.Int(24), ReadOptional: true},
		{Name: "rank_uint24", Type: types.Uint(24), ReadOptional: true},
		{Name: "p_year", Type: types.Year(), ReadOptional: true},
		{Name: "p_inet", Type: types.Inet(), ReadOptional: true},
		{Name: "address", Type: types.Object([]types.Property{
			{Name: "street", Type: types.Text(), ReadOptional: true},
			{Name: "zip_code", Type: types.Int(32), ReadOptional: true},
		})},
	}
	exportedRecords := []map[string]any{
		{
			"rank_int24":  int(1234),
			"rank_uint24": uint(1234),
			"p_year":      2001,
			"p_inet":      "192.128.0.1",
			"address": map[string]any{
				"street":   "123 Strett",
				"zip_code": 12345,
			},
		},
	}

	var parquetFile bytes.Buffer
	recordReader := &testRecordReader{
		t:       t,
		columns: exportedColumns,
		records: exportedRecords,
	}
	err = connector.Write(ctx, &parquetFile, "", recordReader)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("ok, Write returned without any error")

}

// Test RecordReader (used when exporting).

var _ meergo.RecordReader = &testRecordReader{}

type testRecordReader struct {
	t            *testing.T
	columns      []types.Property
	records      []map[string]any
	index        int
	acksReceived int
}

func (records *testRecordReader) Ack(id string, err error) {
	if err != nil {
		records.t.Fatalf("called ack function with an error: %v", err)
	}
	records.acksReceived++
}

func (records *testRecordReader) Columns() []types.Property {
	return records.columns
}

func (records *testRecordReader) Record(ctx context.Context) (ackID string, record map[string]any, err error) {
	if records.index == len(records.records) {
		return "", nil, io.EOF
	}
	ackID = strconv.Itoa(records.index)
	record = records.records[records.index]
	records.index++
	return ackID, record, nil
}

// Test RecordWriter (used when importing).

var _ meergo.RecordWriter = &testRecordWriter{}

type testRecordWriter struct {
	t           *testing.T
	readColumns []types.Property
	readRecords []map[string]any
}

func (writer *testRecordWriter) Columns(columns []types.Property) error {
	if writer.readColumns != nil {
		writer.t.Fatal("Columns method already called")
	}
	writer.readColumns = columns
	return nil
}

func (writer *testRecordWriter) columnByName(name string) types.Property {
	for _, p := range writer.readColumns {
		if p.Name == name {
			return p
		}
	}
	writer.t.Fatalf("column %q not read from the Parquet file", name)
	return types.Property{}
}

func (writer *testRecordWriter) Record(record map[string]any) error {
	// Normalize and do some checks on record values.
	toDelete := []string{}
	for name, value := range record {
		if value == nil {
			toDelete = append(toDelete, name)
			continue
		}
		column := writer.columnByName(name)
		switch column.Type.Kind() {
		case types.IntKind:
			if column.Type.BitSize() <= 32 {
				record[name] = int(value.(int32))
			} else {
				record[name] = int(value.(int64))
			}
		case types.UintKind:
			if column.Type.BitSize() <= 32 {
				record[name] = uint(value.(int32))
			} else {
				record[name] = uint(value.(int64))
			}
		case types.FloatKind:
			if column.Type.BitSize() == 32 {
				record[name] = float64(value.(float32))
			}
		case types.DecimalKind:
			_, ok := record[name].(decimal.Decimal)
			if !ok {
				writer.t.Fatalf("decimal values should have type decimal.Decimal, got %v (type %T)", record[name], record[name])
			}
		case types.UUIDKind:
			record[name], _ = util.UUIDFromBytes(value.([]byte))
		case types.JSONKind:
			record[name] = json.Value(value.([]byte))
		case types.TextKind:
			record[name] = string(value.([]byte))
		}
	}
	for _, name := range toDelete {
		delete(record, name)
	}
	writer.readRecords = append(writer.readRecords, record)
	return nil
}

func (writer *testRecordWriter) RecordSlice(record []any) error {
	panic("method not implemented")
}

func (writer *testRecordWriter) RecordStrings(record []string) error {
	panic("method not implemented")
}

func TestTimestampsBackAndForth(t *testing.T) {

	tests := []time.Time{
		time.Date(2200, 12, 21, 15, 34, 23, 12345, time.UTC),
		time.Date(2200, 12, 21, 15, 34, 23, 12345, time.Local),
		time.Date(2012, 12, 21, 15, 34, 23, 12345, time.UTC),
		time.Date(2012, 12, 21, 15, 34, 23, 12345, time.Local),
		time.Date(1970, 1, 1, 0, 0, 0, 0, time.Local),
		time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(1900, 1, 1, 0, 0, 0, 0, time.Local),
		time.Date(1678, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	unit := parquet.NewTimeUnit()
	unit.NANOS = parquet.NewNanoSeconds()

	for _, test := range tests {

		// First: convert the time.Time to int64.
		i64, err := timeTimeToInt64(test)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Second: convert back the int64 to time.Time.
		back := int64ToTimeTime(i64, unit)

		// Check if they are equal.
		if test.Equal(back) {
			t.Logf("ok: %v", test)
		} else {
			t.Errorf("expected %v, got %v", test, back)
		}
	}

}

func Test_int64ToTimeTime(t *testing.T) {

	nanoUnit := parquet.NewTimeUnit()
	nanoUnit.NANOS = parquet.NewNanoSeconds()

	microUnit := parquet.NewTimeUnit()
	microUnit.MICROS = parquet.NewMicroSeconds()

	milliUnit := parquet.NewTimeUnit()
	milliUnit.MILLIS = parquet.NewMilliSeconds()

	tests := []struct {
		v        int64
		unit     *parquet.TimeUnit
		expected time.Time
	}{
		{
			v:        0,
			unit:     nil,
			expected: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			v:        0,
			unit:     nanoUnit,
			expected: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			v:        0,
			unit:     microUnit,
			expected: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			v:        0,
			unit:     milliUnit,
			expected: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			v:        int64(946684800 * 1_000_000_000),
			unit:     nanoUnit,
			expected: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			v:        int64(946684800 * 1_000_000),
			unit:     microUnit,
			expected: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			v:        int64(946684800 * 1_000),
			unit:     milliUnit,
			expected: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			v:        int64(-2208988800 * 1_000_000_000),
			unit:     nanoUnit,
			expected: time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			v:        int64(-2208988800 * 1_000_000),
			unit:     microUnit,
			expected: time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			v:        int64(-2208988800 * 1_000),
			unit:     milliUnit,
			expected: time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			v:        int64(math.MaxInt64),
			unit:     nanoUnit,
			expected: time.Date(2262, 4, 11, 23, 47, 16, 854775807, time.UTC),
		},
		{
			v:        int64(math.MinInt64),
			unit:     nanoUnit,
			expected: time.Date(1677, 9, 21, 0, 12, 43, 145224192, time.UTC),
		},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got := int64ToTimeTime(test.v, test.unit)
			if !got.Equal(test.expected) {
				t.Fatalf("expected %v, got %v", test.expected, got)
			}
		})
	}
}

func Test_timeTimeToInt64(t *testing.T) {

	tests := []struct {
		ts        time.Time
		expected  int64
		expectErr bool
	}{
		{
			ts:        time.Date(1677, 9, 21, 0, 12, 43, 145224192, time.UTC),
			expectErr: true,
		},
		{
			ts:       time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: int64(-2208988800 * 1_000_000_000),
		},
		{
			ts:       time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: int64(946684800 * 1_000_000_000),
		},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got, gotErr := timeTimeToInt64(test.ts)
			if gotErr != nil && !test.expectErr {
				t.Fatalf("not expected error: %v", gotErr)
			}
			if gotErr == nil && test.expectErr {
				t.Fatal("expected error")
			}
			if gotErr != nil {
				return
			}
			if got != test.expected {
				t.Fatalf("expected %v, got %v", test.expected, got)
			}
		})
	}

}

// lookupPythonExecPath returns the path of the Python executable available on
// this system, or an error if it cannot be found.
func lookupPythonExecPath() (string, error) {
	// TODO: Keep in sync with other copies of this function, scattered
	// throughout the code, that have the same name.
	pythonNames := []string{"python", "python3", "python3.13"}
	for _, name := range pythonNames {
		path, err := exec.LookPath(name)
		if err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("the Python executable cannot be found "+
		"with any of these names: %s", strings.Join(pythonNames, ", "))
}
