import React, { ReactNode, useState, useMemo, useRef, useEffect, useLayoutEffect, useContext } from 'react';
import './Combobox.css';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlMenu from '@shoelace-style/shoelace/dist/react/menu/index.js';
import SlMenuItem from '@shoelace-style/shoelace/dist/react/menu-item/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlTabGroup from '@shoelace-style/shoelace/dist/react/tab-group/index.js';
import SlTabPanel from '@shoelace-style/shoelace/dist/react/tab-panel/index.js';
import SlTab from '@shoelace-style/shoelace/dist/react/tab/index.js';
import { ComboboxItem } from './Combobox.types';
import { ExpressionFragment, parseMapExpression } from '../../../utils/mapExpression';
import { autocompleteExpression } from './Combobox.helpers';
import { MEERGO_FUNCTIONS, MeergoFunction } from '../../../constants/function';
import appContext from '../../../context/AppContext';
import { debounceWithAbort } from '../../../utils/debounce';
import ConnectionContext from '../../../context/ConnectionContext';
import pipelineContext from '../../../context/PipelineContext';
import Type from '../../../lib/api/types/types';

const CONSTANT_REGEX = /"([^"]*)"/;

interface ComboboxProps {
	value: string;
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
	controlled?: boolean;
	autoResize?: boolean;
	disabled?: boolean;
	indentation?: number;
	children?: ReactNode;
	syncOnChange?: any;
	propertiesToHide?: string[] | null;
	[key: string]: any;
}

// Combobox is a combobox component specifically designed to display and handle
// schema properties and expressions.
const Combobox = ({
	value,
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
	controlled = false,
	autoResize,
	disabled,
	indentation,
	children,
	syncOnChange,
	propertiesToHide,
	...rest
}: ComboboxProps) => {
	const [val, setVal] = useState<string>(value == null ? '' : value);
	const [cursorPosition, setCursorPosition] = useState<number>();
	const [isOpen, setIsOpen] = useState<boolean>(false);
	const [listWidth, setListWidth] = useState<number>();
	const [selectedTab, setSelectedTab] = useState<string>();
	const [isErrorExpanded, setIsErrorExpanded] = useState<boolean>(false);
	const [isErrorOverflowing, setIsErrorOverflowing] = useState<boolean>(false);

	const inputRef = useRef<any>();
	const isFirstControl = useRef<any>(true);
	const errorContainerRef = useRef<any>();
	const listRef = useRef<any>();
	const tabGroupRef = useRef<any>();
	const programmaticFocus = useRef(false);

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

	const onKeyDown = (event: KeyboardEvent) => {
		if (
			['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Home', 'End', 'PageUp', 'PageDown'].includes(event.key)
		) {
			setIsOpen(true);
			updateCursorPosition();
		} else if (event.key === 'Escape' || event.key === 'Tab') {
			setIsOpen(false);
		}
	};

	const onClick = () => {
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
		setTimeout(() => {
			const input = inputRef.current.shadowRoot?.querySelector('input');
			if (input) {
				// Disable password managers autofill on the input.
				input.setAttribute('data-1p-ignore', '');
				input.setAttribute('data-bwignore', '');
				input.setAttribute('data-form-type', 'other');
				input.setAttribute('data-lpignore', 'true');

				// Add event listeners on the input.
				input.addEventListener('keydown', onKeyDown);
				input.addEventListener('click', onClick);
				return () => {
					input.removeEventListener('keydown', onKeyDown);
					input.removeEventListener('click', onClick);
				};
			}
		});
	}, [inputRef.current]);

	useEffect(() => {
		if (inputRef.current == null || !caret) {
			return;
		}

		setTimeout(() => {
			const inputSuffix = inputRef.current.shadowRoot?.querySelector('[part="suffix"]');
			inputSuffix.addEventListener('click', () => {
				inputRef.current.focus();
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
			if (isFirstControl.current) {
				// Defer update to next event loop tick to ensure the input is
				// mounted before updating its value.
				setTimeout(() => {
					updateComboboxValue(value);
					isFirstControl.current = false;
				});
			} else {
				updateComboboxValue(value);
			}
		}
		if (autoResize) {
			// Resize the combobox with a short delay. The combobox is resized
			// when the value changes or when the error changes (to take into
			// consideration the error icon shown in the suffix slot of the
			// input).
			setTimeout(() => {
				resizeCombobox();
			}, 50);
		}
	}, [value, error]);

	useEffect(() => {
		if (!isOpen || listRef.current == null) {
			return;
		}
		setTimeout(() => {
			// See if the menu is overflowing the width of the input and
			// in that case expand it to avoid horizontal scrollbars.
			const menu = listRef.current;
			if (menu == null) {
				return;
			}

			let menuItems: any[];
			if (selectedTab == null) {
				menuItems = menu.querySelectorAll('sl-menu-item');
			} else {
				const panel = menu.querySelector(`sl-tab-panel[name="${selectedTab}"]`);
				menuItems = panel.querySelectorAll('sl-menu-item');
			}

			// Get the width of the wider menu item.
			let maxWidth = 0;
			for (const item of menuItems) {
				let w: number;
				if (selectedTab == null || selectedTab == 'properties') {
					const iconWidth = item.querySelector('.schema-combobox-item__type').offsetWidth;
					const textWidth = item.querySelector('.schema-combobox-item__text').offsetWidth;
					w = iconWidth + textWidth;
				} else if (selectedTab === 'functions') {
					const itemWidth = item.querySelector('.function-item').offsetWidth;
					w = itemWidth;
				}
				if (w > maxWidth) {
					maxWidth = w;
				}
			}

			// Get the width of the tabs.
			const tabs = menu.querySelectorAll('sl-tab');
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
	}, [items, isOpen, selectedTab]);

	useEffect(() => {
		if (autoResize) {
			// Resize the combobox after a delay to allow the shadow DOM to
			// fully load.
			setTimeout(() => {
				resizeCombobox();
			}, 50);
		}
	}, []);

	useLayoutEffect(() => {
		if (listRef.current == null || !isOpen) {
			return;
		}
		// Check if the combobox list vertically overflows the viewport
		// and eventually position it on the border top of the combobox
		// input.
		listRef.current.classList.add('combobox-list--computing-position');
		setTimeout(() => {
			const rect = listRef.current.getBoundingClientRect();
			listRef.current.classList.toggle('combobox-list--top', rect.bottom > window.innerHeight);
			listRef.current.classList.remove('combobox-list--computing-position');
		}, 20);
	}, [isOpen]);

	useLayoutEffect(() => {
		if (enumValues == null) {
			return;
		}
		setTimeout(() => {
			const isConstant = CONSTANT_REGEX.test(val);
			if (val === '' || isConstant) {
				tabGroupRef.current?.show('enum');
			}
		});
	}, [isOpen]);

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
		const isAppEventsExport = connection.isAPI && connection.isDestination && pipelineType.target === 'Event';
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

		const prefix = inputRef.current.shadowRoot.querySelector('span[part="prefix"]');
		const prefixWidth = prefix?.offsetWidth || 0;
		const suffix = inputRef.current.shadowRoot.querySelector('span[part="suffix"]');
		const suffixWidth = suffix?.offsetWidth || 0;

		const wrapper = inputRef.current.closest('.combobox');
		wrapper.style.width = `${textWidth + prefixWidth + suffixWidth + 30}px`;
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
		setIsOpen(true);
		updateCursorPosition();
	};

	const onInputBlur = () => {
		if (autoResize) {
			resizeCombobox();
		}
	};

	let functionItems = useMemo(() => {
		if (!isExpression) {
			return [];
		}
		return getFunctionsComboboxItems(MEERGO_FUNCTIONS);
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
			return MEERGO_FUNCTIONS.find((f) => f.name === functionName);
		}
		return null;
	}, [fragment]);

	const hasTabs = useMemo(() => {
		return (
			enumValues != null ||
			(((filteredProperties != null && filteredProperties.length > 0) ||
				(filteredFunctions != null && filteredFunctions.length > 0)) &&
				isExpression)
		);
	}, [filteredProperties, filteredFunctions]);

	useEffect(() => {
		if (enumValues != null) {
			setSelectedTab('enum');
		} else if (hasTabs) {
			// set the initial value of the selected tab.
			setSelectedTab('properties');
		}
	}, []);

	const onSelect = (e, term: string, type: 'property' | 'function' | 'enum') => {
		e.preventDefault();
		e.stopPropagation();

		if (type === 'enum') {
			let v = term;
			updateComboboxValue(v);
			programmaticFocus.current = true;
			inputRef.current.focus();
			setTimeout(() => {
				if (autoResize) {
					resizeCombobox();
				}
			});
			onSelectFunc(name, v);
			validate(name, v);
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
			tabGroupRef.current.show('properties');
		}

		updateComboboxValue(v);

		programmaticFocus.current = true;
		inputRef.current.focus();
		setTimeout(() => {
			if (autoResize) {
				resizeCombobox();
			}
			inputRef.current.setSelectionRange(position, position);
			updateCursorPosition();
		});
		onSelectFunc(name, v);
		validate(name, v);
	};

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
				<SlInput
					data-is-combobox-input
					onSlInput={disabled ? undefined : onInput}
					onSlFocus={onInputFocus}
					onSlBlur={onInputBlur}
					disabled={disabled}
					autocomplete='off'
					size={size}
					ref={inputRef}
					{...rest}
				>
					{children}
					{error && val !== '' && (
						<SlIcon className='combobox-input__error-icon' name='exclamation-circle' slot='suffix'></SlIcon>
					)}
					{caret && (
						<SlIcon className='combobox-input__caret-icon' name='chevron-down' slot='suffix'></SlIcon>
					)}
				</SlInput>
				{error && (
					<div
						className={`combobox-input__error${isErrorExpanded ? ' combobox-input__error--expanded' : ''}`}
						ref={errorContainerRef}
					>
						{isErrorOverflowing && !isErrorExpanded && (
							<div className='combobox-input__error-overlay' onClick={() => setIsErrorExpanded(true)}>
								<SlIcon className='combobox-input__error-expand' name='three-dots' />
							</div>
						)}
						{error}
					</div>
				)}
			</div>
			{isOpen &&
				((isExpression && selectedFunction != null) ||
					enumValues != null ||
					filteredProperties?.length > 0 ||
					filteredFunctions?.length > 0) && (
					<SlMenu
						tabIndex={-1}
						data-is-combobox-list
						className={`combobox-list${isErrorExpanded ? ' combobox-list--expanded' : ''}`}
						ref={listRef}
						style={listWidth != null ? { width: `${listWidth}px` } : null}
					>
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
							<SlTabGroup className='combobox-list__tabs' onSlTabShow={onTabClick} ref={tabGroupRef}>
								{enumValues != null && (
									<SlTab slot='nav' panel='enum'>
										Enum ({enumValues.length})
									</SlTab>
								)}
								{filteredProperties && (
									<SlTab slot='nav' panel='properties'>
										Properties ({filteredProperties.length})
									</SlTab>
								)}
								{filteredFunctions && (
									<SlTab slot='nav' panel='functions'>
										Functions ({filteredFunctions.length})
									</SlTab>
								)}
								{enumValues != null && (
									<SlTabPanel name='enum'>
										{enumValues.map((v) => {
											return (
												<SlMenuItem key={v} onClick={(e) => onSelect(e, v, 'enum')}>
													<div className='enum-item'>{v}</div>
												</SlMenuItem>
											);
										})}
									</SlTabPanel>
								)}
								<SlTabPanel name='properties'>
									{filteredProperties?.map((item) => {
										return (
											<SlMenuItem
												key={item.term}
												onClick={(e) => onSelect(e, item.term, 'property')}
											>
												{item.content}
											</SlMenuItem>
										);
									})}
								</SlTabPanel>
								<SlTabPanel name='functions'>
									{filteredFunctions?.map((item) => {
										return (
											<SlMenuItem
												key={item.term}
												onClick={(e) => onSelect(e, item.term, 'function')}
											>
												{item.content}
											</SlMenuItem>
										);
									})}
								</SlTabPanel>
							</SlTabGroup>
						) : (
							<div>
								{filteredProperties?.map((item) => {
									return (
										<SlMenuItem key={item.term} onClick={(e) => onSelect(e, item.term, 'property')}>
											{item.content}
										</SlMenuItem>
									);
								})}
							</div>
						)}
					</SlMenu>
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
	if (value === '' || (fragment != null && fragment.func == null && fragment.pos == null && fragment.type == null)) {
		// the user is at the start of a new fragment.
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
		const term = item.term;
		if (
			term.includes(searchTerm) ||
			term.includes(searchTerm.charAt(0).toUpperCase() + searchTerm.slice(1)) ||
			term.includes(searchTerm.toUpperCase()) ||
			term.includes(searchTerm.toLowerCase())
		) {
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

const getFunctionsComboboxItems = (functions: MeergoFunction[]): ComboboxItem[] => {
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
