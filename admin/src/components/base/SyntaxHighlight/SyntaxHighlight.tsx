import React, { ReactNode } from 'react';
import './SyntaxHighlight.css';
import SyntaxHighlighter from 'react-syntax-highlighter';
import { arduinoLight } from 'react-syntax-highlighter/dist/esm/styles/hljs';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';

interface SyntaxHighlightProps {
	children: ReactNode;
	language?: string;
	showLineNumbers?: boolean;
	wrapLines?: boolean;
	lineNumberStyle?: any;
	icon?: string;
	className?: string;
	lineProps?: (lineNumber: number) => any;
}

const SyntaxHighlight = ({
	children,
	language,
	showLineNumbers,
	wrapLines = false,
	lineNumberStyle,
	icon,
	className,
	lineProps,
}: SyntaxHighlightProps) => {
	return (
		<div
			className={`syntax-highlight${className != null ? ' ' + className : ''}${icon != null ? ' syntax-highlight--icon' : ''}`}
		>
			{icon != null && <SlIcon name={icon} />}
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
