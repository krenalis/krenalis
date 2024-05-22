import './Popover.css';
import React from 'react';

interface PopoverProps {
	isOpen: boolean;
	content: string;
}

const Popover = ({ isOpen, content }: PopoverProps) => {
	return <div className={`popover${isOpen ? ' popover--open' : ''}`}>{content}</div>;
};

export { Popover };
