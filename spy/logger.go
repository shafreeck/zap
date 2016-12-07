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

package spy

import (
	"sync"

	"github.com/uber-go/zap"
)

// A Log is an encoding-agnostic representation of a log message.
type Log struct {
	Level  zap.Level
	Msg    string
	Fields []zap.Field
}

// A Sink stores Log structs.
type Sink struct {
	sync.Mutex

	logs []Log
}

// WriteLog writes a log message to the LogSink.
func (s *Sink) WriteLog(lvl zap.Level, msg string, fields []zap.Field) {
	s.Lock()
	log := Log{
		Msg:    msg,
		Level:  lvl,
		Fields: fields,
	}
	s.logs = append(s.logs, log)
	s.Unlock()
}

// Logs returns a copy of the sink's accumulated logs.
func (s *Sink) Logs() []Log {
	var logs []Log
	s.Lock()
	logs = append(logs, s.logs...)
	s.Unlock()
	return logs
}

// Facility implements a zap.Facility that captures Log records.
type Facility struct {
	sync.Mutex
	enab    zap.LevelEnabler
	sink    *Sink
	context []zap.Field
}

// With creates a sub spy facility so that all log records recorded under it
// have the given fields attached.
func (sf *Facility) With(fields ...zap.Field) zap.Facility {
	return &Facility{
		sink:    sf.sink,
		context: append(sf.context, fields...),
	}
}

// Enabled always returns true.
func (sf *Facility) Enabled(ent zap.Entry) bool {
	return sf.enab.Enabled(ent.Level)
}

// Log collects all contextual fields, an records the Log record.
func (sf *Facility) Log(ent zap.Entry, fields ...zap.Field) {
	all := make([]zap.Field, 0, len(fields)+len(sf.context))
	all = append(all, sf.context...)
	all = append(all, fields...)
	sf.sink.WriteLog(ent.Level, ent.Message, all)
}

// New creates a new Facility and returns it and its associated Sink.
func New(enab zap.LevelEnabler) (*Facility, *Sink) {
	fac := &Facility{
		enab: enab,
		sink: &Sink{},
	}
	return fac, fac.sink
}
