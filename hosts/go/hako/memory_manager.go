package hako

import (
	"context"
)

// MemoryManager wraps memory operations with QuickJS allocation.
type MemoryManager struct {
	registry *Registry
	memory   *Memory
	ctx      context.Context
}

// NewMemoryManager creates a new MemoryManager.
func NewMemoryManager(registry *Registry, memory *Memory, ctx context.Context) *MemoryManager {
	return &MemoryManager{
		registry: registry,
		memory:   memory,
		ctx:      ctx,
	}
}

// AllocateMemory allocates memory using the JS context allocator.
func (m *MemoryManager) AllocateMemory(ctxPtr ContextPtr, size int32) MemoryPtr {
	return MemoryPtr(m.registry.Malloc(m.ctx, ctxPtr, size))
}

// FreeMemory frees memory allocated via AllocateMemory.
func (m *MemoryManager) FreeMemory(ctxPtr ContextPtr, ptr MemoryPtr) {
	if ptr != 0 {
		m.registry.Free(m.ctx, ctxPtr, ptr)
	}
}

// AllocateString allocates and writes a null-terminated string, returns pointer and length.
func (m *MemoryManager) AllocateString(ctxPtr ContextPtr, s string) (MemoryPtr, int) {
	data := []byte(s)
	size := int32(len(data) + 1)
	ptr := m.AllocateMemory(ctxPtr, size)
	if ptr == 0 {
		return 0, 0
	}

	m.memory.WriteBytes(ptr, append(data, 0))
	return ptr, len(data)
}

// ReadNullTerminatedString reads a null-terminated string from memory.
func (m *MemoryManager) ReadNullTerminatedString(ptr MemoryPtr) string {
	if ptr == 0 {
		return ""
	}
	s, _ := m.memory.ReadString(ptr)
	return s
}

// FreeCString frees a string allocated by QuickJS.
func (m *MemoryManager) FreeCString(ctxPtr ContextPtr, ptr int32) {
	if ptr != 0 {
		m.registry.FreeCString(m.ctx, ctxPtr, ptr)
	}
}

// FreeValuePointer frees a JSValue pointer.
func (m *MemoryManager) FreeValuePointer(ctxPtr ContextPtr, ptr ValuePtr) {
	if ptr != 0 {
		m.registry.FreeValuePointer(m.ctx, ctxPtr, ptr)
	}
}

// DupValuePointer duplicates a JSValue pointer.
func (m *MemoryManager) DupValuePointer(ctxPtr ContextPtr, ptr ValuePtr) ValuePtr {
	return m.registry.DupValuePointer(m.ctx, ctxPtr, ptr)
}

// WriteBytes writes bytes to memory at the given offset.
func (m *MemoryManager) WriteBytes(ptr MemoryPtr, data []byte) bool {
	return m.memory.WriteBytes(ptr, data)
}

// ReadBytes reads bytes from memory.
func (m *MemoryManager) ReadBytes(ptr MemoryPtr, n uint32) ([]byte, bool) {
	return m.memory.ReadBytes(ptr, n)
}

// WriteUint32 writes a uint32 to memory.
func (m *MemoryManager) WriteUint32(ptr MemoryPtr, val uint32) bool {
	return m.memory.WriteUint32(ptr, val)
}

// ReadUint32 reads a uint32 from memory.
func (m *MemoryManager) ReadUint32(ptr MemoryPtr) (uint32, bool) {
	return m.memory.ReadUint32(ptr)
}
