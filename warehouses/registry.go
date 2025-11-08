// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package warehouses

import (
	"fmt"
	"reflect"
	"sync"
)

var (
	registryMu sync.RWMutex
	registry   = struct {
		warehouses map[string]WarehouseDriver
	}{
		warehouses: make(map[string]WarehouseDriver),
	}
)

// RegisterWarehouseDriver makes a warehouse driver available by the provided
// name. If RegisterWarehouseDriver is called twice with the same name or if new
// is nil, it panics.
func RegisterWarehouseDriver[T Warehouse](typ WarehouseDriver, new WarehouseDriverNewFunc[T]) {
	if new == nil {
		panic("meergo: new function is nil for warehouse driver " + typ.Name)
	}
	typ.newFunc = reflect.ValueOf(new)
	typ.ct = reflect.TypeOf((*T)(nil)).Elem()
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry.warehouses[typ.Name]; dup {
		panic("meergo: RegisterWarehouseDriver called twice for type " + typ.Name)
	}
	registry.warehouses[typ.Name] = typ
}

// RegisteredWarehouseDriver returns the warehouse driver registered with the
// given name. If a warehouse driver with this name is not registered, it
// panics.
func RegisteredWarehouseDriver(name string) WarehouseDriver {
	registryMu.Lock()
	warehouse, ok := registry.warehouses[name]
	registryMu.Unlock()
	if !ok {
		panic(fmt.Errorf("meergo: unknown warehouse driver %q (forgotten import?)", name))
	}
	return warehouse
}

// WarehouseDrivers returns the warehouse drivers.
func WarehouseDrivers() []WarehouseDriver {
	registryMu.Lock()
	drivers := make([]WarehouseDriver, 0, len(registry.warehouses))
	for _, t := range registry.warehouses {
		drivers = append(drivers, t)
	}
	registryMu.Unlock()
	return drivers
}
