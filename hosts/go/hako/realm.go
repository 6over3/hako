package hako

import (
	"fmt"
)

// Realm is a JavaScript execution context with its own global object.
type Realm struct {
	Pointer  ContextPtr
	Runtime  *Runtime
	disposed bool
}

// EvalOptions configures code evaluation.
type EvalOptions struct {
	Filename     string
	DetectModule bool
}

// EvalCode evaluates JavaScript code and returns the result.
func (r *Realm) EvalCode(code string) (Value, error) {
	return r.EvalCodeWithOptions(code, nil)
}

// EvalCodeWithOptions evaluates JavaScript code with options.
func (r *Realm) EvalCodeWithOptions(code string, opts *EvalOptions) (Value, error) {
	if r.disposed {
		return Value{}, fmt.Errorf("realm is disposed")
	}

	if len(code) == 0 {
		return r.Undefined(), nil
	}

	if opts == nil {
		opts = &EvalOptions{Filename: "eval", DetectModule: true}
	}
	if opts.Filename == "" {
		opts.Filename = "eval"
	}

	ctx := r.Runtime.ctx
	reg := r.Runtime.Registry
	mem := r.Runtime.Memory

	// Allocate code string (null-terminated)
	codePtr, codeLen := mem.AllocateString(r.Pointer, code)
	if codePtr == 0 {
		return Value{}, fmt.Errorf("failed to allocate code string")
	}
	defer mem.FreeMemory(r.Pointer, codePtr)

	// Allocate filename string
	filenamePtr, _ := mem.AllocateString(r.Pointer, opts.Filename)
	if filenamePtr == 0 {
		return Value{}, fmt.Errorf("failed to allocate filename string")
	}
	defer mem.FreeMemory(r.Pointer, filenamePtr)

	detectModule := int32(0)
	if opts.DetectModule {
		detectModule = 1
	}

	// Call eval
	resultPtr := reg.Eval(ctx, r.Pointer, int32(codePtr), int32(codeLen), int32(filenamePtr), detectModule, 0)

	// Check for exception
	errPtr := reg.GetLastError(ctx, r.Pointer, resultPtr)
	if !errPtr.IsNull() {
		mem.FreeValuePointer(r.Pointer, resultPtr)

		// Get error message
		errVal := Value{realm: r, ptr: errPtr}
		errMsg := errVal.String()
		errVal.Free()

		return Value{}, fmt.Errorf("%s", errMsg)
	}

	return Value{realm: r, ptr: resultPtr}, nil
}

// GetGlobalObject returns the global object.
func (r *Realm) GetGlobalObject() Value {
	ptr := r.Runtime.Registry.GetGlobalObject(r.Runtime.ctx, r.Pointer)
	return Value{realm: r, ptr: ptr}
}

// Undefined returns the undefined value.
func (r *Realm) Undefined() Value {
	ptr := r.Runtime.Registry.GetUndefined(r.Runtime.ctx)
	return Value{realm: r, ptr: ptr, borrowed: true}
}

// Null returns the null value.
func (r *Realm) Null() Value {
	ptr := r.Runtime.Registry.GetNull(r.Runtime.ctx)
	return Value{realm: r, ptr: ptr, borrowed: true}
}

// NewString creates a new JS string value.
func (r *Realm) NewString(s string) Value {
	mem := r.Runtime.Memory
	strPtr, _ := mem.AllocateString(r.Pointer, s)
	defer mem.FreeMemory(r.Pointer, strPtr)

	ptr := r.Runtime.Registry.NewString(r.Runtime.ctx, r.Pointer, int32(strPtr))
	return Value{realm: r, ptr: ptr}
}

// NewNumber creates a new JS number value.
func (r *Realm) NewNumber(n float64) Value {
	ptr := r.Runtime.Registry.NewFloat64(r.Runtime.ctx, r.Pointer, n)
	return Value{realm: r, ptr: ptr}
}

// NewObject creates a new JS object.
func (r *Realm) NewObject() Value {
	ptr := r.Runtime.Registry.NewObject(r.Runtime.ctx, r.Pointer)
	return Value{realm: r, ptr: ptr}
}

// NewArray creates a new JS array.
func (r *Realm) NewArray() Value {
	ptr := r.Runtime.Registry.NewArray(r.Runtime.ctx, r.Pointer)
	return Value{realm: r, ptr: ptr}
}

// Close releases the realm resources.
func (r *Realm) Close() {
	if r.disposed {
		return
	}

	r.Runtime.Callbacks.UnregisterContext(r.Pointer)
	r.dispose()
	r.Runtime.dropRealm(r)
}

func (r *Realm) dispose() {
	if r.disposed {
		return
	}
	r.disposed = true

	if r.Pointer != 0 {
		r.Runtime.Registry.FreeContext(r.Runtime.ctx, r.Pointer)
		r.Pointer = 0
	}
}
