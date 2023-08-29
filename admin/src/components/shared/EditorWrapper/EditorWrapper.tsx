import './EditorWrapper.css';
import React, { ReactNode } from 'react';
import Editor from '@monaco-editor/react';
import { SlSpinner } from '@shoelace-style/shoelace/dist/react/index.js';

// TODO: serve static assets from the server.
const SQL_LOGO =
	'';
const JS_LOGO =
	'';
const PYTHON_LOGO =
	'';

interface EditorWrapperProps {
	defaultLanguage: string;
	width?: number;
	height: number;
	value: string;
	onChange: (value: string | undefined) => void | Promise<void>;
}

const EditorWrapper = ({ defaultLanguage, width, height, value, onChange, ...delegated }: EditorWrapperProps) => {
	const onEditorDidMount = (editor) => {
		let source = value;
		const div = editor._domElement;
		const height = div.offsetHeight;
		const lineCount = height / 25; // 25 is the height of a line.
		if (source == null) {
			source = '';
		}
		const sourceLines = (source.match(/\n/g) || []).length + 1; // +1 is needed for the first line
		const neededNewLines = lineCount - sourceLines - 1; // -1 is needed to prevent vertical overflow
		if (neededNewLines === 0) {
			return;
		}
		for (let i = 0; i < neededNewLines; i++) {
			source += '\n';
		}
		onChange(source);
	};

	let languageLogo: ReactNode;
	switch (defaultLanguage) {
		case 'sql':
			languageLogo = <img src={SQL_LOGO} alt='SQL logo' />;
			break;
		case 'javascript':
			languageLogo = <img src={JS_LOGO} alt='Javascript logo' />;
			break;
		case 'python':
			languageLogo = <img src={PYTHON_LOGO} alt='Python logo' />;
			break;
		default:
			languageLogo = 'file-earmark-code';
			break;
	}

	return (
		<div className='editorWrapper'>
			<div className='heading'>
				<span className='languageLogo'>{languageLogo}</span>
				<span className='language'>{defaultLanguage}</span>
			</div>
			<div className='editor' style={{ width: `${width}px`, height: `${height}px` }}>
				<Editor
					value={value}
					onChange={onChange}
					onMount={onEditorDidMount}
					defaultLanguage={defaultLanguage}
					options={{
						minimap: { enabled: false },
						scrollbar: {
							useShadows: false,
							verticalScrollbarSize: 8,
							verticalSliderSize: 8,
							horizontalScrollbarSize: 8,
							horizontalSliderSize: 8,
							alwaysConsumeMouseWheel: false,
						},
						lineHeight: 25,
						fontSize: 16,
						smoothScrolling: true,
						cursorSmoothCaretAnimation: true,
						overviewRulerBorder: false,
						overviewRulerLanes: 0,
						renderLineHighlight: 'none',
						scrollBeyondLastLine: false,
						renderWhitespace: 'none',
					}}
					loading={<SlSpinner style={{ fontSize: '30px' }}></SlSpinner>}
					{...delegated}
				/>
			</div>
		</div>
	);
};

export default EditorWrapper;
