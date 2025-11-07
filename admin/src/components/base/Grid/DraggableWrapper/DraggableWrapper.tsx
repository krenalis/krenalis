import React, { ReactNode } from 'react';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { CSS } from '@dnd-kit/utilities';
import { useSortable } from '@dnd-kit/sortable';

interface DraggableRowProps {
	id: string | number;
	className?: string;
	children: ReactNode;
}

const DraggableWrapper = ({ id, className, children }: DraggableRowProps) => {
	const { attributes, listeners, setNodeRef, transform, transition } = useSortable({ id: id });

	const style = {
		transform: CSS.Transform.toString(transform),
		transition,
	};

	return (
		<div className={`draggable-wrapper${className ? ` ${className}` : ''}`} ref={setNodeRef} style={style}>
			<button className='draggable-wrapper__handle' {...listeners} {...attributes}>
				<SlIcon name='grip-vertical' />
			</button>
			{children}
		</div>
	);
};

export { DraggableWrapper };
