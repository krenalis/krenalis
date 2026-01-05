// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package appwriter

import (
	"fmt"
	"iter"

	"github.com/meergo/meergo/connectors"
)

// iterator implements the connectors.Records interface to iterate over a
// sequence of records.
type iterator struct {
	writer    *Writer
	index     int // read index in writer.records, set by the writer.
	consumed  bool
	iterating bool
	postponed bool
	discarded bool
}

func newIterator(w *Writer) *iterator {
	it := iterator{writer: w}
	return &it
}

func (it *iterator) All() iter.Seq[connectors.Record] {
	if it.consumed {
		panic(it.writer.connector + " connector: Upsert method called Records.All after the records were consumed")
	}
	it.consumed = true
	return it.seq(opAll)
}

func (it *iterator) Discard(err error) {
	if !it.iterating {
		panic(it.writer.connector + " connector: Upsert method called Records.Discard outside an iteration")
	}
	if it.postponed {
		panic(it.writer.connector + " connector: Upsert method called Records.Discard on a postponed event")
	}
	if it.discarded {
		panic(it.writer.connector + " connector: Upsert method called Records.Discard on a discarded event")
	}
	if err == nil {
		panic(it.writer.connector + " connector: Upsert method called Records.Discard passing a nil error")
	}
	if trace {
		fmt.Printf("iterator.Postpone: iterator %p discardes a record\n", it)
	}
	it.discarded = true
	it.writer.discard(err)
}

func (it *iterator) First() connectors.Record {
	if it.consumed {
		panic(it.writer.connector + " connector: Upsert method called Records.First after the records were consumed")
	}
	it.consumed = true
	if trace {
		fmt.Printf("iterator.First: iterator %p reads only the first record\n", it)
	}
	record, ok := it.writer.read(opAll, true)
	it.writer.complete()
	if !ok {
		panic("core/connectors/appwriter: iterator has called Writer.read, but no records are available")
	}
	return record
}

func (it *iterator) Peek() (connectors.Record, bool) {
	if it.consumed && !it.iterating {
		panic(it.writer.connector + " connector: Upsert method called Records.Peek outside of an iteration")
	}
	if trace {
		fmt.Printf("iterator.Peek: iterator %p peek a record\n", it)
	}
	record, ok := it.writer.read(opAll, false)
	if !ok {
		return connectors.Record{}, false
	}
	return record, true
}

func (it *iterator) Postpone() {
	if !it.iterating {
		panic(it.writer.connector + " connector: Upsert method called Records.Postpone outside an iteration")
	}
	if it.discarded {
		panic(it.writer.connector + " connector: Upsert method called Records.Postpone on a discarded event")
	}
	if it.postponed {
		return
	}
	if trace {
		fmt.Printf("iterator.Postpone: iterator %p postpones a record\n", it)
	}
	it.postponed = true
	it.writer.postpone()
}

func (it *iterator) Same() iter.Seq[connectors.Record] {
	if it.consumed {
		panic(it.writer.connector + " connector: Upsert method called Records.Some after the records were consumed")
	}
	it.consumed = true
	op := opUpdate
	if record, _ := it.writer.read(opAll, false); record.ID == "" {
		op = opCreate
	}
	return it.seq(op)
}

// seq returns a sequence of records. If op is not opAll, it restricts the
// sequence to records of type creation (opCreate) or update (opUpdate).
func (it *iterator) seq(op op) iter.Seq[connectors.Record] {
	return func(yield func(record connectors.Record) bool) {
		if trace {
			fmt.Printf("iterator.seq: iterator %p starting to read %s records\n", it, op)
		}
		it.iterating = true
		for {
			it.postponed = false
			it.discarded = false
			record, ok := it.writer.read(op, true)
			if !ok {
				if trace {
					fmt.Printf("iterator.seq: iterator %p finished reading the records; no more are available\n", it)
				}
				break
			}
			if !yield(record) {
				if trace {
					fmt.Printf("iterator.seq: iterator %p broke out of the loop while reading records\n", it)
				}
				break
			}
		}
		it.iterating = false
		it.writer.complete()
	}
}
