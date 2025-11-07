import React, { ReactNode } from 'react';
import './DangerZone.css';

interface DangerZoneProps {
	className?: string;
	children: ReactNode;
}

const DangerZone = ({ className, children }: DangerZoneProps) => {
	return <div className={`danger-zone${className ? ' ' + className : ''}`}>{children}</div>;
};

export default DangerZone;
