package hako

// Value wraps a JavaScript value. Call Free when done unless borrowed.
type Value struct {
	realm    *Realm
	ptr      ValuePtr
	borrowed bool
}

// Pointer returns the raw value pointer.
func (v Value) Pointer() ValuePtr {
	return v.ptr
}

// IsNull returns true if the value is null.
func (v Value) IsNull() bool {
	if v.realm == nil {
		return true
	}
	return v.realm.Runtime.Registry.IsNull(v.realm.Runtime.ctx, v.ptr) != 0
}

// IsUndefined returns true if the value is undefined.
func (v Value) IsUndefined() bool {
	if v.realm == nil {
		return true
	}
	return v.realm.Runtime.Registry.IsUndefined(v.realm.Runtime.ctx, v.ptr) != 0
}

// String returns the string representation of the value.
func (v Value) String() string {
	if v.realm == nil {
		return ""
	}

	ctx := v.realm.Runtime.ctx
	reg := v.realm.Runtime.Registry
	mem := v.realm.Runtime.Memory

	strPtr := reg.ToCString(ctx, v.realm.Pointer, v.ptr)
	if strPtr == 0 {
		return ""
	}

	s := mem.ReadNullTerminatedString(MemoryPtr(strPtr))
	mem.FreeCString(v.realm.Pointer, strPtr)

	return s
}

// AsNumber returns the value as a float64.
func (v Value) AsNumber() float64 {
	if v.realm == nil {
		return 0
	}
	return v.realm.Runtime.Registry.GetFloat64(v.realm.Runtime.ctx, v.realm.Pointer, v.ptr)
}

// Dup duplicates the value (increases reference count).
func (v Value) Dup() Value {
	if v.realm == nil {
		return Value{}
	}
	ptr := v.realm.Runtime.Memory.DupValuePointer(v.realm.Pointer, v.ptr)
	return Value{realm: v.realm, ptr: ptr}
}

// Free releases the value.
func (v Value) Free() {
	if v.realm == nil || v.ptr == 0 || v.borrowed {
		return
	}
	v.realm.Runtime.Memory.FreeValuePointer(v.realm.Pointer, v.ptr)
}
