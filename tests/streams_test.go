//go:build windows

package tests

import (
	"bytes"
	"testing"

	"github.com/zandercodes/gowinrt/windows/storage/streams"
)

func TestBuffer_CreateAndCapacity(t *testing.T) {
	buf, err := streams.BufferCreate(1024)
	if err != nil {
		t.Fatalf("BufferCreate: %v", err)
	}

	cap, err := buf.GetCapacity()
	if err != nil {
		t.Fatalf("GetCapacity: %v", err)
	}
	if cap != 1024 {
		t.Errorf("Capacity = %d, want 1024", cap)
	}

	length, err := buf.GetLength()
	if err != nil {
		t.Fatalf("GetLength: %v", err)
	}
	if length != 0 {
		t.Errorf("Length = %d, want 0", length)
	}
}

func TestBuffer_SetAndGetLength(t *testing.T) {
	buf, err := streams.BufferCreate(1024)
	if err != nil {
		t.Fatalf("BufferCreate: %v", err)
	}

	if err := buf.SetLength(512); err != nil {
		t.Fatalf("SetLength: %v", err)
	}

	length, err := buf.GetLength()
	if err != nil {
		t.Fatalf("GetLength: %v", err)
	}
	if length != 512 {
		t.Errorf("Length = %d, want 512", length)
	}
}

func TestDataWriter_CreateAndDetach(t *testing.T) {
	writer, err := streams.NewDataWriter()
	if err != nil {
		t.Fatalf("NewDataWriter: %v", err)
	}

	data := []uint8{1, 2, 3, 4, 5}
	if err := writer.WriteBytes(uint32(len(data)), data); err != nil {
		t.Fatalf("WriteBytes: %v", err)
	}

	buf, err := writer.DetachBuffer()
	if err != nil {
		t.Fatalf("DetachBuffer: %v", err)
	}
	if buf == nil {
		t.Fatal("DetachBuffer returned nil")
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestDataWriter_DataReader_Roundtrip(t *testing.T) {
	original := []uint8{10, 20, 30, 40, 50, 60, 70, 80}

	// Write bytes via DataWriter
	writer, err := streams.NewDataWriter()
	if err != nil {
		t.Fatalf("NewDataWriter: %v", err)
	}

	if err := writer.WriteBytes(uint32(len(original)), original); err != nil {
		t.Fatalf("WriteBytes: %v", err)
	}

	ibuf, err := writer.DetachBuffer()
	if err != nil {
		t.Fatalf("DetachBuffer: %v", err)
	}

	// Read bytes back via DataReader
	reader, err := streams.DataReaderFromBuffer(ibuf)
	if err != nil {
		t.Fatalf("DataReaderFromBuffer: %v", err)
	}

	result, err := reader.ReadBytes(uint32(len(original)))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}

	if !bytes.Equal(result, original) {
		t.Errorf("Roundtrip failed:\n  got:  %v\n  want: %v", result, original)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close: %v", err)
	}
}

func TestSignatureConstants_Streams(t *testing.T) {
	tests := []struct {
		name, got, want string
	}{
		{"SignatureBuffer", streams.SignatureBuffer, "rc(Windows.Storage.Streams.Buffer;{905a0fe0-bc53-11df-8c49-001e4fc686da})"},
		{"SignatureDataWriter", streams.SignatureDataWriter, "rc(Windows.Storage.Streams.DataWriter;{64b89265-d341-4922-b38a-dd4af8808c4e})"},
		{"SignatureDataReader", streams.SignatureDataReader, "rc(Windows.Storage.Streams.DataReader;{e2b50029-b4c1-4314-a4b8-fb813a2f275e})"},
		{"GUIDIBuffer", streams.GUIDIBuffer, "905a0fe0-bc53-11df-8c49-001e4fc686da"},
		{"GUIDIDataWriter", streams.GUIDIDataWriter, "64b89265-d341-4922-b38a-dd4af8808c4e"},
		{"GUIDIDataReader", streams.GUIDIDataReader, "e2b50029-b4c1-4314-a4b8-fb813a2f275e"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
			}
		})
	}
}
