import React, { ReactNode } from 'react';
import './SyntaxHighlight.css';
import SyntaxHighlighter from 'react-syntax-highlighter';
import { arduinoLight } from 'react-syntax-highlighter/dist/esm/styles/hljs';

interface SyntaxHighlightProps {
	children: ReactNode;
}

const SyntaxHighlight = ({ children }: SyntaxHighlightProps) => {
	return (
		<div className='syntax-highlight'>
			<SyntaxHighlighter language='javascript' style={arduinoLight}>
				{children}
			</SyntaxHighlighter>
		</div>
	);
};

export default SyntaxHighlight;
