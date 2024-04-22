import React, { ReactNode } from 'react';
import './Toolbar.css';

interface ToolbarProps {
	className?: string;
	children: ReactNode;
}

const Toolbar = ({ className, children }: ToolbarProps) => {
	return <div className={`toolbar${className ? ' ' + className : ''}`}>{children}</div>;
};

export default Toolbar;
