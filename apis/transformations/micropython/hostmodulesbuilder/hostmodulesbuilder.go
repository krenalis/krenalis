//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package hostmodulesbuilder

import (
	"context"
	"io"
	"log"

	"github.com/tetratelabs/wazero/api"
)

// HostModulesBuilder is a builder for the host modules.
type HostModulesBuilder struct {
	ctx    context.Context
	mod    api.Module
	stdout io.Writer
}

// New returns a new instance of HostModulesBuilder. If a function returned by
// this builder writes something, it writes it to stdout.
func New(ctx context.Context, stdout io.Writer) *HostModulesBuilder {
	return &HostModulesBuilder{stdout: stdout}
}

// SetWASMModule sets the module relative to the ".wasm" file. Should be called
// before running the WASM module which uses the functions built with this
// HostModulesBuilder.
func (h *HostModulesBuilder) SetWASMModule(mod api.Module) { h.mod = mod }

// BuildModules builds the modules and returns them. The key of the map is the
// module name, and the value associated to every key is a map from the function
// name to the function itself.
func (h *HostModulesBuilder) BuildModules() map[string]map[string]any {
	return map[string]map[string]any{
		"wasi_snapshot_preview1": {
			"fd_close": func(int32) int32 { panic("not imp") },
			"fd_read":  func(int32, int32, int32, int32) int32 { panic("not imp") },
			"fd_seek":  func(int32, int32, int32, int32, int32) int32 { panic("not imp") },
			"fd_sync":  func(int32) int32 { panic("not imp") },
			"fd_write": func(int32, int32, int32, int32) int32 { panic("not imp") },
		},
		"env": {
			"__syscall_chdir":           func(int32) int32 { panic("'__syscall_chdir' not implemented") },
			"__syscall_fstat64":         func(int32, int32) int32 { panic("'__syscall_fstat64' not implemented") },
			"__syscall_getcwd":          func(int32, int32) int32 { panic("'__syscall_getcwd' not implemented") },
			"__syscall_getdents64":      func(int32, int32, int32) int32 { panic("'__syscall_getdents64' not implemented") },
			"__syscall_lstat64":         func(int32, int32) int32 { panic("'__syscall_lstat64' not implemented") },
			"__syscall_mkdirat":         func(int32, int32, int32) int32 { panic("'__syscall_mkdirat' not implemented") },
			"__syscall_newfstatat":      func(int32, int32, int32, int32) int32 { panic("'__syscall_newfstatat' not implemented") },
			"__syscall_openat":          func(int32, int32, int32, int32) int32 { panic("'__syscall_openat' not implemented") },
			"__syscall_poll":            func(int32, int32, int32) int32 { panic("'__syscall_poll' not implemented") },
			"__syscall_renameat":        func(int32, int32, int32, int32) int32 { panic("'__syscall_renameat' not implemented") },
			"__syscall_rmdir":           func(int32) int32 { panic("'__syscall_rmdir' not implemented") },
			"__syscall_stat64":          func(int32, int32) int32 { return 0 },
			"__syscall_statfs64":        func(int32, int32, int32) int32 { panic("'__syscall_statfs64' not implemented") },
			"__syscall_unlinkat":        func(int32, int32, int32) int32 { panic("'__syscall_unlinkat' not implemented") },
			"_emscripten_throw_longjmp": func() { panic("an exception has been raised by MicroPython") },
			"emscripten_memcpy_big":     h.emscripten_memcpy_big,
			"emscripten_resize_heap":    func(int32) int32 { panic("'emscripten_resize_heap' not implemented") },
			"emscripten_scan_registers": func(int32) { panic("'emscripten_scan_registers' not implemented") },
			"invoke_i":                  h.invoke_i,
			"invoke_ii":                 h.invoke_ii,
			"invoke_iii":                h.invoke_iii,
			"invoke_iiii":               h.invoke_iiii,
			"invoke_iiiii":              h.invoke_iiiii,
			"invoke_v":                  h.invoke_v,
			"invoke_vi":                 h.invoke_vi,
			"invoke_vii":                h.invoke_vii,
			"invoke_viii":               h.invoke_viii,
			"invoke_viiii":              h.invoke_viiii,
			"mp_js_hook":                func() { panic("'mp_js_hook' not implemented") },
			"mp_js_ticks_ms":            func() int32 { panic("'mp_js_ticks_ms' not implemented") },
			"mp_js_write":               h.mpJsWrite,
		},
	}
}

func (h *HostModulesBuilder) emscripten_memcpy_big(dest, src, num int32) {
	mem := h.mod.Memory()
	data, ok := mem.Read(h.ctx, uint32(src), uint32(num))
	if !ok {
		panic("cannot read memory")
	}
	ok = mem.Write(h.ctx, uint32(dest), data)
	if !ok {
		panic("cannot write memory")
	}
}

func (h *HostModulesBuilder) invoke_i(a int32) int32 {
	out, err := h.mod.ExportedFunction("dynCall_i").Call(h.ctx, uint64(a))
	if err != nil {
		log.Panic(err)
	}
	return int32(out[0])
}

func (h *HostModulesBuilder) invoke_ii(a, b int32) int32 {
	out, err := h.mod.ExportedFunction("dynCall_ii").Call(h.ctx, uint64(a), uint64(b))
	if err != nil {
		log.Panic(err)
	}
	return int32(out[0])
}

func (h *HostModulesBuilder) invoke_iii(a, b, c int32) int32 {
	out, err := h.mod.ExportedFunction("dynCall_iii").Call(h.ctx, uint64(a), uint64(b), uint64(c))
	if err != nil {
		log.Panic(err)
	}
	return int32(out[0])
}

func (h *HostModulesBuilder) invoke_iiii(a, b, c, d int32) int32 {
	out, err := h.mod.ExportedFunction("dynCall_iiii").Call(h.ctx, uint64(a), uint64(b), uint64(c), uint64(d))
	if err != nil {
		log.Panic(err)
	}
	return int32(out[0])
}

func (h *HostModulesBuilder) invoke_iiiii(a, b, c, d, e int32) int32 {
	out, err := h.mod.ExportedFunction("dynCall_iiiii").Call(h.ctx, uint64(a), uint64(b), uint64(c), uint64(d), uint64(e))
	if err != nil {
		log.Panic(err)
	}
	return int32(out[0])
}

func (h *HostModulesBuilder) invoke_v(a int32) {
	_, err := h.mod.ExportedFunction("dynCall_v").Call(h.ctx, uint64(a))
	if err != nil {
		log.Panic(err)
	}
}

func (h *HostModulesBuilder) invoke_vi(a, b int32) {
	_, err := h.mod.ExportedFunction("dynCall_vi").Call(h.ctx, uint64(a), uint64(b))
	if err != nil {
		log.Panic(err)
	}
}

func (h *HostModulesBuilder) invoke_vii(a, b, c int32) {
	_, err := h.mod.ExportedFunction("dynCall_vii").Call(h.ctx, uint64(a), uint64(b), uint64(c))
	if err != nil {
		log.Panic(err)
	}
}

func (h *HostModulesBuilder) invoke_viii(a, b, c, d int32) {
	_, err := h.mod.ExportedFunction("dynCall_viii").Call(h.ctx, uint64(a), uint64(b), uint64(c), uint64(d))
	if err != nil {
		log.Panic(err)
	}
}

func (h *HostModulesBuilder) invoke_viiii(a, b, c, d, e int32) {
	_, err := h.mod.ExportedFunction("dynCall_viiii").Call(h.ctx, uint64(a), uint64(b), uint64(c), uint64(d), uint64(e))
	if err != nil {
		log.Panic(err)
	}
}

func (h *HostModulesBuilder) mpJsWrite(ptr, length int32) {
	data, ok := h.mod.Memory().Read(h.ctx, uint32(ptr), uint32(length))
	if !ok {
		log.Panicf("cannot read from %d %d", ptr, length)
	}
	_, err := h.stdout.Write(data)
	if err != nil {
		log.Panic(err)
	}
}
