package hako_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/aspect-build/aspect-cli/hako/hako"
)

func TestHelloWorld(t *testing.T) {
	// Load the WASM binary
	wasmBytes, err := os.ReadFile("../../../engine/hako.wasm")
	if err != nil {
		t.Fatalf("failed to read wasm: %v", err)
	}

	ctx := context.Background()

	// Create runtime
	rt, err := hako.New(ctx, wasmBytes, nil)
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer rt.Close()

	// Create realm
	realm, err := rt.CreateRealm()
	if err != nil {
		t.Fatalf("failed to create realm: %v", err)
	}
	defer realm.Close()

	// Eval hello world
	result, err := realm.EvalCode(`"Hello, World!"`)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}
	defer result.Free()

	got := result.String()
	want := "Hello, World!"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	fmt.Println("Result:", got)
}

func TestArithmetic(t *testing.T) {
	wasmBytes, err := os.ReadFile("../../../engine/hako.wasm")
	if err != nil {
		t.Fatalf("failed to read wasm: %v", err)
	}

	ctx := context.Background()

	rt, err := hako.New(ctx, wasmBytes, nil)
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer rt.Close()

	realm, err := rt.CreateRealm()
	if err != nil {
		t.Fatalf("failed to create realm: %v", err)
	}
	defer realm.Close()

	result, err := realm.EvalCode(`2 + 2`)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}
	defer result.Free()

	got := result.AsNumber()
	want := 4.0
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	fmt.Println("2 + 2 =", got)
}
