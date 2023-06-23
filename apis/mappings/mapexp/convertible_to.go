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
// 'apis/mappings/Conversions.md'.
var convertMatrix = [...]int32{
	//                ┌─── Boolean
	//                │ ┌────── Int
	//                │ │     ┌──── UInt
	//                │ │     │     ┌── Float
	//                │ │     │     │   ┌─ DateTime
	//                │ │     │     │   │   ┌─ Year
	//                │ │     │     │   │   │  ┌── JSON
	//                │ │     │     │   │   │  │   ┌── Array
	//                │ │     │     │   │   │  │   │
	/* Boolean  */ 0b_1_01000_01000_000_000_00_101_000,
	/* Int      */ 0b_0_11111_11111_111_000_10_101_000,
	/* Int8     */ 0b_1_11111_11111_111_000_10_101_000,
	/* Int16    */ 0b_0_11111_11111_111_000_10_101_000,
	/* Int24    */ 0b_0_11111_11111_111_000_10_101_000,
	/* Int64    */ 0b_0_11111_11111_111_000_10_101_000,
	/* UInt     */ 0b_0_11111_11111_111_000_10_101_000,
	/* UInt8    */ 0b_1_11111_11111_111_000_10_101_000,
	/* UInt16   */ 0b_0_11111_11111_111_000_10_101_000,
	/* UInt24   */ 0b_0_11111_11111_111_000_10_101_000,
	/* UInt64   */ 0b_0_11111_11111_111_000_00_101_000,
	/* Float    */ 0b_0_11111_11111_111_000_00_101_000,
	/* Float32  */ 0b_0_11111_11111_111_000_00_101_000,
	/* Decimal  */ 0b_0_11111_11111_111_000_00_101_000,
	/* DateTime */ 0b_0_00000_00000_000_111_00_101_000,
	/* Date     */ 0b_0_00000_00000_000_110_00_101_000,
	/* Time     */ 0b_0_00000_00000_000_001_00_101_000,
	/* Year     */ 0b_0_11111_11111_000_000_10_101_000,
	/* UUID     */ 0b_0_00000_00000_000_000_01_101_000,
	/* JSON     */ 0b_1_11111_11111_111_111_11_111_111,
	/* Inet     */ 0b_0_00000_00000_000_000_00_111_000,
	/* Text     */ 0b_1_11111_11111_111_111_11_111_000,
	/* Array    */ 0b_0_00000_00000_000_000_00_100_100,
	/* Object   */ 0b_0_00000_00000_000_000_00_100_010,
	/* Map      */ 0b_0_00000_00000_000_000_00_100_001,
}

// convertibleTo reports whether the physical type "from" can be converted to
// the physical type "to".
func convertibleTo(from, to types.PhysicalType) bool {
	mask := int32(1 << (types.PtMap - to))
	return convertMatrix[from-1]&mask > 0
}
