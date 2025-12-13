package hako

import (
	"context"
	"fmt"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// Runtime manages a QuickJS instance running in WebAssembly.
//
// Runtime is the top-level entry point, equivalent to HakoRuntime in the C# host.
// It owns the wazero runtime, the QuickJS WASM module, and all created realms.
//
// Use [New] to create a Runtime, [CreateRealm] to create execution contexts,
// and [Close] to release resources.
//
// # Thread Safety
//
// Runtime methods are safe for concurrent use. Each Realm should typically be
// used from a single goroutine, similar to how QuickJS contexts work.
//
// # Comparison to C# Host
//
// This corresponds to HakoRuntime in the C# implementation. Key differences:
//   - Uses wazero instead of wasmtime
//   - Memory is exported via a separate WASM module (see memory workaround below)
//   - Callbacks are registered on a "hako" host module
type Runtime struct {
	// Pointer is the QuickJS JSRuntime pointer in WASM memory.
	Pointer RuntimePtr

	// Registry provides access to all QuickJS WASM exports.
	// Use this for low-level operations not exposed by higher-level APIs.
	Registry *Registry

	// Callbacks handles host function calls from JavaScript.
	// Extend this to implement custom host functions.
	Callbacks *CallbackManager

	// Memory provides memory allocation in WASM linear memory.
	Memory *MemoryManager

	ctx    context.Context
	wazero wazero.Runtime
	module api.Module

	mu       sync.RWMutex
	realms   map[ContextPtr]*Realm
	disposed bool
}

// Options configures Runtime creation.
type Options struct {
	// MemoryLimitBytes sets the QuickJS memory limit.
	// Zero means no limit.
	MemoryLimitBytes int32
}

// New creates a Runtime from a QuickJS WASM binary.
//
// The wasmBytes should be the contents of hako.wasm (or a compatible build).
// The context is used for all WASM operations and should remain valid for
// the lifetime of the Runtime.
//
// Call [Runtime.Close] to release resources when done.
func New(ctx context.Context, wasmBytes []byte, opts *Options) (*Runtime, error) {
	if opts == nil {
		opts = &Options{}
	}

	// QuickJS WASM uses tail calls, which requires experimental wazero support.
	cfg := wazero.NewRuntimeConfig().
		WithCoreFeatures(api.CoreFeaturesV2 | experimental.CoreFeaturesTailCall)
	wzr := wazero.NewRuntimeWithConfig(ctx, cfg)

	if _, err := wasi_snapshot_preview1.Instantiate(ctx, wzr); err != nil {
		wzr.Close(ctx)
		return nil, fmt.Errorf("instantiate WASI: %w", err)
	}

	compiled, err := wzr.CompileModule(ctx, wasmBytes)
	if err != nil {
		wzr.Close(ctx)
		return nil, fmt.Errorf("compile module: %w", err)
	}

	callbacks := NewCallbackManager()
	rawMem := &Memory{}

	// Memory Export Workaround
	//
	// wazero's HostModuleBuilder can only export Go functions, not memory.
	// But hako.wasm imports:
	//   - Memory from "env.memory"
	//   - Callback functions from "hako.*"
	//
	// Solution: Create three WASM modules:
	//   1. "env" - A minimal WASM module that only exports memory
	//   2. "hako" - A Go host module that exports callback functions
	//   3. "hako_quickjs" - The actual QuickJS WASM module
	//
	// The env module is a hand-crafted WASM binary equivalent to:
	//   (module (memory (export "memory") 384 4096))
	//
	// Memory requirements:
	//   - 384 pages minimum (24 MiB) - required by hako.wasm
	//   - 4096 pages maximum (256 MiB)
	envWasm := []byte{
		0x00, 0x61, 0x73, 0x6d, // magic: \0asm
		0x01, 0x00, 0x00, 0x00, // version: 1
		0x05, 0x06, 0x01, // memory section
		0x01, 0x80, 0x03, // limits: min=384
		0x80, 0x20, // limits: max=4096
		0x07, 0x0a, 0x01, // export section
		0x06, 0x6d, 0x65, 0x6d, 0x6f, 0x72, 0x79, // "memory"
		0x02, 0x00, // memory index 0
	}

	envCompiled, err := wzr.CompileModule(ctx, envWasm)
	if err != nil {
		wzr.Close(ctx)
		return nil, fmt.Errorf("compile env module: %w", err)
	}

	envMod, err := wzr.InstantiateModule(ctx, envCompiled, wazero.NewModuleConfig().WithName("env"))
	if err != nil {
		wzr.Close(ctx)
		return nil, fmt.Errorf("instantiate env module: %w", err)
	}
	rawMem.mem = envMod.Memory()

	// Create the "hako" host module with callback functions.
	hakoHostBuilder := wzr.NewHostModuleBuilder("hako")
	if _, err = callbacks.AddToHostModule(ctx, hakoHostBuilder, rawMem); err != nil {
		wzr.Close(ctx)
		return nil, fmt.Errorf("bind callbacks: %w", err)
	}

	// Instantiate QuickJS as "hako_quickjs" (not "hako" to avoid name conflict).
	module, err := wzr.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().
		WithName("hako_quickjs").
		WithStartFunctions("_initialize"))
	if err != nil {
		wzr.Close(ctx)
		return nil, fmt.Errorf("instantiate module: %w", err)
	}

	// Update memory reference to the instantiated module's memory.
	rawMem.mem = module.Memory()

	registry, err := NewRegistry(module)
	if err != nil {
		wzr.Close(ctx)
		return nil, fmt.Errorf("create registry: %w", err)
	}

	rtPtr := registry.NewRuntime(ctx)
	if rtPtr.IsNull() {
		wzr.Close(ctx)
		return nil, fmt.Errorf("create QuickJS runtime: NewRuntime returned null")
	}

	rt := &Runtime{
		Pointer:   rtPtr,
		Registry:  registry,
		Callbacks: callbacks,
		Memory:    NewMemoryManager(registry, rawMem, ctx),
		ctx:       ctx,
		wazero:    wzr,
		module:    module,
		realms:    make(map[ContextPtr]*Realm),
	}

	callbacks.Initialize(registry, rt.Memory)
	callbacks.RegisterRuntime(rtPtr, rt)

	if opts.MemoryLimitBytes > 0 {
		rt.SetMemoryLimit(opts.MemoryLimitBytes)
	}

	return rt, nil
}

// CreateRealm creates a new JavaScript execution context.
//
// Each Realm has its own global object and built-in objects. Multiple Realms
// can exist within a single Runtime for sandboxed execution.
//
// Call [Realm.Close] when done to free resources.
func (rt *Runtime) CreateRealm() (*Realm, error) {
	if rt.disposed {
		return nil, fmt.Errorf("runtime is disposed")
	}

	ctxPtr := rt.Registry.NewContext(rt.ctx, rt.Pointer, 0)
	if ctxPtr.IsNull() {
		return nil, fmt.Errorf("create context: NewContext returned null")
	}

	realm := &Realm{
		Pointer: ctxPtr,
		Runtime: rt,
	}

	rt.mu.Lock()
	rt.realms[ctxPtr] = realm
	rt.mu.Unlock()

	rt.Callbacks.RegisterContext(ctxPtr, realm)

	return realm, nil
}

// SetMemoryLimit sets the QuickJS memory limit in bytes.
// Zero means no limit.
func (rt *Runtime) SetMemoryLimit(bytes int32) {
	rt.Registry.RuntimeSetMemoryLimit(rt.ctx, rt.Pointer, bytes)
}

// RunGC triggers QuickJS garbage collection.
func (rt *Runtime) RunGC() {
	rt.Registry.RunGC(rt.ctx, rt.Pointer)
}

// IsMicrotaskPending reports whether there are pending promise jobs.
func (rt *Runtime) IsMicrotaskPending() bool {
	return rt.Registry.IsJobPending(rt.ctx, rt.Pointer) != 0
}

// ExecuteMicrotasks runs pending promise jobs (microtasks).
//
// maxJobs limits how many jobs to execute. Use -1 for no limit.
// Returns the number of jobs executed, or -1 on error.
//
// This is equivalent to ExecuteMicrotasks in the C# host.
func (rt *Runtime) ExecuteMicrotasks(maxJobs int32) int32 {
	return rt.Registry.ExecutePendingJob(rt.ctx, rt.Pointer, maxJobs, 0)
}

// dropRealm removes a realm from internal tracking (called by Realm.Close).
func (rt *Runtime) dropRealm(realm *Realm) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	delete(rt.realms, realm.Pointer)
}

// Close releases all resources associated with the Runtime.
//
// All Realms created from this Runtime are also closed.
// After Close, the Runtime cannot be used.
func (rt *Runtime) Close() error {
	if rt.disposed {
		return nil
	}
	rt.disposed = true

	// Close all realms first.
	rt.mu.Lock()
	for _, realm := range rt.realms {
		realm.dispose()
	}
	rt.realms = nil
	rt.mu.Unlock()

	// Free the QuickJS runtime.
	if rt.Pointer != 0 {
		rt.Callbacks.UnregisterRuntime(rt.Pointer)
		rt.Registry.FreeRuntime(rt.ctx, rt.Pointer)
		rt.Pointer = 0
	}

	// Close the wazero runtime (closes all modules).
	if rt.wazero != nil {
		return rt.wazero.Close(rt.ctx)
	}
	return nil
}
