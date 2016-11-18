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

	"github.com/uber-go/atomic"
)

// Meta is implementation-agnostic state management for Loggers. Most Logger
// implementations can reduce the required boilerplate by embedding a Meta.
//
// Note that while the level-related fields and methods are safe for concurrent
// use, the remaining fields are not.
type Meta struct {
	Development bool
	Encoder     Encoder
	Hooks       []Hook
	Output      WriteSyncer
	ErrorOutput WriteSyncer

	lvl *atomic.Int32
}

// MakeMeta returns a new meta struct with sensible defaults: logging at
// InfoLevel, development mode off, and writing to standard error and standard
// out.
func MakeMeta(enc Encoder, options ...Option) Meta {
	m := Meta{
		lvl:         atomic.NewInt32(int32(InfoLevel)),
		Encoder:     enc,
		Output:      newLockedWriteSyncer(os.Stdout),
		ErrorOutput: newLockedWriteSyncer(os.Stderr),
	}
	for _, opt := range options {
		opt.apply(&m)
	}
	return m
}

// Level returns the minimum enabled log level. It's safe to call concurrently.
func (m Meta) Level() Level {
	return Level(m.lvl.Load())
}

// SetLevel atomically alters the the logging level for this Meta and all its
// clones.
func (m Meta) SetLevel(lvl Level) {
	m.lvl.Store(int32(lvl))
}

// Clone creates a copy of the meta struct. It deep-copies the encoder, but not
// the hooks (since they rarely change).
func (m Meta) Clone() Meta {
	m.Encoder = m.Encoder.Clone()
	return m
}

// Configure creates a copy of the meta object, using Clone(), and applies any
// given options.
func (m Meta) Configure(options ...Option) Meta {
	m = m.Clone()
	for _, opt := range options {
		opt.apply(&m)
	}
	return m
}

// Enabled returns true if logging a message at a particular level is enabled.
func (m Meta) Enabled(lvl Level) bool {
	return lvl >= m.Level()
}

// Check returns a CheckedMessage logging the given message is Enabled, nil
// otherwise.
func (m Meta) Check(log Logger, lvl Level, msg string) *CheckedMessage {
	switch lvl {
	case PanicLevel, FatalLevel:
		// Panic and Fatal should always cause a panic/exit, even if the level
		// is disabled.
		break
	default:
		if !m.Enabled(lvl) {
			return nil
		}
	}
	return NewCheckedMessage(log, lvl, msg)
}

// InternalError prints an internal error message to the configured
// ErrorOutput. This method should only be used to report internal logger
// problems and should not be used to report user-caused problems.
func (m Meta) InternalError(cause string, err error) {
	fmt.Fprintf(m.ErrorOutput, "%v %s error: %v\n", _timeNow().UTC(), cause, err)
	m.ErrorOutput.Sync()
}
