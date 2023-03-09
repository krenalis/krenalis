//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package hostmodulesbuilder

import (
	"io"
	"log"

	"github.com/tetratelabs/wazero/api"
)

// HostModulesBuilder is a builder for the host modules.
type HostModulesBuilder struct {
	mod    api.Module
	stdout io.Writer
}

// New returns a new instance of HostModulesBuilder. If a function returned by
// this builder writes something, it writes it to stdout.
func New(stdout io.Writer) *HostModulesBuilder {
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
			"mp_js_hook":                func() { panic("'mp_js_hook' not implemented") },
			"mp_js_ticks_ms":            func() int32 { panic("'mp_js_ticks_ms' not implemented") },
			"mp_js_write":               h.mpJsWrite,
		},
	}
}

func (h *HostModulesBuilder) emscripten_memcpy_big(dest, src, num int32) {
	mem := h.mod.Memory()
	data, ok := mem.Read(uint32(src), uint32(num))
	if !ok {
		panic("cannot read memory")
	}
	ok = mem.Write(uint32(dest), data)
	if !ok {
		panic("cannot write memory")
	}
}

func (h *HostModulesBuilder) mpJsWrite(ptr, length int32) {
	data, ok := h.mod.Memory().Read(uint32(ptr), uint32(length))
	if !ok {
		log.Panicf("cannot read from %d %d", ptr, length)
	}
	_, err := h.stdout.Write(data)
	if err != nil {
		log.Panic(err)
	}
}
