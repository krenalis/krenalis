import React, { useEffect, useState, ReactNode } from 'react';
import './EditorWrapper.css';
import Editor from '@monaco-editor/react';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SlDropdown from '@shoelace-style/shoelace/dist/react/dropdown/index.js';
import SlMenu from '@shoelace-style/shoelace/dist/react/menu/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlMenuItem from '@shoelace-style/shoelace/dist/react/menu-item/index.js';

import getLanguageLogo from '../../helpers/getLanguageLogo';

interface EditorWrapperProps {
	language: string;
	languageChoices?: string[];
	onLanguageChange?: (language: string) => void;
	languageDropdownRef?: any;
	actions?: ReactNode;
	width?: number;
	height?: number;
	name: string;
	value: string;
	onChange: (value: string | undefined) => void | Promise<void>;
	onClick?: () => void;
	onMount?: (editor: any) => void;
	isReadOnly?: boolean;
	showHeading?: boolean;
	className?: string;
}

const EditorWrapper = ({
	language,
	languageChoices,
	onLanguageChange,
	languageDropdownRef,
	actions,
	width,
	height,
	name,
	value,
	onChange,
	isReadOnly,
	onClick,
	onMount,
	showHeading,
	className,
	...delegated
}: EditorWrapperProps) => {
	const [key, setKey] = useState(name);

	useEffect(() => {
		setKey(`${name}-${language}`);
	}, [language]);

	const onEditorDidMount = (editor) => {
		if (onMount) {
			onMount(editor);
		}
	};

	const languageLogo = getLanguageLogo(language);

	return (
		<div className={`editorWrapper${className ? ' ' + className : ''}`} onClick={onClick}>
			{showHeading && (
				<div className='heading'>
					<div className='logoAndLanguage'>
						<span className='languageLogo'>{languageLogo}</span>
						{languageChoices ? (
							<SlDropdown className='switchEditorLanguageDropdown' ref={languageDropdownRef}>
								<SlButton slot='trigger' variant='text' size='small' caret>
									{language}
								</SlButton>
								<SlMenu onSlSelect={onLanguageChange}>
									{languageChoices.map((language) => (
										<SlMenuItem key={language} value={language}>
											{language}
										</SlMenuItem>
									))}
								</SlMenu>
							</SlDropdown>
						) : (
							<span className='language'>{language}</span>
						)}
					</div>
					<div className='actions'>{actions}</div>
				</div>
			)}
			<div className='editor' style={{ width: width ? `${width}px` : '', height: height ? `${height}px` : '' }}>
				<Editor
					key={key}
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
						readOnly: isReadOnly,
					}}
					loading={<SlSpinner style={{ fontSize: '30px' }}></SlSpinner>}
					{...delegated}
				/>
			</div>
		</div>
	);
};

export default EditorWrapper;
