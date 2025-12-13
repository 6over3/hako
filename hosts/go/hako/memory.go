package hako

import (
	"github.com/tetratelabs/wazero/api"
)

// Memory provides raw access to WASM linear memory.
type Memory struct {
	mem api.Memory
}

// NewMemory wraps a wazero memory instance.
func NewMemory(mem api.Memory) *Memory {
	return &Memory{mem: mem}
}

// ReadBytes reads n bytes from the given offset.
func (m *Memory) ReadBytes(offset MemoryPtr, n uint32) ([]byte, bool) {
	return m.mem.Read(uint32(offset), n)
}

// WriteBytes writes bytes to the given offset.
func (m *Memory) WriteBytes(offset MemoryPtr, data []byte) bool {
	return m.mem.Write(uint32(offset), data)
}

// ReadUint32 reads a little-endian uint32 from the given offset.
func (m *Memory) ReadUint32(offset MemoryPtr) (uint32, bool) {
	return m.mem.ReadUint32Le(uint32(offset))
}

// WriteUint32 writes a little-endian uint32 to the given offset.
func (m *Memory) WriteUint32(offset MemoryPtr, val uint32) bool {
	return m.mem.WriteUint32Le(uint32(offset), val)
}

// ReadString reads a null-terminated string from the given offset.
func (m *Memory) ReadString(offset MemoryPtr) (string, bool) {
	data, ok := m.mem.Read(uint32(offset), m.mem.Size()-uint32(offset))
	if !ok {
		return "", false
	}
	for i, b := range data {
		if b == 0 {
			return string(data[:i]), true
		}
	}
	return string(data), true
}

// WriteString writes a null-terminated string and returns the number of bytes written.
func (m *Memory) WriteString(offset MemoryPtr, s string) bool {
	data := append([]byte(s), 0)
	return m.mem.Write(uint32(offset), data)
}
