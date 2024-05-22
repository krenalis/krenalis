import React, { ReactNode } from 'react';
import './Flex.css';

interface FlexProps {
	className?: string;
	justifyContent?: string;
	alignItems?: string;
	gap?: number;
	children: ReactNode;
}

const Flex = ({ className, justifyContent, alignItems, gap, children }: FlexProps) => {
	return (
		<div
			className={`flex${className != null ? ` ${className}` : ''}`}
			style={{ justifyContent: justifyContent, alignItems: alignItems, gap: `${gap}px` }}
		>
			{children}
		</div>
	);
};

export default Flex;
