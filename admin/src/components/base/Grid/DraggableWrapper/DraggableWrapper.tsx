import React, { ReactNode } from 'react';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { CSS } from '@dnd-kit/utilities';
import { useSortable } from '@dnd-kit/sortable';

interface DraggableRowProps {
	id: string | number;
	className?: string;
	children: ReactNode;
	disabled?: boolean;
}

const DraggableWrapper = ({ id, className, children, disabled }: DraggableRowProps) => {
	const { attributes, listeners, setNodeRef, transform, transition } = useSortable({ id, disabled });

	const style = {
		transform: CSS.Transform.toString(transform),
		transition,
	};

	return (
		<div className={`draggable-wrapper${className ? ` ${className}` : ''}`} ref={setNodeRef} style={style}>
			<button className='draggable-wrapper__handle' {...listeners} {...attributes} disabled={disabled}>
				<SlIcon name='grip-vertical' />
			</button>
			{children}
		</div>
	);
};

export { DraggableWrapper };
