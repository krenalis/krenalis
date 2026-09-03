import { useCallback, useMemo, useRef, useState } from 'react';
import { EditableSchema, getParentPropertyKey } from './SchemaEdit.helpers';

type PropertyReorderHistory = ReadonlyMap<string, number>;

interface UsePropertyReorderingOptions {
	editableSchema: EditableSchema | null | undefined;
	initialEditableSchema: EditableSchema | null | undefined;
	setEditableSchema: (schema: EditableSchema) => void;
}

interface PropertyReordering {
	reorderedPropertyKeys: ReadonlySet<string>;
	onSortRow: (overRowID: string, movedRowID: string) => void;
	resetMoveHistory: () => void;
}

// Bitmasks let us apply the tie-break rules without storing the full subsequence for every candidate.
interface LCSCandidate {
	key: string;
	currentPosition: number;
	moveBit: bigint;
	initialPositionBit: bigint;
}

interface LCSPath {
	length: number;
	stationaryMovesMask: bigint;
	initialPositionsMask: bigint;
	previousCandidateIndex: number;
}

const usePropertyReordering = ({
	editableSchema,
	initialEditableSchema,
	setEditableSchema,
}: UsePropertyReorderingOptions): PropertyReordering => {
	const [moveHistory, setMoveHistory] = useState<PropertyReorderHistory>(() => new Map());
	const nextMoveSequence = useRef(0);

	const reorderedPropertyKeys = useMemo(() => {
		return getReorderedPropertyKeys(editableSchema, initialEditableSchema, moveHistory);
	}, [editableSchema, initialEditableSchema, moveHistory]);

	const onSortRow = useCallback(
		(overRowID: string, movedRowID: string) => {
			const reorderedSchema = reorderEditableSchema(editableSchema, overRowID, movedRowID);
			if (reorderedSchema == null) {
				return;
			}

			nextMoveSequence.current++;
			const sequence = nextMoveSequence.current;
			setMoveHistory((current) => new Map(current).set(movedRowID, sequence));
			setEditableSchema(reorderedSchema);
		},
		[editableSchema, setEditableSchema],
	);

	const resetMoveHistory = useCallback(() => {
		nextMoveSequence.current = 0;
		setMoveHistory(new Map());
	}, []);

	return { reorderedPropertyKeys, onSortRow, resetMoveHistory };
};

// A reorder should be attributed to the properties the user moved, not to every
// property whose absolute position changed as a result. For example, moving A
// after D changes the positions of B, C, and D, but only A should be marked as
// modified.
//
// The comparison considers only pre-existing properties that are still present
// in the current schema. It runs independently for each group of direct siblings.
// This keeps added or removed properties out of the result and prevents moving
// an object subtree from affecting the order of the object's children.
//
// Move history records the user's intent. Siblings that have never been moved
// act as anchors when their relative order is still consistent. Within each
// interval between anchors, a deterministic longest common subsequence identifies
// the properties that can remain stationary; its complement is marked as
// reordered. Since keys are unique, the LCS can be calculated as an LIS of current
// positions.
//
// When several subsequences have the same length, ties are broken by leaving the
// most recently moved properties out of the stationary subsequence, so they are
// marked as reordered. If there is still a tie, the earliest possible positions
// from the initial order are preserved.
//
// If the relative order of the anchors is inconsistent with the initial order,
// an unconstrained deterministic LCS provides a stable fallback. Restoring the
// initial order always produces an empty result, regardless of the retained move
// history.
const getReorderedPropertyKeys = (
	currentSchema: EditableSchema | null | undefined,
	initialSchema: EditableSchema | null | undefined,
	moveHistory: PropertyReorderHistory,
): Set<string> => {
	if (currentSchema == null || initialSchema == null) {
		return new Set<string>();
	}

	const reorderedKeys = new Set<string>();
	const currentKeys = Object.keys(currentSchema);
	const initialKeys = Object.keys(initialSchema);
	const initialKeySet = new Set(initialKeys);
	// A newly added property can reuse the key of a removed property. The isEditable
	// flag distinguishes it from the original property.
	const survivingKeys = new Set(
		currentKeys.filter((key) => initialKeySet.has(key) && currentSchema[key].isEditable !== true),
	);
	const currentGroups = groupKeysByParent(currentKeys, survivingKeys);
	const initialGroups = groupKeysByParent(initialKeys, survivingKeys);

	for (const [parentKey, currentOrder] of currentGroups) {
		const initialOrder = initialGroups.get(parentKey) ?? [];
		if (areEqualSequences(initialOrder, currentOrder)) {
			continue;
		}

		const anchors = currentOrder.filter((key) => !moveHistory.has(key));
		const initialAnchors = initialOrder.filter((key) => !moveHistory.has(key));
		const hasConsistentAnchorOrder = areEqualSequences(initialAnchors, anchors);
		let stationaryKeys: Set<string>;
		if (!hasConsistentAnchorOrder) {
			// Use an unconstrained LCS when the move history cannot explain the current order.
			stationaryKeys = new Set(getCanonicalStationarySubsequence(initialOrder, currentOrder, moveHistory));
		} else {
			stationaryKeys = new Set(anchors);
			const anchorSet = new Set(anchors);
			const initialIntervals = splitAroundAnchors(initialOrder, anchorSet);
			const currentIntervals = splitAroundAnchors(currentOrder, anchorSet);
			for (let index = 0; index < initialIntervals.length; index++) {
				const stationaryInterval = getCanonicalStationarySubsequence(
					initialIntervals[index],
					currentIntervals[index],
					moveHistory,
				);
				for (const key of stationaryInterval) {
					stationaryKeys.add(key);
				}
			}
		}

		for (const key of currentOrder) {
			if (!stationaryKeys.has(key)) {
				reorderedKeys.add(key);
			}
		}
	}

	return reorderedKeys;
};

const areEqualSequences = (first: string[], second: string[]): boolean => {
	return first.length === second.length && first.every((key, index) => key === second[index]);
};

const getBit = (index: number): bigint => {
	return BigInt(1) << BigInt(index);
};

const getCanonicalStationarySubsequence = (
	initialOrder: string[],
	currentOrder: string[],
	moveHistory: PropertyReorderHistory,
): string[] => {
	// With unique keys, finding an LIS of the current positions is equivalent to finding an LCS.
	// Prefer longer paths, then paths that leave newer moves out, and finally earlier initial positions.
	const currentPositions = new Map(currentOrder.map((key, index) => [key, index]));
	const commonKeys = initialOrder.filter((key) => currentPositions.has(key));
	const movedKeys = commonKeys
		.filter((key) => moveHistory.has(key))
		.sort((first, second) => moveHistory.get(first)! - moveHistory.get(second)! || first.localeCompare(second));
	// Assign higher bits to newer moves so the smaller mask excludes the newest move at the first difference.
	const moveBits = new Map(movedKeys.map((key, index) => [key, getBit(index)]));
	// Assign higher bits to earlier initial positions so the larger mask prefers the lexicographically earliest sequence.
	const candidates: LCSCandidate[] = [];
	for (let initialIndex = 0; initialIndex < initialOrder.length; initialIndex++) {
		const key = initialOrder[initialIndex];
		const currentPosition = currentPositions.get(key);
		if (currentPosition == null) {
			continue;
		}
		candidates.push({
			key,
			currentPosition,
			moveBit: moveBits.get(key) ?? BigInt(0),
			initialPositionBit: getBit(initialOrder.length - initialIndex - 1),
		});
	}

	const bestPathsByEnd: LCSPath[] = [];
	let bestPathEndIndex = -1;
	for (let index = 0; index < candidates.length; index++) {
		const candidate = candidates[index];
		let bestPathEndingHere: LCSPath = {
			length: 1,
			stationaryMovesMask: candidate.moveBit,
			initialPositionsMask: candidate.initialPositionBit,
			previousCandidateIndex: -1,
		};
		for (let previousCandidateIndex = 0; previousCandidateIndex < index; previousCandidateIndex++) {
			if (candidates[previousCandidateIndex].currentPosition >= candidate.currentPosition) {
				continue;
			}
			const previousPath = bestPathsByEnd[previousCandidateIndex];
			const extendedPath: LCSPath = {
				length: previousPath.length + 1,
				stationaryMovesMask: previousPath.stationaryMovesMask | candidate.moveBit,
				initialPositionsMask: previousPath.initialPositionsMask | candidate.initialPositionBit,
				previousCandidateIndex,
			};
			if (isPreferredLCSPath(extendedPath, bestPathEndingHere)) {
				bestPathEndingHere = extendedPath;
			}
		}
		bestPathsByEnd.push(bestPathEndingHere);
		if (bestPathEndIndex === -1 || isPreferredLCSPath(bestPathEndingHere, bestPathsByEnd[bestPathEndIndex])) {
			bestPathEndIndex = index;
		}
	}

	const stationaryKeys: string[] = [];
	let candidateIndex = bestPathEndIndex;
	while (candidateIndex !== -1) {
		stationaryKeys.push(candidates[candidateIndex].key);
		candidateIndex = bestPathsByEnd[candidateIndex].previousCandidateIndex;
	}
	stationaryKeys.reverse();
	return stationaryKeys;
};

const groupKeysByParent = (keys: string[], includedKeys: ReadonlySet<string>): Map<string, string[]> => {
	const groups = new Map<string, string[]>();
	for (const key of keys) {
		if (!includedKeys.has(key)) {
			continue;
		}
		const parentKey = getParentPropertyKey(key);
		const group = groups.get(parentKey) ?? [];
		group.push(key);
		groups.set(parentKey, group);
	}
	return groups;
};

const isPreferredLCSPath = (candidate: LCSPath, currentBest: LCSPath): boolean => {
	if (candidate.length !== currentBest.length) {
		return candidate.length > currentBest.length;
	}
	if (candidate.stationaryMovesMask !== currentBest.stationaryMovesMask) {
		return candidate.stationaryMovesMask < currentBest.stationaryMovesMask;
	}
	// With unique keys, equal initial-position masks identify the same subsequence, so no additional tie-break is needed.
	return candidate.initialPositionsMask > currentBest.initialPositionsMask;
};

const reorderEditableSchema = (
	schema: EditableSchema | null | undefined,
	overRowID: string,
	movedRowID: string,
): EditableSchema | null => {
	if (
		schema == null ||
		overRowID === movedRowID ||
		schema[overRowID] == null ||
		schema[movedRowID] == null ||
		getParentPropertyKey(overRowID) !== getParentPropertyKey(movedRowID)
	) {
		return null;
	}

	const keys = Object.keys(schema);
	const overIndex = keys.indexOf(overRowID);
	const movedIndex = keys.indexOf(movedRowID);
	const movedKeys = keys.filter((key) => key === movedRowID || key.startsWith(`${movedRowID}.`));
	const reorderedKeys = keys.filter((key) => !movedKeys.includes(key));
	let insertIndex = reorderedKeys.indexOf(overRowID);
	if (overIndex > movedIndex) {
		insertIndex++;
	}
	reorderedKeys.splice(insertIndex, 0, ...movedKeys);
	if (areEqualSequences(keys, reorderedKeys)) {
		return null;
	}

	const reorderedSchema: EditableSchema = {};
	for (const key of reorderedKeys) {
		reorderedSchema[key] = schema[key];
	}
	return reorderedSchema;
};

const splitAroundAnchors = (keys: string[], anchors: ReadonlySet<string>): string[][] => {
	const intervals: string[][] = [[]];
	for (const key of keys) {
		if (anchors.has(key)) {
			intervals.push([]);
		} else {
			intervals[intervals.length - 1].push(key);
		}
	}
	return intervals;
};

export { getReorderedPropertyKeys, usePropertyReordering };
