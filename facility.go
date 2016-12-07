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
	"os"
)

// Facility is a destination for log entries. It can have pervasive fields
// added with With().
type Facility interface {
	With(...Field) Facility
	Enabled(Entry) bool
	Log(Entry, ...Field) error
	// XXX idea on how we could restore internal enoding error repuorting:
	//     SetErrorOutput(ws WriteSyncer)
}

// WriterFacility creates a facility that writes logs to an io.Writer. By
// default, if w is nil, os.Stdout is used.
func WriterFacility(enc Encoder, w io.Writer) Facility {
	if w == nil {
		w = os.Stdout
	}
	return ioFacility{
		Encoder: enc,
		Output:  newLockedWriteSyncer(AddSync(w)),
	}
}

type ioFacility struct {
	Encoder Encoder
	Output  WriteSyncer
}

func (iof ioFacility) With(fields ...Field) Facility {
	iof.Encoder = iof.Encoder.Clone()
	addFields(iof.Encoder, fields)
	return iof
}

func (ioFacility) Enabled(Entry) bool { return true }

func (iof ioFacility) Log(ent Entry, fields ...Field) error {
	if err := ent.EncodeTo(iof.Output, iof.Encoder, fields); err != nil {
		return err
	}
	if ent.Level > ErrorLevel {
		// Sync on Panic and Fatal, since they may crash the program.
		iof.Output.Sync()
	}
}
