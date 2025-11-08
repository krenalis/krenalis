// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package warehouses

import (
	"fmt"
	"reflect"
	"sync"
)

var registry = struct {
	sync.RWMutex
	warehouses map[string]Driver
}{
	warehouses: make(map[string]Driver),
}

// Drivers returns the warehouse drivers.
func Drivers() []Driver {
	registry.Lock()
	drivers := make([]Driver, 0, len(registry.warehouses))
	for _, t := range registry.warehouses {
		drivers = append(drivers, t)
	}
	registry.Unlock()
	return drivers
}

// Register makes a warehouse driver available by the provided name. If Register
// is called twice with the same name or if new is nil, it panics.
func Register[T Warehouse](typ Driver, new NewFunc[T]) {
	if new == nil {
		panic("meergo/warehouses: new function is nil for warehouse driver " + typ.Name)
	}
	typ.newFunc = reflect.ValueOf(new)
	typ.ct = reflect.TypeOf((*T)(nil)).Elem()
	registry.Lock()
	defer registry.Unlock()
	if _, dup := registry.warehouses[typ.Name]; dup {
		panic("meergo/warehouses: Register called twice for type " + typ.Name)
	}
	registry.warehouses[typ.Name] = typ
}

// Registered returns the warehouse driver registered with the given name.
// If a warehouse driver with this name is not registered, it panics.
func Registered(name string) Driver {
	registry.Lock()
	warehouse, ok := registry.warehouses[name]
	registry.Unlock()
	if !ok {
		panic(fmt.Errorf("meergo/warehouses: unknown warehouse driver %q (forgotten import?)", name))
	}
	return warehouse
}
