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
	"fmt"
	"os"
	"time"
)

// For tests.
var _exit = os.Exit

// A Logger enables leveled, structured logging. All methods are safe for
// concurrent use.
type Logger interface {
	// Create a child logger, and optionally add some context to that logger.
	With(...Field) Logger

	// Check returns a CheckedMessage if logging a message at the specified level
	// is enabled. It's a completely optional optimization; in high-performance
	// applications, Check can help avoid allocating a slice to hold fields.
	//
	// See CheckedMessage for an example.
	Check(Level, string) *CheckedMessage

	// Log a message at the given level. Messages include any context that's
	// accumulated on the logger, as well as any fields added at the log site.
	//
	// Calling Panic should panic() and calling Fatal should terminate the
	// process, but calling Log(PanicLevel, ...) or Log(FatalLevel, ...) should
	// not. It may not be possible for compatibility wrappers to comply with
	// this last part (e.g. the bark wrapper).
	Log(Level, string, ...Field) // TODO: can/should we drop this? Or should we change it to take Entry?
	Debug(string, ...Field)
	Info(string, ...Field)
	Warn(string, ...Field)
	Error(string, ...Field)
	DPanic(string, ...Field)
	Panic(string, ...Field)
	Fatal(string, ...Field)
}

type logger struct {
	fac Facility

	enab        LevelEnabler
	development bool
	hooks       []Hook
	errorOutput WriteSyncer
}

// New returns a new logger with sensible defaults: logging at InfoLevel,
// development mode off, errors writtten to stdand error, and logs JSON encoded
// to standard output.
func New(fac Facility, options ...Option) Logger {
	if fac == nil {
		fac = WriterFacility(NewJSONEncoder(), nil)
	}
	log := &logger{
		fac:         fac,
		enab:        InfoLevel,
		errorOutput: newLockedWriteSyncer(os.Stderr),
	}
}

func (log *logger) With(fields ...Field) Logger {
	return &logger{
		fac:         log.fac.With(fields...),
		enab:        log.enab,
		development: log.development,
		hooks:       log.hooks,
		errorOutput: log.errorOutput,
	}
}

func (log *logger) Check(lvl Level, msg string) *Entry {
	ent := Entry{
		Time:    time.Now().UTC(),
		Level:   lvl,
		Message: msg,
	}
	switch lvl {
	case PanicLevel, FatalLevel:
		// Panic and Fatal should always cause a panic/exit, even if the level
		// is disabled.
		break
	case DPanicLevel:
		if log.Development {
			break
		}
		fallthrough
	default:
		if !log.LevelEnabler.Enabled(lvl) {
			return nil
		}
		if !log.Facility.Enabled(ent) {
			return nil
		}
	}
	ent.fac = log.Facility
	return &ent
}

func (log *logger) Debug(msg string, fields ...Field) {
	log.Log(DebugLevel, msg, fields...)
}

func (log *logger) Info(msg string, fields ...Field) {
	log.Log(InfoLevel, msg, fields...)
}

func (log *logger) Warn(msg string, fields ...Field) {
	log.Log(WarnLevel, msg, fields...)
}

func (log *logger) Error(msg string, fields ...Field) {
	log.Log(ErrorLevel, msg, fields...)
}

func (log *logger) DPanic(msg string, fields ...Field) {
	log.Log(DPanicLevel, msg, fields...)
	if log.Development {
		panic(msg)
	}
}

func (log *logger) Panic(msg string, fields ...Field) {
	log.Log(PanicLevel, msg, fields...)
	panic(msg)
}

func (log *logger) Fatal(msg string, fields ...Field) {
	log.Log(FatalLevel, msg, fields...)
	_exit(1)
}

func (log *logger) Log(lvl Level, msg string, fields ...Field) {
	ent := Entry{
		Time:    time.Now().UTC(),
		Level:   lvl,
		Message: msg,
	}
	if !log.LevelEnabler.Enabled(ent.Level) {
		return
	}
	if !log.Facility.Enabled(ent) {
		return
	}
	for _, hook := range log.Hooks {
		if err := hook(&ent); err != nil {
			log.InternalError("hook", err)
		}
	}
	if err := log.Facility.Log(ent, fields...); err != nil {
		log.InternalError("encoder", err)
	}
}

// InternalError prints an internal error message to the configured
// ErrorOutput. This method should only be used to report internal logger
// problems and should not be used to report user-caused problems.
func (log *logger) InternalError(cause string, err error) {
	fmt.Fprintf(log.ErrorOutput, "%v %s error: %v\n", time.Now().UTC(), cause, err)
	log.ErrorOutput.Sync()
}
