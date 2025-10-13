import React, { useEffect, ReactNode, useRef } from 'react';
import './EditorWrapper.css';
import * as monaco from 'monaco-editor';

async function loadMonacoEditor() {
	window.MonacoEnvironment = {
		getWorkerUrl: function (_, label) {
			switch (label) {
				case 'javascript':
					return '/admin/src/monaco/vs/language/typescript/ts.worker.js';
				case 'editorWorkerService':
					return '/admin/src/monaco/vs/editor/editor.worker.js';
			}
			throw new Error('unexpected Monaco worker label ' + label);
		},
	};
	await import('monaco-editor');
}
void loadMonacoEditor();

interface EditorWrapperProps {
	language: string;
	languageChoices?: string[];
	onLanguageChange?: (e) => void;
	languageDropdownRef?: any;
	actions?: ReactNode;
	width?: number;
	height?: number;
	name: string;
	value: string;
	sync?: boolean;
	onChange?: (value: string | undefined) => void | Promise<void>;
	onClick?: () => void;
	onMount?: (editor: any) => void;
	isReadOnly?: boolean;
	hideGutter?: boolean;
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
	sync,
	onChange,
	isReadOnly,
	onClick,
	onMount,
	hideGutter,
	className,
	...delegated
}: EditorWrapperProps) => {
	useEffect(() => {
		const href = '/admin/src/monaco/vs/editor/editor.main.css';
		const isAlreadyLoaded = document.querySelector(`link[rel="stylesheet"][href="${href}"]`);
		if (!isAlreadyLoaded) {
			const link = document.createElement('link');
			link.rel = 'stylesheet';
			link.href = href;
			document.head.appendChild(link);
		}
	}, []);

	return (
		<div
			className={`editor-wrapper${className ? ' ' + className : ''}${hideGutter ? ' editor-wrapper--hide-gutter' : ''}`}
			onClick={onClick}
		>
			<div
				className='editor-wrapper__editor'
				style={{ width: width ? `${width}px` : '', height: height ? `${height}px` : '' }}
			>
				<Editor
					value={value}
					sync={sync}
					onChange={onChange}
					onMount={onMount}
					language={language.toLowerCase()}
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
						lineNumbers: hideGutter ? 'off' : undefined,
						glyphMargin: hideGutter ? false : undefined,
						folding: hideGutter ? false : undefined,
						lineDecorationsWidth: hideGutter ? 0 : undefined,
						lineNumbersMinChars: hideGutter ? 0 : undefined,
						...delegated,
					}}
				/>
			</div>
		</div>
	);
};

interface EditorProps {
	value: string;
	sync?: boolean;
	language: string;
	onChange?: (value: string) => void;
	onMount?: (editor: any) => void;
	options: Record<string, any>;
}

const Editor = ({ value, sync, language, onChange, onMount, options }: EditorProps) => {
	const containerRef = useRef(null);
	const editorRef = useRef(null);

	useEffect(() => {
		if (!containerRef.current) {
			return;
		}

		const model = monaco.editor.createModel(value, language);
		editorRef.current = monaco.editor.create(containerRef.current, {
			value,
			language: language.toLowerCase(),
			...options,
		});

		const disposable = editorRef.current.onDidChangeModelContent(() => {
			const val = editorRef.current.getValue();
			onChange?.(val);
		});

		const resizeObserver = new ResizeObserver(() => {
			editorRef.current?.layout();
		});
		resizeObserver.observe(containerRef.current);

		// force layout after mount
		setTimeout(() => {
			editorRef.current?.layout();
		}, 0);

		if (onMount != null) {
			onMount(editorRef.current);
		}

		return () => {
			disposable.dispose();
			editorRef.current?.dispose();
			model.dispose();
			resizeObserver.disconnect();
		};
	}, []);

	useEffect(() => {
		const handler = (event: PromiseRejectionEvent) => {
			// avoid the behaviour of Monaco which prints an error in
			// the console when an asynchronous editor operation is
			// cancelled after the editor is closed. The cancellation of
			// operations is a standard optimisation task and is not to
			// be regarded as an error in our use case.
			if (event.reason?.name === 'Canceled' || event.reason?.message === 'Canceled') {
				event.preventDefault();
			}
		};
		window.addEventListener('unhandledrejection', handler);
		return () => {
			window.removeEventListener('unhandledrejection', handler);
		};
	}, []);

	useEffect(() => {
		const model = editorRef.current.getModel();
		monaco.editor.setModelLanguage(model, language.toLowerCase());
		editorRef.current.setValue(value);
	}, [language]);

	useEffect(() => {
		setTimeout(() => {
			editorRef.current.setValue(value);
		}, 50);
	}, [sync]);

	return (
		<div
			ref={containerRef}
			style={{
				width: '100%',
				height: '100%',
			}}
		/>
	);
};

export default EditorWrapper;
