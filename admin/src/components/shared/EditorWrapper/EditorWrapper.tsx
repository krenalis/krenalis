import './EditorWrapper.css';
import React, { ReactNode } from 'react';
import Editor from '@monaco-editor/react';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlDropdown from '@shoelace-style/shoelace/dist/react/dropdown/index.js';
import SlMenu from '@shoelace-style/shoelace/dist/react/menu/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlMenuItem from '@shoelace-style/shoelace/dist/react/menu-item/index.js';

import getLanguageLogo from '../../helpers/getLanguageLogo';

interface EditorWrapperProps {
	language: string;
	languageChoices?: string[];
	onLanguageChange?: (language: string) => void;
	width?: number;
	height: number;
	value: string;
	onChange: (value: string | undefined) => void | Promise<void>;
}

const EditorWrapper = ({
	language,
	languageChoices,
	onLanguageChange,
	width,
	height,
	value,
	onChange,
	...delegated
}: EditorWrapperProps) => {
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

	const languageLogo = getLanguageLogo(language);

	return (
		<div className='editorWrapper'>
			<div className='heading'>
				<span className='languageLogo'>{languageLogo}</span>
				{languageChoices ? (
					<SlDropdown className='switchEditorLanguageDropdown'>
						<SlButton slot='trigger' variant='text' size='small' caret>
							{language}
						</SlButton>
						<SlMenu onSlSelect={onLanguageChange}>
							{languageChoices.map((language) => (
								<SlMenuItem value={language}>{language}</SlMenuItem>
							))}
						</SlMenu>
					</SlDropdown>
				) : (
					<span className='language'>{language}</span>
				)}
			</div>
			<div className='editor' style={{ width: `${width}px`, height: `${height}px` }}>
				<Editor
					value={value}
					onChange={onChange}
					onMount={onEditorDidMount}
					defaultLanguage={language.toLowerCase()}
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
