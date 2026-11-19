import React, { ReactNode, useState, useMemo, useRef, useEffect, useLayoutEffect, useContext } from 'react';
import { flushSync } from 'react-dom';
import './Combobox.css';
import { ComboboxKeyboardHints } from './ComboboxKeyboardHints';
import WaInput from '@awesome.me/webawesome/dist/react/input/index.js';
import WaIcon from '@awesome.me/webawesome/dist/react/icon/index.js';
import WaTabGroup from '@awesome.me/webawesome/dist/react/tab-group/index.js';
import WaTabPanel from '@awesome.me/webawesome/dist/react/tab-panel/index.js';
import WaTab from '@awesome.me/webawesome/dist/react/tab/index.js';
import { ComboboxItem } from './Combobox.types';
import { ExpressionFragment, parseMapExpression } from '../../../utils/mapExpression';
import { autocompleteExpression } from './Combobox.helpers';
import { KRENALIS_FUNCTIONS, KrenalisFunction } from '../../../constants/function';
import appContext from '../../../context/AppContext';
import { debounceWithAbort } from '../../../utils/debounce';
import ConnectionContext from '../../../context/ConnectionContext';
import pipelineContext from '../../../context/PipelineContext';
import Type from '../../../lib/api/types/types';

interface SuggestionGroup {
	name: string;
	label: string;
	type: 'enum' | 'property' | 'function';
	items: ComboboxItem[];
}

interface HighlightedOption {
	group: string;
	term: string;
}

interface ComboboxProps {
	value: string;
	displayValue?: string;
	items: ComboboxItem[];
	onInput: (path: string, value: string) => void;
	onSelect: (path: string, value: string) => void;
	name: string;
	isExpression: boolean;
	updateError?: (path: string, errorMessage: string) => void;
	type?: Type;
	enumValues?: string[];
	size?: 'small' | 'medium' | 'large';
	className?: string;
	error?: string;
	caret?: boolean;
	clearable?: boolean;
	controlled?: boolean;
	autoResize?: boolean;
	disabled?: boolean;
	indentation?: number;
	children?: ReactNode;
	label?: string;
	helpText?: string;
	syncOnChange?: any;
	propertiesToHide?: string[] | null;
	[key: string]: any;
}

// Combobox is a combobox component specifically designed to display and handle
// schema properties and expressions.
const Combobox = ({
	value,
	displayValue,
	items,
	onInput: onInputFunc,
	onSelect: onSelectFunc,
	name,
	isExpression,
	updateError,
	type,
	enumValues,
	size = 'medium',
	className,
	error,
	caret = false,
	clearable = false,
	controlled = false,
	autoResize,
	disabled = false,
	indentation,
	children,
	label,
	helpText,
	syncOnChange,
	propertiesToHide,
	...rest
}: ComboboxProps) => {
	const [val, setVal] = useState<string>(displayValue ?? value ?? '');
	const [cursorPosition, setCursorPosition] = useState<number>();
	const [isOpen, setIsOpen] = useState<boolean>(false);
	const [listWidth, setListWidth] = useState<number>();
	const [selectedTab, setSelectedTab] = useState<string>();
	const [keyboardHighlight, setKeyboardHighlight] = useState<HighlightedOption | 'tabs' | null>(null);
	const [isErrorExpanded, setIsErrorExpanded] = useState<boolean>(false);
	const [isErrorOverflowing, setIsErrorOverflowing] = useState<boolean>(false);

	const inputRef = useRef<any>();
	const isFirstControl = useRef<any>(true);
	const errorContainerRef = useRef<any>();
	const listRef = useRef<any>();
	const bodyRef = useRef<HTMLDivElement>(null);
	const optionsRef = useRef<HTMLDivElement>(null);
	const programmaticFocus = useRef(false);
	const isEditing = useRef(false);

	const { api, handleError } = useContext(appContext);
	const { connection } = useContext(ConnectionContext);
	const { pipelineType } = useContext(pipelineContext);

	useLayoutEffect(() => {
		setIsErrorExpanded(false);
		if (error !== '') {
			const container = errorContainerRef.current;
			if (container == null) {
				return;
			}
			setIsErrorOverflowing(container.scrollWidth > container.clientWidth);
		}
	}, [error]);

	const updateCursorPosition = (setStart?: boolean) => {
		setTimeout(() => {
			const inputElement = inputRef.current?.input;
			if (inputElement) {
				setCursorPosition(setStart ? 0 : inputElement.selectionStart);
			}
		});
	};

	const onClick = (event: React.MouseEvent) => {
		if (event.nativeEvent.composedPath()[0] !== inputRef.current?.input) {
			// WaInput also receives clicks from controls such as the
			// dropdown caret.
			return;
		}
		setKeyboardHighlight(null);
		setIsOpen(true);
		if (isExpression) {
			updateCursorPosition();
		} else {
			// Set the cursor to the start to show every item in the list.
			updateCursorPosition(true);
		}
	};

	useEffect(() => {
		if (inputRef.current == null) {
			return;
		}

		// Add a delay to ensure that the shadow DOM is fully loaded.
		const timeout = window.setTimeout(() => {
			const input = inputRef.current?.input;
			if (input != null) {
				// Disable password managers autofill on the input.
				input.setAttribute('data-1p-ignore', '');
				input.setAttribute('data-bwignore', '');
				input.setAttribute('data-form-type', 'other');
				input.setAttribute('data-lpignore', 'true');

				input.setAttribute('role', 'combobox');
				input.setAttribute('aria-autocomplete', 'list');
				input.setAttribute('aria-expanded', String(listRef.current != null));
			}
		});
		return () => window.clearTimeout(timeout);
	}, []);

	useEffect(() => {
		if (inputRef.current == null || !caret) {
			return;
		}

		setTimeout(() => {
			const inputSuffix = inputRef.current.shadowRoot?.querySelector('[part="end"]');
			inputSuffix.addEventListener('click', () => {
				inputRef.current.focus();
				setKeyboardHighlight(null);
				setIsOpen(true);
				updateCursorPosition(true);
			});
		});
	}, [inputRef.current, caret]);

	useEffect(
		() => {
			setTimeout(() => {
				updateComboboxValue(value);
			});
		},
		Array.isArray(syncOnChange) ? syncOnChange : [],
	);

	useEffect(() => {
		if (controlled) {
			const inputValue = isEditing.current ? value : (displayValue ?? value);
			if (isFirstControl.current) {
				// Defer update to next event loop tick to ensure the input is
				// mounted before updating its value.
				setTimeout(() => {
					updateComboboxValue(inputValue);
					isFirstControl.current = false;
				});
			} else {
				updateComboboxValue(inputValue);
			}
		}
		if (autoResize && !isFirstControl.current) {
			// Resize the combobox. The combobox is resized when the value
			// changes or when the error changes (to take into consideration the
			// error icon shown in the suffix slot of the input).
			resizeCombobox();
		}
	}, [displayValue, value, error]);

	useEffect(() => {
		if (!isOpen || listRef.current == null) {
			return;
		}
		const timeout = window.setTimeout(() => {
			// See if the menu is overflowing the width of the input and
			// in that case expand it to avoid horizontal scrollbars.
			const menu = listRef.current;
			if (menu == null) {
				return;
			}

			const menuItems = optionsRef.current?.children ?? [];

			// Get the width of the wider menu item.
			let maxWidth = 0;
			for (const item of Array.from<Element>(menuItems)) {
				let w: number;
				if (activeGroup?.type === 'property') {
					const iconWidth = item.querySelector<HTMLElement>('.schema-combobox-item__type').offsetWidth;
					const textWidth = item.querySelector<HTMLElement>('.schema-combobox-item__text').offsetWidth;
					w = iconWidth + textWidth;
				} else if (activeGroup?.type === 'function') {
					w = item.querySelector<HTMLElement>('.function-item').offsetWidth;
				} else {
					continue;
				}
				if (w > maxWidth) {
					maxWidth = w;
				}
			}

			// Get the width of the tabs.
			const tabs = menu.querySelectorAll('wa-tab');
			let tabsWidth = 0;
			for (const t of tabs) {
				tabsWidth += t.offsetWidth;
			}
			if (tabsWidth > maxWidth) {
				maxWidth = tabsWidth;
			}

			if (listWidth == null) {
				const isOverflowing = inputRef.current.offsetWidth < maxWidth;
				if (isOverflowing) {
					setListWidth(maxWidth + 50);
				}
			} else {
				setListWidth(maxWidth + 50);
			}
		});
		return () => window.clearTimeout(timeout);
	}, [items, cursorPosition, isOpen, selectedTab, enumValues]);

	useEffect(() => {
		const onPageClick = (e) => {
			const target = e.target;
			const isInCombobox = target.closest('.combobox') != null;
			const isInAnotherCombobox = target.closest('.combobox')?.dataset.id !== name;
			if (!isInCombobox || isInAnotherCombobox) {
				setIsOpen(false);
			}
		};

		window.addEventListener('click', onPageClick);

		return () => {
			window.removeEventListener('click', onPageClick);
		};
	}, []);

	const validateExpression = async (name: string, value: string, signal: AbortSignal) => {
		if (!isExpression || connection == null || type == null || pipelineType == null) {
			return;
		}
		let errorMessage = '';
		if (value !== '') {
			try {
				errorMessage = await api.validateExpression(value, pipelineType.inputSchema.properties, type, signal);
			} catch (err) {
				if (err.name === 'AbortError') {
					return;
				}
				handleError(err);
				return;
			}
		}
		if (errorMessage === '' && propertiesToHide != null && propertiesToHide.includes(value)) {
			errorMessage = `Property "${value}" does not exist`;
		}
		const doesNotExist = errorMessage.endsWith('does not exist');
		const isEventBasedUserImport = connection.isEventBased && connection.isSource && pipelineType.target === 'User';
		const isAppEventsExport =
			connection.isApplication && connection.isDestination && pipelineType.target === 'Event';
		if (doesNotExist) {
			if (isEventBasedUserImport) {
				errorMessage += `, perhaps you meant "traits.${value}"?`;
			} else if (isAppEventsExport) {
				errorMessage += `, perhaps you meant "properties.${value}" or "traits.${value}"?`;
			}
		}
		updateError(name, errorMessage);
	};

	const debouncedValidateExpression = useMemo(() => debounceWithAbort(validateExpression, 750), [pipelineType, type]);

	const validate = async (name: string, value: string) => {
		debouncedValidateExpression(name, value);
	};

	const onInput = (e) => {
		e.preventDefault();
		setKeyboardHighlight(null);
		isEditing.current = true;
		if (!isOpen) {
			// if the user has closed the list via escape button.
			setIsOpen(true);
		}

		const lastValue = val;
		let newValue: string = e.target.value;
		let position = inputRef.current?.input.selectionStart;

		// Autocompletion.
		if (isExpression) {
			const isPasted = Math.abs(lastValue.length - newValue.length) > 1;
			const isBackspaced = lastValue.length > newValue.length;
			const isEqual = lastValue.length === newValue.length;
			if (!isPasted && !isBackspaced && !isEqual) {
				const autocompleted = autocompleteExpression(newValue, position);
				if (autocompleted != null) {
					newValue = autocompleted;
				}
			}
		}

		updateComboboxValue(newValue);
		updateCursorPosition();
		onInputFunc(name, newValue);
		validate(name, newValue);
	};

	const updateComboboxValue = (value: string) => {
		if (inputRef.current == null || inputRef.current.input == null) {
			return;
		}
		// The update of the combobox is done programmatically on the input in
		// the DOM, and outside of the React lifecycle, to prevent race
		// conditions and cursor jumps caused by the usage of setSelectionRange
		// (necessary to manipulate the cursor position, e.g. to move the cursor
		// back after an autocompletion) when there are delays in the React
		// rendering.
		//
		// The updated value is saved in the component state but only to
		// implement the features of the component (e.g. filtering, dynamic
		// resizing, async validation etc. etc.), and it is never used to
		// directly control the value inside the input in the DOM.
		let pos = inputRef.current.input.selectionStart;
		inputRef.current.setRangeText(value, 0, -1, 'end');
		inputRef.current.setSelectionRange(pos, pos);
		setVal(value);
	};

	const resizeCombobox = () => {
		const text = inputRef.current.value;
		const canvas = document.createElement('canvas');
		const context = canvas.getContext('2d');
		const inputElement = inputRef.current.shadowRoot.querySelector('input');
		if (inputElement == null) {
			return;
		}
		const style = window.getComputedStyle(inputElement);
		context.font = style.font;
		let textWidth = context.measureText(text).width;
		if (textWidth === 0) {
			textWidth = 100; // min width.
		}

		const prefix = inputRef.current.shadowRoot.querySelector('[part="start"]');
		const prefixWidth = prefix?.offsetWidth || 0;
		const suffix = inputRef.current.shadowRoot.querySelector('[part="end"]');
		const suffixWidth = suffix?.offsetWidth || 0;

		const wrapper = inputRef.current.closest('.combobox');
		wrapper.style.width = `${textWidth + prefixWidth + suffixWidth + 30}px`;
	};

	const onInputBlur = () => {
		setKeyboardHighlight(null);
		isEditing.current = false;
		programmaticFocus.current = false;
		if (displayValue != null) {
			updateComboboxValue(displayValue);
			if (autoResize) {
				resizeCombobox();
			}
		}
	};

	const onInputFocus = () => {
		if (programmaticFocus.current) {
			// Prevent opening the list when the focus is set
			// programmatically, for example after selecting a property
			// from the list. In this case, the intent is just to place
			// the cursor back in the input to allow the user to
			// continue typing, not to immediately open the list again.
			programmaticFocus.current = false;
			return;
		}
		isEditing.current = true;
		if (displayValue != null) {
			updateComboboxValue(value);
			if (autoResize) {
				resizeCombobox();
			}
		}
		setIsOpen(true);
		updateCursorPosition();
	};

	let functionItems = useMemo(() => {
		if (!isExpression) {
			return [];
		}
		return getFunctionsComboboxItems(KRENALIS_FUNCTIONS);
	}, []);

	let fragment = useMemo(() => {
		return parseMapExpression(val == null ? '' : val, cursorPosition);
	}, [val, cursorPosition, items]);

	let { filteredProperties, filteredFunctions } = useMemo(() => {
		return filterComboboxItems(fragment, cursorPosition, val, items, isExpression ? functionItems : []);
	}, [fragment]);

	let selectedFunction = useMemo(() => {
		if (!isExpression) {
			return null;
		}
		if (fragment != null && fragment.func != null) {
			const functionName = fragment.func.name;
			return KRENALIS_FUNCTIONS.find((f) => f.name === functionName);
		}
		return null;
	}, [fragment]);

	const enumItems = useMemo(
		() => enumValues?.map((term) => ({ term, content: <div className='enum-item'>{term}</div> })),
		[enumValues],
	);
	const groups: SuggestionGroup[] = [];
	if (enumItems != null) {
		groups.push({ name: 'enum', label: 'Enum', type: 'enum', items: enumItems });
	}
	if (filteredProperties != null) {
		groups.push({ name: 'properties', label: 'Properties', type: 'property', items: filteredProperties });
	}
	const hasTabs =
		enumItems != null || (isExpression && (filteredProperties?.length > 0 || filteredFunctions?.length > 0));
	if (hasTabs && filteredFunctions != null) {
		groups.push({ name: 'functions', label: 'Functions', type: 'function', items: filteredFunctions });
	}
	const activeGroup = groups.find((group) => group.name === selectedTab) ?? groups[0];
	const hasTabNavigation = hasTabs && groups.length > 1;
	const isListVisible = isOpen && (selectedFunction != null || hasTabs || activeGroup?.items.length > 0);

	let highlight: number | 'tabs' | null = null;
	if (keyboardHighlight === 'tabs') {
		highlight = hasTabNavigation ? 'tabs' : null;
	} else if (keyboardHighlight != null && activeGroup?.name === keyboardHighlight.group) {
		const index = activeGroup.items.findIndex((item) => item.term === keyboardHighlight.term);
		highlight = index >= 0 ? index : null;
	}

	useLayoutEffect(() => {
		if (!isOpen) {
			setKeyboardHighlight(null);
			setSelectedTab(undefined);
		}
	}, [isOpen]);

	useLayoutEffect(() => {
		if (highlight == null && keyboardHighlight != null) {
			setKeyboardHighlight(null);
		}
	}, [highlight, keyboardHighlight]);

	useLayoutEffect(() => {
		const input = inputRef.current?.input;
		if (input == null) {
			return;
		}
		const body = bodyRef.current;
		const options = optionsRef.current;
		const tab = body?.querySelector<HTMLElement>(`wa-tab[panel="${activeGroup?.name}"]`);
		const target = highlight === 'tabs' ? tab : typeof highlight === 'number' ? options?.children[highlight] : null;
		// Element references link the native input in the shadow root to the visible suggestions.
		input.ariaExpanded = String(isListVisible);
		input.ariaControlsElements = [options, tab?.parentElement].filter((element) => element != null);
		input.ariaActiveDescendantElement = target ?? null;
		if (body == null || target == null) {
			return;
		}
		const bounds = body.getBoundingClientRect();
		const top = bounds.top + (target === tab ? 0 : (tab?.offsetHeight ?? 0));
		const rect = target.getBoundingClientRect();
		if (rect.top < top) {
			body.scrollTop += rect.top - top;
		} else if (rect.bottom > bounds.bottom) {
			body.scrollTop += rect.bottom - bounds.bottom;
		}
	}, [highlight, isListVisible, activeGroup?.name, activeGroup?.items]);

	useLayoutEffect(() => {
		const list = listRef.current;
		if (list == null) {
			return;
		}
		const positionList = () => {
			if (list.offsetParent == null) {
				return;
			}
			list.classList.remove('combobox-list--top');
			const bottom = list.offsetParent.getBoundingClientRect().top + list.offsetTop + list.offsetHeight;
			list.classList.toggle('combobox-list--top', bottom > window.innerHeight);
		};
		positionList();
		const observer = new ResizeObserver(positionList);
		observer.observe(list);
		return () => observer.disconnect();
	}, [isListVisible]);

	const onSelect = (e, term: string, type: 'property' | 'function' | 'enum', selectedDisplayValue?: string) => {
		e.preventDefault();
		e.stopPropagation();
		setKeyboardHighlight(null);
		isEditing.current = true;

		if (type === 'enum') {
			let v = term;
			updateComboboxValue(v);
			programmaticFocus.current = true;
			inputRef.current.focus();
			onSelectFunc(name, v);
			validate(name, v);
			setIsOpen(false);
			return;
		}
		if (type === 'property' && !isExpression && selectedDisplayValue != null) {
			updateComboboxValue(term);
			programmaticFocus.current = true;
			inputRef.current.focus();
			onSelectFunc(name, term);
			validate(name, term);
			setIsOpen(false);
			return;
		}

		let position = 0;
		let v = '';
		if (fragment.func != null && fragment.pos == null) {
			const expressionStart = val.slice(0, cursorPosition);
			const expressionEnd = val.slice(cursorPosition);
			v = `${expressionStart}${term}${expressionEnd}`;
			position = cursorPosition + term.length;
		} else if (fragment != null && fragment.pos != null) {
			const expressionStart = val.slice(0, fragment.pos.start);
			const expressionEnd = val.slice(fragment.pos.end);
			v = `${expressionStart}${term}${expressionEnd}`;
			if (val === '') {
				position = term.length;
			} else {
				position = fragment.pos.start + term.length;
			}
		} else {
			v = val + term;
			position = val.length + term.length;
		}

		if (type === 'function') {
			// add parenthesis if necessary.
			const expressionStart = v.slice(0, position);
			const expressionEnd = v.slice(position);
			const hasAlreadyParenthesis = v[position] === '(';
			if (!hasAlreadyParenthesis) {
				v = `${expressionStart}()${expressionEnd}`;
				position += 1;
			}
		} else if (type === 'property') {
			// remove parenthesis if necessary.
			if (v[position] === '(' && v[position + 1] === ')') {
				const expressionStart = v.slice(0, position);
				const expressionEnd = v.slice(position + 2);
				v = `${expressionStart}${expressionEnd}`;
			}
		}
		if (type === 'property' && fragment.func == null) {
			// if a property has been selected and is not an argument for a
			// function, the combobox list can be closed.
			setIsOpen(false);
		} else if (type === 'function') {
			// probably now the user wants to select a property to pass as
			// argument to the function.
			setSelectedTab('properties');
		}

		updateComboboxValue(v);

		programmaticFocus.current = true;
		inputRef.current.focus();
		setTimeout(() => {
			inputRef.current.setSelectionRange(position, position);
			updateCursorPosition();
		});
		onSelectFunc(name, v);
		validate(name, v);
	};

	const onKeyDown = (event: React.KeyboardEvent) => {
		if (event.nativeEvent.composedPath()[0] !== inputRef.current?.input || event.nativeEvent.isComposing) {
			return;
		}
		const { key } = event;
		if (key === 'Escape' || key === 'Tab') {
			// Remove the menu tabs before the browser advances focus.
			flushSync(() => setIsOpen(false));
			if (key === 'Escape' && isListVisible) {
				event.preventDefault();
				event.stopPropagation();
			}
			return;
		}
		const hasModifier = event.altKey || event.ctrlKey || event.metaKey || event.shiftKey;
		if (!hasModifier && (key === 'ArrowDown' || key === 'ArrowUp')) {
			setIsOpen(true);
			const count = activeGroup?.items.length ?? 0;
			if (count === 0 && !hasTabNavigation) {
				return;
			}
			const index = typeof highlight === 'number' ? highlight : -1;
			if (count === 0 || (key === 'ArrowUp' && hasTabNavigation && index <= 0)) {
				setKeyboardHighlight('tabs');
			} else {
				let nextIndex = count - 1;
				if (key === 'ArrowDown') {
					nextIndex = Math.min(index + 1, count - 1);
				} else if (index !== -1) {
					nextIndex = Math.max(0, index - 1);
				}
				setKeyboardHighlight({ group: activeGroup.name, term: activeGroup.items[nextIndex].term });
			}
		} else if (!hasModifier && highlight === 'tabs' && (key === 'ArrowLeft' || key === 'ArrowRight')) {
			const index = groups.indexOf(activeGroup) + (key === 'ArrowRight' ? 1 : -1);
			setSelectedTab(groups[(index + groups.length) % groups.length].name);
		} else if (
			!hasModifier &&
			key === 'Enter' &&
			typeof highlight === 'number' &&
			activeGroup?.items[highlight] != null
		) {
			const item = activeGroup.items[highlight];
			onSelect(event, item.term, activeGroup.type, item.displayValue);
			return;
		} else {
			if (
				['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Home', 'End', 'PageUp', 'PageDown'].includes(key)
			) {
				setKeyboardHighlight(null);
				setIsOpen(true);
				updateCursorPosition();
			}
			return;
		}
		event.preventDefault();
		event.stopPropagation();
	};

	const renderOptions = (group: SuggestionGroup) => (
		<div role='listbox' aria-label={group.label} ref={group === activeGroup ? optionsRef : undefined}>
			{group.items.map((item, index) => (
				<div
					key={item.term}
					className='combobox-list__item'
					role='option'
					aria-selected={group === activeGroup && highlight === index}
					onClick={(event) => onSelect(event, item.term, group.type, item.displayValue)}
				>
					{item.content}
				</div>
			))}
		</div>
	);

	const onTabClick = (e: any) => {
		inputRef.current.focus();
		setSelectedTab(e.detail.name);
	};

	return (
		<div
			className={`combobox${isOpen ? ' combobox--open' : ''}${isExpression ? ' combobox--expression' : ''}${caret ? ' combobox--caret' : ''}${className ? ` ${className}` : ''}`}
			data-id={name}
			style={
				{
					'--combobox-input-height':
						size === 'small'
							? '1.875rem'
							: size === 'medium'
								? '2.5rem'
								: size === 'large'
									? '3.125rem'
									: '0px',
					'--combobox-indentation': indentation != null ? `${indentation}px` : '0px',
				} as React.CSSProperties
			}
		>
			<div className='combobox-input'>
				<WaInput
					data-is-combobox-input
					onInput={disabled ? undefined : onInput}
					onKeyDown={onKeyDown}
					onClick={onClick}
					onBlur={onInputBlur}
					onFocus={onInputFocus}
					disabled={disabled}
					autocomplete='off'
					size={size === 'small' ? 's' : size === 'large' ? 'l' : 'm'}
					withClear={clearable}
					hint={helpText}
					label={label}
					ref={inputRef}
					{...rest}
				>
					{children}
					<WaIcon name='x-circle-fill' slot='clear-icon' />
					{error && val !== '' && (
						<WaIcon className='combobox-input__error-icon' name='exclamation-circle' slot='end'></WaIcon>
					)}
					{caret && <WaIcon className='combobox-input__caret-icon' name='chevron-down' slot='end'></WaIcon>}
				</WaInput>
				{error && (
					<div
						className={`combobox-input__error${isErrorExpanded ? ' combobox-input__error--expanded' : ''}`}
						ref={errorContainerRef}
					>
						{isErrorOverflowing && !isErrorExpanded && (
							<div className='combobox-input__error-overlay' onClick={() => setIsErrorExpanded(true)}>
								<WaIcon className='combobox-input__error-expand' name='three-dots' />
							</div>
						)}
						{error}
					</div>
				)}
			</div>
			{isListVisible && (
				<div
					data-is-combobox-list
					className={`combobox-list${isErrorExpanded ? ' combobox-list--expanded' : ''}`}
					ref={listRef}
					style={listWidth != null ? { width: `${listWidth}px` } : null}
				>
					<div className='combobox-list__body' ref={bodyRef}>
						{isExpression && selectedFunction != null && (
							<div className='combobox-list__function'>
								<div className='combobox-list__function-signature'>
									{selectedFunction.name}(
									{selectedFunction.params.map((p, i) => {
										let param = '';
										if (i > 0) {
											param += ', ';
										}
										return (
											<span
												key={p}
												className={`combobox-list__function-param${i === fragment.func.parameter ? ' combobox-list__function-param--current' : ''}`}
											>
												{param + p}
											</span>
										);
									})}
									): {selectedFunction.return}
								</div>
								<div className='combobox-list__function-description'>
									{selectedFunction.description}
								</div>
							</div>
						)}
						{hasTabs ? (
							<WaTabGroup
								className='combobox-list__tabs'
								active={activeGroup.name}
								onWaTabShow={onTabClick}
							>
								{groups.map((group) => (
									<WaTab
										key={group.name}
										slot='nav'
										panel={group.name}
										data-highlighted={
											highlight === 'tabs' && group === activeGroup ? '' : undefined
										}
									>
										{group.label} ({group.items.length})
									</WaTab>
								))}
								{groups.map((group) => (
									<WaTabPanel key={group.name} name={group.name}>
										{renderOptions(group)}
									</WaTabPanel>
								))}
							</WaTabGroup>
						) : (
							activeGroup != null && renderOptions(activeGroup)
						)}
					</div>
					{(hasTabNavigation || activeGroup?.items.length > 0) && (
						<ComboboxKeyboardHints
							hasTabNavigation={hasTabNavigation}
							highlight={highlight}
							hasOptions={activeGroup.items.length > 0}
						/>
					)}
				</div>
			)}
		</div>
	);
};

interface ComboboxState {
	filteredProperties: ComboboxItem[] | null;
	filteredFunctions: ComboboxItem[] | null;
}

const filterComboboxItems = (
	fragment: ExpressionFragment | null,
	cursorPosition: number,
	value: string,
	propertyItems: ComboboxItem[],
	functionItems: ComboboxItem[],
): ComboboxState => {
	if (
		cursorPosition === 0 ||
		value === '' ||
		(fragment != null && fragment.func == null && fragment.pos == null && fragment.type == null)
	) {
		// Every completion is relevant at the start of the value or of a new fragment.
		return { filteredProperties: propertyItems, filteredFunctions: functionItems };
	}

	if (fragment == null || (fragment.type !== 'Property' && fragment.type !== 'Function' && fragment.func == null)) {
		return { filteredProperties: null, filteredFunctions: null };
	}

	if (fragment.func != null && fragment.pos == null) {
		return { filteredProperties: propertyItems, filteredFunctions: functionItems };
	}

	let searchTerm = value.slice(fragment.pos.start, cursorPosition);
	if (searchTerm === '') {
		return { filteredProperties: propertyItems, filteredFunctions: functionItems };
	}

	const filteredProperties = filterItems(searchTerm, propertyItems);
	const filteredFunctions = filterItems(searchTerm, functionItems);

	return { filteredProperties: filteredProperties, filteredFunctions: filteredFunctions };
};

const filterItems = (searchTerm: string, items: ComboboxItem[]): ComboboxItem[] => {
	const filtered: ComboboxItem[] = [];
	for (const item of items) {
		if ([item.term, item.displayValue].some((term) => term != null && termMatches(term, searchTerm))) {
			filtered.push(item);
		}
	}

	filtered.sort((a, b) => {
		const aTerm = a.term;
		const bTerm = b.term;
		if (aTerm === searchTerm) return -1;
		if (bTerm === searchTerm) return 1;
		if (aTerm.startsWith(searchTerm) && !bTerm.startsWith(searchTerm)) return -1;
		else if (!aTerm.startsWith(searchTerm) && bTerm.startsWith(searchTerm)) return 1;
		return 0;
	});

	return filtered;
};

const termMatches = (term: string, searchTerm: string): boolean =>
	term.includes(searchTerm) ||
	term.includes(searchTerm.charAt(0).toUpperCase() + searchTerm.slice(1)) ||
	term.includes(searchTerm.toUpperCase()) ||
	term.includes(searchTerm.toLowerCase());

const getFunctionsComboboxItems = (functions: KrenalisFunction[]): ComboboxItem[] => {
	const functionItems: ComboboxItem[] = [];

	for (const func of functions) {
		functionItems.push({
			term: func.name,
			content: (
				<div className='function-item'>
					<div className='function-item__head'>
						<span className='function-item__head-name'>
							{func.name}
							<span className='function-item__head-params'>
								{`(${func.params.map((p, i) => {
									let param = '';
									if (i > 0) {
										param += ' ';
									}
									param += p;
									return param;
								})}): ${func.return}`}
							</span>
						</span>
					</div>
					<div className='function-item__description'> {func.description}</div>
				</div>
			),
		});
	}

	return functionItems;
};

export { Combobox };
