import React, { CSSProperties, ReactNode, useState } from 'react';
import './GridHeaderRow.css';
import GridHeaderCell from '../GridHeaderCell/GridHeaderCell';
import { getChildIndexClassname } from '../Grid.helpers';
import { GridColumn } from '../Grid.types';
import {
	closestCenter,
	DndContext,
	DragOverlay,
	DragEndEvent,
	DragStartEvent,
	KeyboardSensor,
	PointerSensor,
	useSensor,
	useSensors,
} from '@dnd-kit/core';
import { restrictToHorizontalAxis, restrictToParentElement } from '@dnd-kit/modifiers';
import {
	horizontalListSortingStrategy,
	SortableContext,
	sortableKeyboardCoordinates,
	useSortable,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';

interface GridHeaderRowProps {
	columns: GridColumn[];
	onSortColumn?: (overColumnKey: string, movedColumnKey: string) => void;
	semanticRow?: boolean;
}

interface SortableGridHeaderCellProps {
	className: string;
	column: GridColumn;
	columnKey: string;
	semanticCell?: boolean;
}

const SortableGridHeaderCell = ({ className, column, columnKey, semanticCell }: SortableGridHeaderCellProps) => {
	const { attributes, isDragging, listeners, setActivatorNodeRef, setNodeRef, transform, transition } = useSortable({
		id: columnKey,
	});
	const style: CSSProperties = {
		transform: CSS.Translate.toString(transform),
		transition,
	};

	return (
		<GridHeaderCell
			ref={setNodeRef}
			className={`${className} grid__header-cell--reorderable${isDragging ? ' grid__header-cell--dragging' : ''}`}
			value={column.name}
			alignment={column.alignment}
			explanation={column.explanation}
			reorderHandle={
				<button
					ref={setActivatorNodeRef}
					type='button'
					className='grid__column-reorder-handle'
					{...attributes}
					{...listeners}
					aria-label={`Move ${column.name} column`}
				>
					<SlIcon name='grip-vertical' aria-hidden='true' />
				</button>
			}
			semanticCell={semanticCell}
			style={style}
		/>
	);
};

const GridHeaderRow = ({ columns, onSortColumn, semanticRow }: GridHeaderRowProps) => {
	const [activeColumnKey, setActiveColumnKey] = useState<string>();
	const sensors = useSensors(
		useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
		useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
	);
	const sortableColumnKeys = columns.flatMap((column) =>
		column.reorderable && column.key != null ? [column.key] : [],
	);
	const canReorderColumns = onSortColumn != null && sortableColumnKeys.length > 1;
	const activeColumn = columns.find((column) => column.key === activeColumnKey);
	const gridHeaderCells = [] as ReactNode[];
	for (const [i, column] of columns.entries()) {
		const className = getChildIndexClassname(i, columns.length);
		const isReorderable = column.reorderable === true && column.key != null;
		const reactKey = column.key ?? column.name;
		if (canReorderColumns && isReorderable) {
			gridHeaderCells.push(
				<SortableGridHeaderCell
					key={reactKey}
					className={`grid__header-cell ${className}`}
					column={column}
					columnKey={column.key}
					semanticCell={semanticRow}
				/>,
			);
			continue;
		}
		gridHeaderCells.push(
			<GridHeaderCell
				key={reactKey}
				className={`grid__header-cell ${className}`}
				value={column.name}
				alignment={column.alignment}
				explanation={column.explanation}
				semanticCell={semanticRow}
			/>,
		);
	}

	const headerRow = (
		<div className='grid__header-row' role={semanticRow ? 'row' : undefined}>
			{gridHeaderCells}
		</div>
	);
	if (!canReorderColumns) {
		return headerRow;
	}

	const onDragEnd = ({ active, over }: DragEndEvent) => {
		if (over != null && active.id !== over.id) {
			onSortColumn(String(over.id), String(active.id));
		}
		setActiveColumnKey(undefined);
	};

	return (
		<DndContext
			sensors={sensors}
			collisionDetection={closestCenter}
			modifiers={[restrictToHorizontalAxis, restrictToParentElement]}
			onDragCancel={() => setActiveColumnKey(undefined)}
			onDragEnd={onDragEnd}
			onDragStart={({ active }: DragStartEvent) => setActiveColumnKey(String(active.id))}
		>
			<SortableContext items={sortableColumnKeys} strategy={horizontalListSortingStrategy}>
				{headerRow}
			</SortableContext>
			<DragOverlay>
				{activeColumn != null ? (
					<div className='grid__column-reorder-overlay' aria-hidden='true'>
						<GridHeaderCell
							className='grid__header-cell grid__header-cell--reorderable grid__header-cell--reorder-overlay'
							value={activeColumn.name}
							alignment={activeColumn.alignment}
							reorderHandle={
								<span className='grid__column-reorder-handle'>
									<SlIcon name='grip-vertical' aria-hidden='true' />
								</span>
							}
						/>
					</div>
				) : null}
			</DragOverlay>
		</DndContext>
	);
};

export default GridHeaderRow;
