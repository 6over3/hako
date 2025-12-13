package hako

import (
	"context"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// CallbackManager routes WASM callbacks to the appropriate Realm/Runtime.
type CallbackManager struct {
	mu       sync.RWMutex
	contexts map[ContextPtr]*Realm
	runtimes map[RuntimePtr]*Runtime
	memory   *MemoryManager
	registry *Registry
}

// NewCallbackManager creates a new CallbackManager.
func NewCallbackManager() *CallbackManager {
	return &CallbackManager{
		contexts: make(map[ContextPtr]*Realm),
		runtimes: make(map[RuntimePtr]*Runtime),
	}
}

// Initialize sets the registry and memory manager references.
func (cm *CallbackManager) Initialize(registry *Registry, memory *MemoryManager) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.registry = registry
	cm.memory = memory
}

// RegisterContext registers a realm for callback routing.
func (cm *CallbackManager) RegisterContext(ptr ContextPtr, realm *Realm) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.contexts[ptr] = realm
}

// UnregisterContext removes a realm from the registry.
func (cm *CallbackManager) UnregisterContext(ptr ContextPtr) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.contexts, ptr)
}

// RegisterRuntime registers a runtime for callback routing.
func (cm *CallbackManager) RegisterRuntime(ptr RuntimePtr, rt *Runtime) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.runtimes[ptr] = rt
}

// UnregisterRuntime removes a runtime from the registry.
func (cm *CallbackManager) UnregisterRuntime(ptr RuntimePtr) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.runtimes, ptr)
}

// AddToHostModule adds callback functions to an existing host module builder.
// Signatures must match bindings.json imports exactly.
func (cm *CallbackManager) AddToHostModule(ctx context.Context, builder wazero.HostModuleBuilder, mem *Memory) (api.Closer, error) {
	return builder.
		// call_function: (i32, i32, i32, i32, i32) -> i32
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, jsCtx, funcID, thisArg, argc, argv int32) int32 {
			return int32(cm.handleCallFunction(ContextPtr(jsCtx), funcID, thisArg, argc, argv))
		}).
		Export("call_function").
		// interrupt_handler: (i32, i32, i32) -> i32
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, rt, opaque, unused int32) int32 {
			if cm.handleInterrupt(RuntimePtr(rt), opaque) {
				return 1
			}
			return 0
		}).
		Export("interrupt_handler").
		// normalize_module: (i32, i32, i32, i32, i32) -> i32
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, jsCtx, baseNamePtr, namePtr, opaque, outPtr int32) int32 {
			baseName, _ := mem.ReadString(MemoryPtr(baseNamePtr))
			name, _ := mem.ReadString(MemoryPtr(namePtr))
			result := cm.handleNormalizeModule(ContextPtr(jsCtx), baseName, name, opaque)
			mem.WriteBytes(MemoryPtr(outPtr), []byte(result))
			return int32(len(result))
		}).
		Export("normalize_module").
		// load_module: (i32, i32, i32, i32, i32) -> i32
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, rt, jsCtx, moduleNamePtr, opaque, outPtr int32) int32 {
			moduleName, _ := mem.ReadString(MemoryPtr(moduleNamePtr))
			srcType, srcPtr, srcLen := cm.handleLoadModule(RuntimePtr(rt), ContextPtr(jsCtx), moduleName, opaque)
			// Write result struct: type (i32), ptr (i32), len (i32)
			mem.WriteUint32(MemoryPtr(outPtr), uint32(srcType))
			mem.WriteUint32(MemoryPtr(outPtr+4), uint32(srcPtr))
			mem.WriteUint32(MemoryPtr(outPtr+8), uint32(srcLen))
			return int32(srcType)
		}).
		Export("load_module").
		// module_init: (i32, i32) -> i32
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, jsCtx, m int32) int32 {
			return cm.handleModuleInit(ContextPtr(jsCtx), ModuleDefPtr(m))
		}).
		Export("module_init").
		// class_finalizer: (i32, i32, i32) -> nil
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, rt, opaque, classID int32) {
			cm.handleClassFinalizer(RuntimePtr(rt), opaque, ClassID(classID))
		}).
		Export("class_finalizer").
		// class_gc_mark: (i32, i32, i32, i32) -> nil
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, rt, val, markFunc, classID int32) {
			cm.handleClassGCMark(RuntimePtr(rt), ValuePtr(val), markFunc, ClassID(classID))
		}).
		Export("class_gc_mark").
		// class_constructor: (i32, i32, i32, i32, i32) -> i32
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, jsCtx, newTarget, argc, argv, classID int32) int32 {
			return int32(cm.handleClassConstructor(ContextPtr(jsCtx), ValuePtr(newTarget), ClassID(classID)))
		}).
		Export("class_constructor").
		// promise_rejection_tracker: (i32, i32, i32, i32, i32) -> nil
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, jsCtx, promise, reason, isHandled, opaque int32) {
			cm.handlePromiseRejectionTracker(ContextPtr(jsCtx), ValuePtr(promise), ValuePtr(reason), isHandled != 0, opaque)
		}).
		Export("promise_rejection_tracker").
		Instantiate(ctx)
}

// ModuleSourceType indicates the type of module source.
type ModuleSourceType int32

const (
	ModuleSourceString      ModuleSourceType = 0
	ModuleSourcePrecompiled ModuleSourceType = 1
	ModuleSourceError       ModuleSourceType = 2
)

func (cm *CallbackManager) handleCallFunction(ctx ContextPtr, funcID, thisArg, argc, argv int32) ValuePtr {
	return 0
}

func (cm *CallbackManager) handleInterrupt(rt RuntimePtr, opaque int32) bool {
	return false
}

func (cm *CallbackManager) handleLoadModule(rt RuntimePtr, ctx ContextPtr, moduleName string, opaque int32) (ModuleSourceType, MemoryPtr, int32) {
	return ModuleSourceError, 0, 0
}

func (cm *CallbackManager) handleNormalizeModule(ctx ContextPtr, baseName, name string, opaque int32) string {
	return name
}

func (cm *CallbackManager) handleModuleInit(ctx ContextPtr, m ModuleDefPtr) int32 {
	return 0
}

func (cm *CallbackManager) handleClassConstructor(ctx ContextPtr, newTarget ValuePtr, classID ClassID) ValuePtr {
	return 0
}

func (cm *CallbackManager) handleClassFinalizer(rt RuntimePtr, opaque int32, classID ClassID) {
}

func (cm *CallbackManager) handleClassGCMark(rt RuntimePtr, val ValuePtr, markFunc int32, classID ClassID) {
}

func (cm *CallbackManager) handlePromiseRejectionTracker(ctx ContextPtr, promise, reason ValuePtr, isHandled bool, opaque int32) {
}
