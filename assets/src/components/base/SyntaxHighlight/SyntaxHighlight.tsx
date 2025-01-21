import React, { ReactNode } from 'react';
import './SyntaxHighlight.css';
import SyntaxHighlighter from 'react-syntax-highlighter';
import { arduinoLight } from 'react-syntax-highlighter/dist/esm/styles/hljs';

interface SyntaxHighlightProps {
	children: ReactNode;
	language?: string;
	showLineNumbers?: boolean;
	wrapLines?: boolean;
	lineNumberStyle?: any;
	lineProps?: (lineNumber: number) => any;
}

const SyntaxHighlight = ({
	children,
	language,
	showLineNumbers,
	wrapLines = false,
	lineNumberStyle,
	lineProps,
}: SyntaxHighlightProps) => {
	return (
		<div className='syntax-highlight'>
			<SyntaxHighlighter
				language={language ? language : 'javascript'}
				showLineNumbers={showLineNumbers}
				wrapLines={wrapLines}
				lineNumberStyle={lineNumberStyle}
				lineProps={lineProps}
				style={arduinoLight}
			>
				{children}
			</SyntaxHighlighter>
		</div>
	);
};

export default SyntaxHighlight;
