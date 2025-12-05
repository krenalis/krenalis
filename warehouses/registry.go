// Copyright 2025 Open2b. All rights reserved.
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

// Register makes a warehouse platform available by the provided name.
// If Register is called twice with the same name or if new is nil, it panics.
func Register[T Warehouse](platform Platform, new NewFunc[T]) {
	if new == nil {
		panic("meergo/warehouses: new function is nil for warehouse platform " + platform.Name)
	}
	platform.newFunc = reflect.ValueOf(new)
	platform.ct = reflect.TypeOf((*T)(nil)).Elem()
	registry.Lock()
	defer registry.Unlock()
	if _, dup := registry.platforms[platform.Name]; dup {
		panic("meergo/warehouses: Register called twice for type " + platform.Name)
	}
	registry.platforms[platform.Name] = platform
}

// Registered returns the warehouse platform registered with the given name.
// If a warehouse platform with this name is not registered, it panics.
func Registered(name string) Platform {
	registry.Lock()
	warehouse, ok := registry.platforms[name]
	registry.Unlock()
	if !ok {
		panic(fmt.Errorf("meergo/warehouses: unknown warehouse platform %q (forgotten import?)", name))
	}
	return warehouse
}
