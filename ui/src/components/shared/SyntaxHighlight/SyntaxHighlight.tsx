import React, { ReactNode } from 'react';
import './SyntaxHighlight.css';
import SyntaxHighlighter from 'react-syntax-highlighter';
import { arduinoLight } from 'react-syntax-highlighter/dist/esm/styles/hljs';

interface SyntaxHighlightProps {
	children: ReactNode;
	language?: string;
}

const SyntaxHighlight = ({ children, language }: SyntaxHighlightProps) => {
	return (
		<div className='syntax-highlight'>
			<SyntaxHighlighter language={language ? language : 'javascript'} style={arduinoLight}>
				{children}
			</SyntaxHighlighter>
		</div>
	);
};

export default SyntaxHighlight;
