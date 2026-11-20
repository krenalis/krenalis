import React, { ReactNode, useState, useMemo, useRef, useEffect, useLayoutEffect, useContext } from 'react';
import { flushSync } from 'react-dom';
import './Combobox.css';
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
	const [menuMode, setMenuMode] = useState<'closed' | 'filtered' | 'all'>('closed');
	const [listWidth, setListWidth] = useState<number>();
	const [selectedTab, setSelectedTab] = useState<string>();
	const [keyboardHighlight, setKeyboardHighlight] = useState<HighlightedOption | null>(null);
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

	const isOpen = menuMode !== 'closed';

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

	const updateCursorPosition = () => {
		setTimeout(() => {
			const inputElement = inputRef.current?.input;
			if (inputElement) {
				setCursorPosition(inputElement.selectionStart);
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
		setMenuMode(isExpression ? 'filtered' : 'all');
		updateCursorPosition();
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
				setMenuMode('all');
				updateCursorPosition();
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
	}, [items, cursorPosition, menuMode, selectedTab, enumValues]);

	useEffect(() => {
		const onPageClick = (e) => {
			const target = e.target;
			const isInCombobox = target.closest('.combobox') != null;
			const isInAnotherCombobox = target.closest('.combobox')?.dataset.id !== name;
			if (!isInCombobox || isInAnotherCombobox) {
				setMenuMode('closed');
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
		setMenuMode('filtered');

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
		setCursorPosition(position);
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

	const onBlur = (event: React.FocusEvent<HTMLDivElement>) => {
		if (event.currentTarget.contains(event.relatedTarget)) {
			return;
		}
		setKeyboardHighlight(null);
		setSelectedTab(undefined);
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
		setMenuMode('filtered');
		updateCursorPosition();
	};

	let functionItems = useMemo(() => {
		if (!isExpression) {
			return [];
		}
		return getFunctionsComboboxItems(KRENALIS_FUNCTIONS);
	}, [isExpression]);

	let fragment = useMemo(() => {
		return parseMapExpression(val == null ? '' : val, cursorPosition);
	}, [val, cursorPosition]);

	let { filteredProperties, filteredFunctions } = useMemo(() => {
		return filterComboboxItems(fragment, cursorPosition, val, items, functionItems);
	}, [fragment, cursorPosition, val, items, functionItems]);

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
	const getGroups = (showAll: boolean) => {
		const properties = showAll ? items : filteredProperties;
		const functions = showAll ? functionItems : filteredFunctions;
		const groups: SuggestionGroup[] = [];
		if (enumItems != null) {
			groups.push({ name: 'enum', label: 'Enum', type: 'enum', items: enumItems });
		}
		if (properties != null) {
			groups.push({ name: 'properties', label: 'Properties', type: 'property', items: properties });
		}
		const hasOptions = properties?.length > 0 || functions?.length > 0;
		if (isExpression && functions != null && (enumItems != null || hasOptions)) {
			groups.push({ name: 'functions', label: 'Functions', type: 'function', items: functions });
		}
		return groups;
	};
	const groups = getGroups(menuMode === 'all');
	const hasTabs = groups.some((group) => group.type !== 'property');
	const activeGroup = groups.find((group) => group.name === selectedTab) ?? groups[0];
	const isListVisible = isOpen && (selectedFunction != null || hasTabs || activeGroup?.items.length > 0);

	let highlight: number | null = null;
	if (keyboardHighlight != null && activeGroup?.name === keyboardHighlight.group) {
		const index = activeGroup.items.findIndex((item) => item.term === keyboardHighlight.term);
		highlight = index >= 0 ? index : null;
	}

	useLayoutEffect(() => {
		if (!isOpen) {
			setKeyboardHighlight(null);
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
		const target = highlight != null ? options?.children[highlight] : null;
		// Element references link the native input in the shadow root to the visible suggestions.
		input.ariaExpanded = String(isListVisible);
		input.ariaControlsElements = [options, tab?.parentElement].filter((element) => element != null);
		input.ariaActiveDescendantElement = target ?? null;
		if (body == null || target == null) {
			return;
		}
		const bounds = body.getBoundingClientRect();
		const top = bounds.top + (tab?.offsetHeight ?? 0);
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

	const onSelect = (e, term: string, type: 'property' | 'function' | 'enum') => {
		e.preventDefault();
		e.stopPropagation();
		setKeyboardHighlight(null);
		isEditing.current = true;

		if (type === 'enum' || !isExpression) {
			updateComboboxValue(term);
			programmaticFocus.current = true;
			inputRef.current.focus();
			onSelectFunc(name, term);
			validate(name, term);
			setMenuMode('closed');
			return;
		}

		const { selectionStart, selectionEnd } = inputRef.current.input;
		const insertionFragment = parseMapExpression(val, selectionStart);
		const start = insertionFragment?.pos?.start ?? selectionStart;
		let end = insertionFragment?.pos?.end ?? selectionEnd;
		if (insertionFragment?.type === 'String') {
			end++; // Include the closing quote.
		}
		let v = val.slice(0, start) + term + val.slice(end);
		let position = start + term.length;

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
		if (type === 'property' && insertionFragment?.func == null) {
			// if a property has been selected and is not an argument for a
			// function, the combobox list can be closed.
			setMenuMode('closed');
		} else {
			setMenuMode('filtered');
			if (type === 'function') {
				// Show properties to pass as arguments to the function.
				setSelectedTab('properties');
			}
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

	const highlightOption = (group: SuggestionGroup, index: number) => {
		setSelectedTab(group.name);
		setKeyboardHighlight({ group: group.name, term: group.items[index].term });
	};

	const onKeyDown = (event: React.KeyboardEvent) => {
		const isOnTab = event.target instanceof HTMLElement && event.target.closest('wa-tab') != null;
		const isOnInput = event.nativeEvent.composedPath()[0] === inputRef.current?.input;
		if ((!isOnTab && !isOnInput) || event.nativeEvent.isComposing) {
			return;
		}
		const { key } = event;
		if (key === 'Escape' || key === 'Tab') {
			if (isOnTab) {
				programmaticFocus.current = true;
				inputRef.current.focus();
			}
			// Remove the menu tabs before the browser advances focus.
			flushSync(() => setMenuMode('closed'));
			if (key === 'Escape' && isListVisible) {
				event.preventDefault();
				event.stopPropagation();
			}
			return;
		}
		const hasModifier = event.altKey || event.ctrlKey || event.metaKey || event.shiftKey;
		if (isOnTab) {
			if (hasModifier || (key !== 'ArrowDown' && key !== 'ArrowUp')) {
				return;
			}
			if (key === 'ArrowUp' || activeGroup.items.length > 0) {
				programmaticFocus.current = true;
				inputRef.current.focus();
				if (key === 'ArrowDown') {
					highlightOption(activeGroup, 0);
				}
			}
		} else if (!hasModifier && (key === 'ArrowDown' || key === 'ArrowUp')) {
			let available = groups.filter((group) => group.items.length > 0);
			if (key === 'ArrowDown' && (!isListVisible || available.length === 0)) {
				available = getGroups(true).filter((group) => group.items.length > 0);
				setMenuMode('all');
			} else if (!isOpen) {
				setMenuMode('filtered');
			}
			const group = available.find((group) => group.name === selectedTab) ?? available[0];
			if (group != null) {
				const count = group.items.length;
				const index = key === 'ArrowDown' ? (highlight ?? -1) + 1 : (highlight ?? 0) - 1;
				highlightOption(group, (index + count) % count);
			} else if (key === 'ArrowUp') {
				updateCursorPosition();
				return;
			}
		} else if (!hasModifier && highlight != null && (key === 'ArrowLeft' || key === 'ArrowRight')) {
			const available = groups.filter((group) => group.items.length > 0);
			if (available.length > 1) {
				const index = available.indexOf(activeGroup) + (key === 'ArrowRight' ? 1 : -1);
				highlightOption(available[(index + available.length) % available.length], 0);
			}
		} else if (!hasModifier && key === 'Enter' && highlight != null) {
			const item = activeGroup.items[highlight];
			onSelect(event, item.term, activeGroup.type);
			return;
		} else {
			if (
				['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Home', 'End', 'PageUp', 'PageDown'].includes(key)
			) {
				setKeyboardHighlight(null);
				setMenuMode('filtered');
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
					onMouseDown={(event) => event.preventDefault()}
					onClick={(event) => onSelect(event, item.term, group.type)}
				>
					{item.content}
				</div>
			))}
		</div>
	);

	return (
		<div
			className={`combobox${isOpen ? ' combobox--open' : ''}${isExpression ? ' combobox--expression' : ''}${caret ? ' combobox--caret' : ''}${className ? ` ${className}` : ''}`}
			data-id={name}
			onBlur={onBlur}
			onKeyDown={onKeyDown}
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
					onClick={onClick}
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
								onWaTabShow={(event) => setSelectedTab(event.detail.name)}
							>
								{groups.map((group) => (
									<WaTab
										key={group.name}
										slot='nav'
										panel={group.name}
										onFocus={() => setKeyboardHighlight(null)}
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
