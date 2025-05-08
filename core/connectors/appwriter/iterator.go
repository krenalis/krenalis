//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package appwriter

import (
	"fmt"
	"iter"

	"github.com/meergo/meergo"
)

// iterator implements the meergo.Records interface to iterate over a sequence
// of records.
type iterator struct {
	writer    *Writer
	index     int // read index in writer.records, set by the writer.
	consumed  bool
	iterating bool
	skipped   bool
}

func newIterator(w *Writer) *iterator {
	it := iterator{writer: w}
	return &it
}

func (it *iterator) All() iter.Seq2[int, meergo.Record] {
	if it.consumed {
		panic(it.writer.name + " connector: Upsert method called Records.All after the records were consumed")
	}
	it.consumed = true
	return it.seq(opAll)
}

func (it *iterator) First() meergo.Record {
	if it.consumed {
		panic(it.writer.name + " connector: Upsert method called Records.First after the records were consumed")
	}
	it.consumed = true
	if trace {
		fmt.Printf("iterator.First: iterator %p reads only the first record\n", it)
	}
	record, ok := it.writer.read(opAll, 0)
	it.writer.complete()
	if !ok {
		panic("core/connectors/appwriter: iterator has called Writer.read, but no records are available")
	}
	return record
}

func (it *iterator) Peek() (meergo.Record, bool) {
	if it.consumed && !it.iterating {
		panic(it.writer.name + " connector: Upsert method called Records.Peek outside of an iteration")
	}
	if trace {
		fmt.Printf("iterator.Peek: iterator %p peek a record\n", it)
	}
	record, ok := it.writer.read(opAll, dontConsume)
	if !ok {
		return meergo.Record{}, false
	}
	return record, true
}

func (it *iterator) Same() iter.Seq2[int, meergo.Record] {
	if it.consumed {
		panic(it.writer.name + " connector: Upsert method called Records.Some after the records were consumed")
	}
	it.consumed = true
	op := opUpdate
	if record, _ := it.writer.read(opAll, dontConsume); record.ID == "" {
		op = opCreate
	}
	return it.seq(op)
}

func (it *iterator) Skip() {
	if !it.iterating {
		panic(it.writer.name + " connector: Upsert method called Records.Skip outside an iteration")
	}
	if it.skipped {
		return
	}
	if trace {
		fmt.Printf("iterator.Skip: iterator %p skips a record\n", it)
	}
	it.skipped = true
	it.writer.skip()
}

// seq returns a sequence of records. If op is not opAll, it restricts the
// sequence to records of type creation (opCreate) or update (opUpdate).
func (it *iterator) seq(op op) iter.Seq2[int, meergo.Record] {
	return func(yield func(i int, record meergo.Record) bool) {
		if trace {
			fmt.Printf("iterator.seq: iterator %p starting to read %s records\n", it, op)
		}
		it.iterating = true
		i := 0
		for {
			it.skipped = false
			record, ok := it.writer.read(op, i)
			if !ok {
				if trace {
					fmt.Printf("iterator.seq: iterator %p finished reading the records; no more are available\n", it)
				}
				break
			}
			if !yield(i, record) {
				if trace {
					fmt.Printf("iterator.seq: iterator %p broke out of the loop while reading records\n", it)
				}
				break
			}
			if !it.skipped {
				i++
			}
		}
		it.iterating = false
		it.writer.complete()
	}
}
