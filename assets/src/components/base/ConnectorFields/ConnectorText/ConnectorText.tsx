import React from 'react';
import './ConnectorText.css';

interface ConnectorTextProps {
	label: string;
	text: string;
}

const ConnectorText = ({ label, text }: ConnectorTextProps) => {
	return (
		<div className='connector-text'>
			<div className='connector-text__label'>{label}</div>
			<div className='connector-text__text'>{text}</div>
		</div>
	);
};

export default ConnectorText;
