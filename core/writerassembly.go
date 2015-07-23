// Copyright 2015 trivago GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package core

import (
	"github.com/trivago/gollum/core/log"
	"io"
)

// WriterAssembly is a helper struct for io.Writer compatible classes that use
// messagebatch.
type WriterAssembly struct {
	writer       io.Writer
	flush        func(Message)
	formatter    Formatter
	dropStreamID MessageStreamID
	buffer       []byte
	validate     func() bool
	handleError  func(error) bool
}

// NewWriterAssembly creates a new adapter between io.Writer and the MessageBatch
// AssemblyFunc function signature
func NewWriterAssembly(writer io.Writer, flush func(Message), formatter Formatter) WriterAssembly {
	return WriterAssembly{
		writer:    writer,
		formatter: formatter,
		flush:     flush,
	}
}

// SetValidator sets a callback that is called if a write was successfull.
// Validate needs to return true to prevent messages to be flushed.
func (asm *WriterAssembly) SetValidator(validate func() bool) {
	asm.validate = validate
}

// SetErrorHandler sets a callback that is called if an error occured.
// HandleError needs to return true to prevent messages to be flushed.
func (asm *WriterAssembly) SetErrorHandler(handleError func(error) bool) {
	asm.handleError = handleError
}

// SetWriter changes the writer interface used during Assemble
func (asm *WriterAssembly) SetWriter(writer io.Writer) {
	asm.writer = writer
}

// SetFlush changes the bound flush function
func (asm *WriterAssembly) SetFlush(flush func(Message)) {
	asm.flush = flush
}

// Write is an AssemblyFunc compatible implementation to pass all messages from
// a MessageBatch to an io.Writer.
// Messages are formatted using a given formatter. If the io.Writer fails to
// write the assembled buffer all messages are passed to the FLush() method.
func (asm *WriterAssembly) Write(messages []Message) {
	contentLen := 0
	for _, msg := range messages {
		payload, _ := asm.formatter.Format(msg)
		if contentLen+len(payload) > len(asm.buffer) {
			asm.buffer = append(asm.buffer[:contentLen], payload...)
		} else {
			copy(asm.buffer[contentLen:], payload)
		}
		contentLen += len(payload)
	}

	// Route all messages if they could not be written
	if _, err := asm.writer.Write(asm.buffer[:contentLen]); err != nil {
		Log.Error.Print("Stream write error:", err)
		if asm.handleError == nil || !asm.handleError(err) {
			asm.Flush(messages)
		}
		return // ### return, error handled ###
	}

	if asm.validate != nil && !asm.validate() {
		asm.Flush(messages)
	}
}

// Flush is an AssemblyFunc compatible implementation to pass all messages from
// a MessageBatch to e.g. the Drop function of a producer.
// Flush will also be called by Write if the io.Writer reported an error.
func (asm *WriterAssembly) Flush(messages []Message) {
	for _, msg := range messages {
		asm.flush(msg)
	}
}