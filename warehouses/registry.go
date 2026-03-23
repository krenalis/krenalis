// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package warehouses

import (
	"fmt"
	"reflect"
	"sync"
)

var registry = struct {
	sync.RWMutex
	platforms map[string]Platform
}{
	platforms: make(map[string]Platform),
}

// Platforms returns the warehouse platforms.
func Platforms() []Platform {
	registry.Lock()
	platforms := make([]Platform, 0, len(registry.platforms))
	for _, t := range registry.platforms {
		platforms = append(platforms, t)
	}
	registry.Unlock()
	return platforms
}

// Register makes a warehouse available with the provided platform name. If
// Register is called twice with the same platform or if new is nil, it panics.
func Register[T Warehouse](platform Platform, new NewFunc[T]) {
	if new == nil {
		panic("krenalis/warehouses: new function is nil for warehouse platform " + platform.Name)
	}
	platform.newFunc = reflect.ValueOf(new)
	platform.ct = reflect.TypeFor[T]()
	registry.Lock()
	defer registry.Unlock()
	if _, dup := registry.platforms[platform.Name]; dup {
		panic("krenalis/warehouses: Register called twice for type " + platform.Name)
	}
	registry.platforms[platform.Name] = platform
}

// Registered returns the registered warehouse for the given platform.
// It panics if the platform is not registered.
func Registered(platform string) Platform {
	registry.Lock()
	warehouse, ok := registry.platforms[platform]
	registry.Unlock()
	if !ok {
		panic(fmt.Errorf("krenalis/warehouses: unknown warehouse platform %q (forgotten import?)", platform))
	}
	return warehouse
}
