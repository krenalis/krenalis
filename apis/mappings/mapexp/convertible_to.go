//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mapexp

import "chichi/connector/types"

// convertMatrix is a matrix which holds information about valid conversions.
//
// NOTE: keep this in sync with the content of the file
// 'apis/mappings/mapexp/Conversions.md'.
var convertMatrix = [...]int32{
	//                ┌─── Boolean
	//                │ ┌────── Int
	//                │ │ ┌──── Uint
	//                │ │ │ ┌── Float
	//                │ │ │ │ ┌─ DateTime
	//                │ │ │ │ │   ┌─ Year
	//                │ │ │ │ │   │  ┌── JSON
	//                │ │ │ │ │   │  │   ┌── Array
	//                │ │ │ │ │   │  │   │
	/* Boolean  */ 0b_1_0_0_00_000_00_101_000,
	/* Int      */ 0b_0_1_1_11_000_10_101_000,
	/* Uint     */ 0b_0_1_1_11_000_10_101_000,
	/* Float    */ 0b_0_1_1_11_000_00_101_000,
	/* Decimal  */ 0b_0_1_1_11_000_00_101_000,
	/* DateTime */ 0b_0_0_0_00_111_00_101_000,
	/* Date     */ 0b_0_0_0_00_110_00_101_000,
	/* Time     */ 0b_0_0_0_00_001_00_101_000,
	/* Year     */ 0b_0_1_1_00_000_10_101_000,
	/* UUID     */ 0b_0_0_0_00_000_01_101_000,
	/* JSON     */ 0b_1_1_1_11_111_11_111_111,
	/* Inet     */ 0b_0_0_0_00_000_00_111_000,
	/* Text     */ 0b_1_1_1_11_111_11_111_000,
	/* Array    */ 0b_0_0_0_00_000_00_100_100,
	/* Object   */ 0b_0_0_0_00_000_00_100_010,
	/* Map      */ 0b_0_0_0_00_000_00_100_001,
}

// convertibleTo reports whether a value of type st can be converted to type dt.
func convertibleTo(st, dt types.Type) bool {
	spt := st.PhysicalType()
	dpt := dt.PhysicalType()
	mask := int32(1 << (types.PtMap - dpt))
	if convertMatrix[spt-1]&mask == 0 {
		if spt == types.PtBoolean && dpt == types.PtInt && dt.BitSize() == 8 { // Boolean is convertible to Int(8)
			return true
		}
		if spt == types.PtInt && dpt == types.PtBoolean && st.BitSize() == 8 { // Int(8) is convertible to Boolean.
			return true
		}
		return false
	}
	if spt == types.PtJSON {
		return true
	}
	switch dpt {
	case types.PtArray, types.PtMap:
		return convertibleTo(st.Elem(), dt.Elem())
	case types.PtObject:
		var hasSameNameProperty bool
		for _, sp := range st.Properties() {
			if dp, ok := dt.Property(sp.Name); ok {
				if !convertibleTo(sp.Type, dp.Type) {
					return false
				}
				hasSameNameProperty = true
			}
		}
		return hasSameNameProperty
	}
	return true
}
