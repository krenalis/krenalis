//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package micropython

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"

	"chichi/apis/transformations/micropython/hostmodulesbuilder"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/emscripten"
)

// VM is a virtual machine which can run Python code through MicroPython.
type VM struct {
	ctx context.Context
	rt  wazero.Runtime
	mod api.Module
}

//go:embed "wasm/micropython.wasm"
var microPythonSource []byte

// NewVM returns a VM which can run Python code through MicroPython.
// The caller of NewVM should then close the VM calling the 'Close' method.
func NewVM(ctx context.Context, stackSize uint64, stdout io.Writer, enableDecimal bool) (*VM, error) {

	r := wazero.NewRuntime(ctx)
	h := hostmodulesbuilder.New(ctx, stdout)

	// Instantiate the host modules.
	for modName, modFuncs := range h.BuildModules() {
		mod := r.NewHostModuleBuilder(modName)
		for name, f := range modFuncs {
			mod.NewFunctionBuilder().WithFunc(f).Export(name)
		}
		if modName == "env" {
			emscripten.NewFunctionExporter().ExportFunctions(mod)
		}
		_, err := mod.Instantiate(ctx, r)
		if err != nil {
			return nil, fmt.Errorf("cannot instantiate module %q: %s", modName, err)
		}
	}

	// Instantiate "micropython.wasm".
	mod, err := r.InstantiateModuleFromBinary(ctx, microPythonSource)
	if err != nil {
		return nil, fmt.Errorf("cannot instantiate module from binary: %s", err)
	}

	// Initialize the MicroPython stack.
	_, err = mod.ExportedFunction("mp_js_init").Call(ctx, stackSize)
	if err != nil {
		return nil, fmt.Errorf("cannot init stack: %s", err)
	}

	// Make the module accessible to the host functions.
	h.SetWASMModule(mod)
	return &VM{ctx: ctx, rt: r, mod: mod}, nil
}

// Close closes the VM.
func (mv *VM) Close() error {
	return mv.rt.Close(mv.ctx)
}

// InitREPL initializes the REPL. This should be called before calling any
// method which operate on the REPL. This method should be called once.
func (vm *VM) InitREPL() error {
	_, err := vm.mod.ExportedFunction("mp_js_init_repl").Call(vm.ctx)
	return err
}

// REPLSendChar sends the given rune to the REPL. Before calling this method,
// 'InitREPL' should be called (see the documentation of InitREPL).
func (vm *VM) REPLSendChar(r rune) error {
	_, err := vm.mod.ExportedFunction("mp_js_process_char").Call(vm.ctx, uint64(r))
	return err
}

// REPLSendChar sends 'enter' to the REPL. Before calling this method,
// 'InitREPL' should be called (see the documentation of InitREPL).
func (vm *VM) REPLSendEnter() error { return vm.REPLSendChar(13) }

// RunSourceCode runs the given source code.
func (vm *VM) RunSourceCode(src []byte) error {
	codeOffset := uint32(0)
	ok := vm.mod.Memory().Write(vm.ctx, codeOffset, src)
	if !ok {
		return errors.New("out of range")
	}
	_, err := vm.mod.ExportedFunction("mp_js_do_str").Call(vm.ctx, uint64(codeOffset))
	if err != nil {
		return fmt.Errorf("cannot run code: %s", err)
	}
	return nil
}
