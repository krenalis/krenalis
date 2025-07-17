import React, { useState } from 'react';
import './ConnectorKeyvalue.css';
import ConnectorField from '../ConnectorField';
import { KeyContext } from '../../../../context/KeyContext';
import { ValueContext } from '../../../../context/ValueContext';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import ConnectorFieldInterface from '../../../../lib/api/types/ui';
import {
	DndContext,
	closestCenter,
	KeyboardSensor,
	PointerSensor,
	useSensor,
	useSensors,
	DragOverlay,
} from '@dnd-kit/core';
import { SortableContext, sortableKeyboardCoordinates, verticalListSortingStrategy } from '@dnd-kit/sortable';
import { restrictToVerticalAxis, restrictToParentElement } from '@dnd-kit/modifiers';
import { DraggableWrapper } from '../../Grid/DraggableWrapper/DraggableWrapper';
import { OverlayRow } from '../../OverlayRow/OverlayRow';

interface KeyValueType {
	key: string;
	value: any;
}

type KeyValueValue = '' | KeyValueType[];

interface KeyValueRow {
	id: number;
	key: string;
	value: any;
}

interface ConnectorKeyValueProps {
	name: string;
	label: string;
	keyComponent: ConnectorFieldInterface;
	keyLabel: string;
	valueComponent: ConnectorFieldInterface;
	valueLabel: string;
	error: string;
	val: KeyValueValue;
	onChange: (...args: any) => void;
}

const ConnectorKeyValue = ({
	name,
	label,
	keyComponent,
	keyLabel,
	valueComponent,
	valueLabel,
	error,
	val,
	onChange,
}: ConnectorKeyValueProps) => {
	const [activeRow, setActiveRow] = useState(null);
	const [rows, setRows] = useState<KeyValueRow[]>(transformRows(val));

	const sensors = useSensors(
		useSensor(PointerSensor),
		useSensor(KeyboardSensor, {
			coordinateGetter: sortableKeyboardCoordinates,
		}),
	);

	const onAddRowClick = () => {
		const cloned = structuredClone(rows);
		let maxID = 0;
		for (const r of cloned) {
			if (r.id > maxID) {
				maxID = r.id;
			}
		}
		cloned.push({ id: maxID + 1, key: '', value: '' });
		setRows(cloned);
		onChange(name, normalizeRows(cloned));
	};

	const onRemoveRowClick = (id: number) => {
		const cloned = structuredClone(rows);
		const filtered = cloned.filter((r) => r.id !== id);
		setRows(filtered);
		onChange(name, normalizeRows(filtered));
	};

	const onKeyChange = (_, key, e) => {
		const id = Number(e.currentTarget.closest('.connector-keyvalue__row').dataset.id);
		const updated = rows.map((r) => {
			if (r.id === id) {
				return { ...r, key: key };
			}
			return r;
		});
		setRows(updated);
		onChange(name, normalizeRows(updated));
	};

	const onValueChange = (_, value, e) => {
		const id = Number(e.currentTarget.closest('.connector-keyvalue__row').dataset.id);
		const updated = rows.map((r) => {
			if (r.id === id) {
				return { ...r, value: value };
			}
			return r;
		});
		setRows(updated);
		onChange(name, normalizeRows(updated));
	};

	const onSortRow = (overRowID: number, movedRowID: number) => {
		const rws = [...rows];
		const overPropertyIndex = rws.findIndex((r) => r.id === overRowID);
		const movedPropertyIndex = rws.findIndex((r) => r.id === movedRowID);
		const isAfter = overPropertyIndex > movedPropertyIndex;
		const rowToMove = rws[movedPropertyIndex];
		const result = rws.filter((r) => r.id !== rowToMove.id);
		let insertIndex = result.findIndex((r) => r.id === overRowID);
		if (isAfter) {
			insertIndex++;
		}
		result.splice(insertIndex, 0, rowToMove);
		setRows(result);
		onChange(name, normalizeRows(result));
	};

	function onDragEnd(e) {
		const { over, active } = e;
		if (over.id !== active.id) {
			onSortRow(over.id, active.id);
		}
		setActiveRow(null);
	}

	function onDragStart(e) {
		const { active } = e;
		setActiveRow(active.id);
	}

	const sortableRowComponents = rows.map((r, index) => {
		return {
			id: r.id,
			row: (
				<div className='connector-keyvalue__row' data-id={r.id} key={r.id}>
					<KeyContext.Provider value={{ value: r.key, onChange: onKeyChange }}>
						<div className='connector-keyvalue__cell'>
							<ConnectorField field={keyComponent} />
						</div>
					</KeyContext.Provider>
					<ValueContext.Provider value={{ value: r.value, onChange: onValueChange }}>
						<div className='connector-keyvalue__cell'>
							<ConnectorField field={valueComponent} />
						</div>
					</ValueContext.Provider>
					{index !== 0 && (
						<SlIcon
							className='connector-keyvalue__remove-row'
							name='trash3'
							onClick={() => onRemoveRowClick(r.id)}
						/>
					)}
				</div>
			),
		};
	});

	return (
		<div className='connector-keyvalue'>
			<div className='connector-keyvalue__label'>{label}</div>
			<div className='connector-keyvalue__grid'>
				<div className='connector-keyvalue__row'>
					<div className='connector-keyvalue__key-label'>{keyLabel}</div>
					<div className='connector-keyvalue__value-label'>{valueLabel}</div>
				</div>
				{sortableRowComponents.length > 1 ? (
					<DndContext
						sensors={sensors}
						collisionDetection={closestCenter}
						modifiers={[restrictToVerticalAxis, restrictToParentElement]}
						onDragStart={onDragStart}
						onDragEnd={onDragEnd}
					>
						<SortableContext items={sortableRowComponents} strategy={verticalListSortingStrategy}>
							{sortableRowComponents.map(({ id, row }) => (
								<DraggableWrapper className='connector-keyvalue__draggable-wrapper' key={id} id={id}>
									{row}
								</DraggableWrapper>
							))}
						</SortableContext>
						<DragOverlay>
							{activeRow ? (
								<OverlayRow>{sortableRowComponents.find((c) => c.id === activeRow).row}</OverlayRow>
							) : null}
						</DragOverlay>
					</DndContext>
				) : (
					sortableRowComponents[0].row
				)}
			</div>
			<SlIcon className='connector-keyvalue__add-row' onClick={onAddRowClick} name='plus-circle' />
			{error !== '' && <div className='connector-ui__fields-error'>{error}</div>}
		</div>
	);
};

const transformRows = (value: KeyValueValue): KeyValueRow[] => {
	if (value !== '' && value.length > 0) {
		const rows: any[] = [];
		let counter = 1;
		for (const v of value) {
			const { key, value } = v;
			rows.push({ id: counter, key: key, value: value });
			counter++;
		}
		return rows;
	} else {
		return [{ id: 1, key: '', value: '' }];
	}
};

const normalizeRows = (rows: KeyValueRow[]): KeyValueValue => {
	const formatted = [];
	for (const row of rows) {
		formatted.push({ key: row.key, value: row.value });
	}
	return formatted;
};

export default ConnectorKeyValue;
