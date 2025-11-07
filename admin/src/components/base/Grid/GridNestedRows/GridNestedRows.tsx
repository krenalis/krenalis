import React, { useState, ReactNode, Fragment } from 'react';
import './GridNestedRows.css';
import GridRow from '../GridRow/GridRow';
import { NestedGridRows, GridColumn, SortableGridRow } from '../Grid.types';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
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
import { DraggableWrapper } from '../DraggableWrapper/DraggableWrapper';
import { OverlayRow } from '../../OverlayRow/OverlayRow';

interface GridNestedRowsProps {
	rows: NestedGridRows;
	columns: GridColumn[];
	className?: string;
	nesting: number;
	onSortRow?: (overRowID: string, movedRowID: string) => void;
	isSortable?: boolean;
	reloadColumnsWidths: () => void;
}

const GridNestedRows = ({
	rows,
	columns,
	className,
	nesting,
	onSortRow,
	isSortable,
	reloadColumnsWidths,
}: GridNestedRowsProps) => {
	const [activeRow, setActiveRow] = useState(null);
	const [isExpanded, setIsExpanded] = useState(false);
	const sensors = useSensors(
		useSensor(PointerSensor),
		useSensor(KeyboardSensor, {
			coordinateGetter: sortableKeyboardCoordinates,
		}),
	);

	const onDragEnd = (e) => {
		const { over, active } = e;
		if (over.id !== active.id) {
			onSortRow(over.id, active.id);
		}
		setActiveRow(null);
	};

	const onDragStart = (e) => {
		const { active } = e;
		setActiveRow(active.id);
	};

	const onExpand = () => {
		setIsExpanded(!isExpanded);
		reloadColumnsWidths();
	};

	let parentComponent: ReactNode = null;
	let childrenComponents: any[] = [];
	for (const [i, row] of rows.entries()) {
		if (Array.isArray(row)) {
			const r = row as NestedGridRows;
			const component = (
				<GridNestedRows
					key={i}
					rows={r}
					columns={columns}
					className='grid__nested-rows grid__nested-rows--children'
					nesting={nesting + 1}
					onSortRow={onSortRow}
					isSortable={isSortable}
					reloadColumnsWidths={reloadColumnsWidths}
				/>
			);
			if (isSortable) {
				const r = row as SortableGridRow[];
				childrenComponents.push({
					id: r[0].dragKey,
					row: component,
				});
			} else {
				childrenComponents.push(component);
			}
		} else {
			const r = row as any;
			if (i === 0) {
				parentComponent = (
					<Fragment key={i}>
						<SlIcon className='grid__row-expand' name='caret-right-fill' onClick={onExpand}></SlIcon>
						<GridRow row={r} columns={columns} className='grid__row grid__row--parent' />
					</Fragment>
				);
			} else {
				const component = (
					<GridRow key={i} row={r} columns={columns} className='grid__row grid__row--children' />
				);
				if (isSortable) {
					const r = row as SortableGridRow;
					childrenComponents.push({
						id: r.dragKey,
						row: component,
					});
				} else {
					childrenComponents.push(component);
				}
			}
		}
	}

	const parentIndentation = 50 + 30 * (nesting - 1) + 'px'; // takes the indentation of the previous level.
	const childrenIndentation = 50 + 30 * nesting + 'px'; // takes the incremented indentation.
	return (
		<div
			className={`${className}${isExpanded ? ' grid__nested-rows--expanded' : ''}`}
			style={
				{
					'--parent-indentation': parentIndentation,
					'--children-indentation': childrenIndentation,
				} as React.CSSProperties
			}
		>
			{parentComponent}
			{isSortable ? (
				<DndContext
					sensors={sensors}
					collisionDetection={closestCenter}
					modifiers={[restrictToVerticalAxis, restrictToParentElement]}
					onDragStart={onDragStart}
					onDragEnd={onDragEnd}
				>
					<SortableContext items={childrenComponents} strategy={verticalListSortingStrategy}>
						{childrenComponents.map(({ id, row }) => (
							<DraggableWrapper key={id} id={id}>
								{row}
							</DraggableWrapper>
						))}
					</SortableContext>
					<DragOverlay>
						{activeRow ? (
							<OverlayRow>{childrenComponents.find((c) => c.id === activeRow).row}</OverlayRow>
						) : null}
					</DragOverlay>
				</DndContext>
			) : (
				childrenComponents
			)}
		</div>
	);
};

export default GridNestedRows;
