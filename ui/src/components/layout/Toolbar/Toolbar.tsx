import React, { ReactNode } from 'react';
import './Toolbar.css';

interface ToolbarProps {
	children: ReactNode;
}

const Toolbar = ({ children }: ToolbarProps) => {
	return <div className='toolbar'>{children}</div>;
};

export default Toolbar;
