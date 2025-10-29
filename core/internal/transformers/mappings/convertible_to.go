// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package mappings

import (
	"github.com/meergo/meergo/core/types"
)

// convertMatrix is a matrix which holds information about valid conversions.
//
// NOTE: keep this in sync with the content of the file
// 'core/internal/transformers/mappings/Conversions.md'.
var convertMatrix = [...]int32{
	//                ┌─── text
	//                │ ┌─── boolean
	//                │ │ ┌────── int
	//                │ │ │ ┌──── uint
	//                │ │ │ │ ┌── float
	//                │ │ │ │ │  ┌─ datetime
	//                │ │ │ │ │  │   ┌─ year
	//                │ │ │ │ │  │   │  ┌── json
	//                │ │ │ │ │  │   │  │  ┌── array
	//                │ │ │ │ │  │   │  │  │
	/* text     */ 0b_1_1_1_1_11_111_11_11_000,
	/* boolean  */ 0b_1_1_0_0_00_000_00_10_000,
	/* int      */ 0b_1_0_1_1_11_000_10_10_000,
	/* uint     */ 0b_1_0_1_1_11_000_10_10_000,
	/* float    */ 0b_1_0_1_1_11_000_00_10_000,
	/* decimal  */ 0b_1_0_1_1_11_000_00_10_000,
	/* datetime */ 0b_1_0_0_0_00_111_00_10_000,
	/* date     */ 0b_1_0_0_0_00_110_00_10_000,
	/* time     */ 0b_1_0_0_0_00_001_00_10_000,
	/* year     */ 0b_1_0_1_1_00_000_10_10_000,
	/* uuid     */ 0b_1_0_0_0_00_000_01_10_000,
	/* json     */ 0b_1_1_1_1_11_111_11_11_111,
	/* inet     */ 0b_1_0_0_0_00_000_00_11_000,
	/* array    */ 0b_0_0_0_0_00_000_00_10_100,
	/* object   */ 0b_0_0_0_0_00_000_00_10_011,
	/* map      */ 0b_0_0_0_0_00_000_00_10_011,
}

// convertibleTo reports whether a value of type st can be converted to type dt.
func convertibleTo(st, dt types.Type) bool {
	sk := st.Kind()
	dk := dt.Kind()
	mask := int32(1 << (types.MapKind - dk))
	if convertMatrix[sk-1]&mask == 0 {
		if sk == types.BooleanKind && dk == types.IntKind && dt.BitSize() == 8 { // boolean is convertible to int(8)
			return true
		}
		if sk == types.IntKind && dk == types.BooleanKind && st.BitSize() == 8 { // int(8) is convertible to boolean.
			return true
		}
		return false
	}
	if sk == types.JSONKind {
		return true
	}
	switch dk {
	case types.ArrayKind:
		if sk == types.ArrayKind {
			return convertibleTo(st.Elem(), dt.Elem())
		}
	case types.MapKind:
		switch sk {
		case types.MapKind:
			return convertibleTo(st.Elem(), dt.Elem())
		case types.ObjectKind:
			mapElemType := dt.Elem()
			for _, sp := range st.Properties().All() {
				if !convertibleTo(sp.Type, mapElemType) {
					return false
				}
			}
			return true
		}
	case types.ObjectKind:
		dProperties := dt.Properties()
		switch sk {
		case types.ObjectKind:
			sProperties := st.Properties()
			var hasSameNameProperty bool
			for _, p := range sProperties.All() {
				if dp, ok := dProperties.ByName(p.Name); ok {
					if !convertibleTo(p.Type, dp.Type) {
						return false
					}
					hasSameNameProperty = true
				}
			}
			return hasSameNameProperty
		case types.MapKind:
			mapElemType := st.Elem()
			for _, p := range dProperties.All() {
				if !convertibleTo(mapElemType, p.Type) {
					return false
				}
			}
			return true
		}
	}
	return true
}
