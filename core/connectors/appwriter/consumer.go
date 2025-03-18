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

// consumer implements the meergo.Records interface.
type consumer struct {
	writer    *Writer
	consumed  bool
	iterating bool
	skipped   bool
}

func newConsumer(w *Writer) *consumer {
	it := consumer{writer: w}
	return &it
}

func (c *consumer) All() iter.Seq2[int, meergo.Record] {
	if c.consumed {
		panic(c.writer.name + " connector: Upsert method called Records.All after the records were consumed")
	}
	c.consumed = true
	return c.seq(opAll)
}

func (c *consumer) First() meergo.Record {
	if c.consumed {
		panic(c.writer.name + " connector: Upsert method called Records.First after the records were consumed")
	}
	c.consumed = true
	if trace {
		fmt.Printf("consumer.First: consumer %p reads only the first record\n", c)
	}
	record, ok := c.writer.read(opAll, 0)
	c.writer.complete()
	if !ok {
		panic("core/connectors/appwriter: consumer has called Writer.read, but no records are available")
	}
	return record
}

func (c *consumer) Peek() (meergo.Record, bool) {
	if c.consumed && !c.iterating {
		panic(c.writer.name + " connector: Upsert method called Records.Peek outside of an iteration")
	}
	if trace {
		fmt.Printf("consumer.Peek: consumer %p peek a record\n", c)
	}
	record, ok := c.writer.read(opAll, dontConsume)
	if !ok {
		return meergo.Record{}, false
	}
	return record, true
}

func (c *consumer) Same() iter.Seq2[int, meergo.Record] {
	if c.consumed {
		panic(c.writer.name + " connector: Upsert method called Records.Some after the records were consumed")
	}
	c.consumed = true
	op := opUpdate
	if record, _ := c.writer.read(opAll, dontConsume); record.ID == "" {
		op = opCreate
	}
	return c.seq(op)
}

func (c *consumer) Skip() {
	if !c.iterating {
		panic(c.writer.name + " connector: Upsert method called Records.Skip outside an iteration")
	}
	if c.skipped {
		return
	}
	if trace {
		fmt.Printf("consumer.Skip: consumer %p skips a record\n", c)
	}
	c.skipped = true
	c.writer.skip()
}

// seq returns a sequence of records. If op is not opAll, it restricts the
// sequence to records of type creation (opCreate) or update (opUpdate).
func (c *consumer) seq(op op) iter.Seq2[int, meergo.Record] {
	return func(yield func(i int, record meergo.Record) bool) {
		if trace {
			fmt.Printf("consumer.seq: consumer %p starting to read %s records\n", c, op)
		}
		c.iterating = true
		i := 0
		for {
			c.skipped = false
			record, ok := c.writer.read(op, i)
			if !ok {
				if trace {
					fmt.Printf("consumer.seq: consumer %p finished reading the records; no more are available\n", c)
				}
				break
			}
			if !yield(i, record) {
				if trace {
					fmt.Printf("consumer.seq: consumer %p broke out of the loop while reading records\n", c)
				}
				break
			}
			if !c.skipped {
				i++
			}
		}
		c.iterating = false
		c.writer.complete()
	}
}
