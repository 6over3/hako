// Package hako provides a Go host for QuickJS via WebAssembly.
package hako

import "fmt"

// RuntimePtr is a pointer to a JSRuntime in WASM memory.
// JSRuntime is the top-level QuickJS runtime that manages memory and contexts.
type RuntimePtr int32

// IsNull reports whether the pointer is null (zero).
func (p RuntimePtr) IsNull() bool { return p == 0 }

func (p RuntimePtr) String() string { return fmt.Sprintf("RuntimePtr(0x%x)", int32(p)) }

// ContextPtr is a pointer to a JSContext in WASM memory.
// JSContext is a JavaScript execution context (realm) with its own global object.
type ContextPtr int32

// IsNull reports whether the pointer is null (zero).
func (p ContextPtr) IsNull() bool { return p == 0 }

func (p ContextPtr) String() string { return fmt.Sprintf("ContextPtr(0x%x)", int32(p)) }

// ValuePtr is a pointer to a JSValue in WASM memory.
// JSValue represents any JavaScript value (number, string, object, function, etc.).
type ValuePtr int32

// IsNull reports whether the pointer is null (zero).
func (p ValuePtr) IsNull() bool { return p == 0 }

func (p ValuePtr) String() string { return fmt.Sprintf("ValuePtr(0x%x)", int32(p)) }

// ModuleDefPtr is a pointer to a JSModuleDef in WASM memory.
// JSModuleDef represents an ES6 module definition.
type ModuleDefPtr int32

// IsNull reports whether the pointer is null (zero).
func (p ModuleDefPtr) IsNull() bool { return p == 0 }

func (p ModuleDefPtr) String() string { return fmt.Sprintf("ModuleDefPtr(0x%x)", int32(p)) }

// ClassID is a QuickJS class identifier used for custom JS classes.
type ClassID int32

// IsValid reports whether the class ID is valid (non-zero).
func (id ClassID) IsValid() bool { return id != 0 }

func (id ClassID) String() string { return fmt.Sprintf("ClassID(%d)", int32(id)) }

// MemoryPtr is a pointer to allocated memory in WASM linear memory.
type MemoryPtr int32

// IsNull reports whether the pointer is null (zero).
func (p MemoryPtr) IsNull() bool { return p == 0 }

func (p MemoryPtr) String() string { return fmt.Sprintf("MemoryPtr(0x%x)", int32(p)) }
