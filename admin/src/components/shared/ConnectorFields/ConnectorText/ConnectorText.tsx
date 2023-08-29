import React from 'react';
import './ConnectorText.css';

interface ConnectorTextProps {
	label: string;
	text: string;
}

const ConnectorText = ({ label, text }: ConnectorTextProps) => {
	return (
		<div className='connectorText'>
			<div className='label'>{label}</div>
			<div className='text'>{text}</div>
		</div>
	);
};

export default ConnectorText;
