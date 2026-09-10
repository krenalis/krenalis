import React, { useState } from 'react';
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
import { OverlayRow } from '../OverlayRow/OverlayRow';
import { DraggableWrapper } from './DraggableWrapper/DraggableWrapper';
import { SortableRowComponent } from './Grid.types';

interface SortableRowsProps {
	children: SortableRowComponent[];
	className?: string;
	disabled?: boolean;
	onSortRow: (overRowID: string, movedRowID: string) => void;
}

const SortableRows = ({ children, className, disabled, onSortRow }: SortableRowsProps) => {
	const [activeRow, setActiveRow] = useState(null);
	const sensors = useSensors(
		useSensor(PointerSensor),
		useSensor(KeyboardSensor, {
			coordinateGetter: sortableKeyboardCoordinates,
		}),
	);

	const onDragEnd = (event) => {
		const { over, active } = event;
		if (over != null && over.id !== active.id) {
			onSortRow(over.id, active.id);
		}
		setActiveRow(null);
	};

	const onDragStart = (event) => {
		setActiveRow(event.active.id);
	};

	const sortableRows = (
		<DndContext
			sensors={sensors}
			collisionDetection={closestCenter}
			modifiers={[restrictToVerticalAxis, restrictToParentElement]}
			onDragStart={onDragStart}
			onDragEnd={onDragEnd}
		>
			<SortableContext items={children} strategy={verticalListSortingStrategy}>
				{children.map(({ id, row }) => (
					<DraggableWrapper key={id} id={id} disabled={disabled}>
						{row}
					</DraggableWrapper>
				))}
			</SortableContext>
			<DragOverlay>
				{activeRow ? (
					<OverlayRow>{children.find((component) => component.id === activeRow).row}</OverlayRow>
				) : null}
			</DragOverlay>
		</DndContext>
	);

	return className == null ? sortableRows : <div className={className}>{sortableRows}</div>;
};

export { SortableRows };
