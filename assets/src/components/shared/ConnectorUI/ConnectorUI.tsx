import React, { ReactNode } from 'react';
import './ConnectorUI.css';
import { ConnectorUIContext } from '../../../context/ConnectorUIContext';
import { ConnectorValues } from '../../../lib/api/types/responses';

interface ConnectorUIProps {
	fields: ReactNode[];
	buttons?: ReactNode[];
	values: ConnectorValues;
	onChange: (name: string, value: any) => void;
	children?: ReactNode;
}

const ConnectorUI = ({ fields, buttons, values, onChange, children }: ConnectorUIProps) => {
	return (
		<div className='connector-ui'>
			<ConnectorUIContext.Provider value={{ values, onChange }}>
				<div className='connector-ui__fields'>{fields}</div>
			</ConnectorUIContext.Provider>
			{children && children}
			{buttons && <div className='connector-ui__buttons'>{buttons}</div>}
		</div>
	);
};

export default ConnectorUI;
