// Copyright (c) 2016 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package zap

import (
	"io"
	"time"
)

// An Entry represents a log mesasge being logged. It is created to capture
// state beneath a Logger method, like Info, and passed around to all Facility
// values attached to the logger.
type Entry struct {
	Level   Level
	Time    time.Time
	Message string

	fieldSets []Field
}

// AddFields adds a new set of fields to the log entry.
func (e Entry) AddFields(fields ...Field) Entry {
	// TODO: pool []Field?
	e.fieldSets = append(e.fieldSets, fields)
	return e
}

// EachField calls the given function for each Field added by AddFields.
func (e Entry) EachField(f func(Field) bool) {
	for i := range e.fieldSets {
		for j := range e.fieldSets[i] {
			if f(e.fieldSets[i][j]) {
				return
			}
		}
	}
}

// EncodeTo encodes the entry and fields to an io.Writer using an Encoder,
// returning any error.
func (e Entry) EncodeTo(w io.Writer, enc Encoder, fields []Field) error {
	enc = enc.Clone()
	addFields(enc, fields)
	err := enc.WriteEntry(w, msg, lvl, t)
	enc.Free()
	return err
}

// Fields returns a mutable reference to the entry's accumulated context.
func (e Entry) Fields() KeyValue {
	return e.enc
}
