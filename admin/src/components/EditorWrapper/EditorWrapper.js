import './EditorWrapper.css';
import Editor from '@monaco-editor/react';
import { SlSpinner, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const EditorWrapper = ({ defaultLanguage, width, height, value, onChange, ...delegated }) => {
	const onEditorDidMount = (editor) => {
		let source = value;
		let div = editor._domElement;
		let height = div.offsetHeight;
		let lineCount = height / 25; // 25 is the height of a line.
		if (source == null) {
			source = '';
		}
		let sourceLines = (source.match(/\n/g) || []).length + 1; // +1 is needed for the first line
		let neededNewLines = lineCount - sourceLines - 1; // -1 is needed to prevent vertical overflow
		if (neededNewLines === 0) {
			return;
		}
		for (let i = 0; i < neededNewLines; i++) {
			source += '\n';
		}
		onChange(source);
	};

	let iconName;
	switch (defaultLanguage) {
		case 'sql':
			iconName = 'filetype-sql';
			break;
		case 'javascript':
			iconName = 'filetype-js';
			break;
		case 'python':
			iconName = 'filetype-py';
			break;
		default:
			iconName = 'file-earmark-code';
			break;
	}

	return (
		<div className='editorWrapper'>
			<div className='heading'>
				<SlIcon name={iconName}></SlIcon>
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
						},
						lineHeight: 25,
						smoothScrolling: true,
						cursorSmoothCaretAnimation: 'on',
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
