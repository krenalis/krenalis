import React, { ReactNode, useState, useMemo, useRef, useEffect, useLayoutEffect } from 'react';
import './Combobox.css';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlMenu from '@shoelace-style/shoelace/dist/react/menu/index.js';
import SlMenuItem from '@shoelace-style/shoelace/dist/react/menu-item/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlTabGroup from '@shoelace-style/shoelace/dist/react/tab-group/index.js';
import SlTabPanel from '@shoelace-style/shoelace/dist/react/tab-panel/index.js';
import SlTab from '@shoelace-style/shoelace/dist/react/tab/index.js';
import { ComboboxItem } from './Combobox.types';
import { ExpressionFragment, parseMapExpression } from '../../../utils/parseMapExpression';
import { autocompleteExpression } from './Combobox.helpers';
import { MEERGO_FUNCTIONS, MeergoFunction } from '../../../constants/function';
import { TransformedMapping } from '../../../lib/core/action';

interface ComboboxProps {
	value: string;
	items: ComboboxItem[];
	sharedMapping?: React.MutableRefObject<TransformedMapping>;
	onInput: (name: string, value: string) => void;
	onSelect: (name: string, value: string) => void;
	name: string;
	isExpression: boolean;
	size?: 'small' | 'medium' | 'large';
	className?: string;
	error?: string;
	caret?: boolean;
	controlled?: boolean;
	autoResize?: boolean;
	disabled?: boolean;
	children?: ReactNode;
	[key: string]: any;
}

// Combobox is a combobox component specifically designed to display and handle
// schema properties and expressions and is an uncontrolled component. The
// passed value is only used as the initial value, and any subsequent updates
// must be synced by the caller.
const Combobox = ({
	value,
	items,
	sharedMapping,
	onInput: onInputFunc,
	onSelect: onSelectFunc,
	name,
	isExpression,
	size = 'medium',
	className,
	error,
	caret = false,
	controlled = false,
	autoResize,
	disabled,
	children,
	...rest
}: ComboboxProps) => {
	const [val, setVal] = useState<string>(value == null ? '' : value);
	const [cursorPosition, setCursorPosition] = useState<number>();
	const [isOpen, setIsOpen] = useState<boolean>(false);
	const [listWidth, setListWidth] = useState<number>();
	const [selectedTab, setSelectedTab] = useState<string>();

	const inputRef = useRef<any>();
	const listRef = useRef<any>();
	const tabGroupRef = useRef<any>();

	const updateCursorPosition = (setStart?: boolean) => {
		const inputElement = inputRef.current?.input;
		if (inputElement) {
			setCursorPosition(setStart ? 0 : inputElement.selectionStart);
		}
	};

	const onKeyUp = (event: KeyboardEvent) => {
		if (
			['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Home', 'End', 'PageUp', 'PageDown'].includes(event.key)
		) {
			setIsOpen(true);
			updateCursorPosition();
		} else if (event.key === 'Escape') {
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
				input.addEventListener('keyup', onKeyUp);
				input.addEventListener('click', onClick);
				return () => {
					input.removeEventListener('keyup', onKeyUp);
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

	useEffect(() => {
		if (controlled) {
			setVal(value);
			if (autoResize) {
				// Resize the combobox after a delay to allow the shadow DOM to
				// fully load.
				setTimeout(() => {
					resizeCombobox();
				}, 50);
			}
		}
	}, [value]);

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

	useEffect(() => {
		if (sharedMapping?.current) {
			setVal(sharedMapping.current[name]?.value || '');
		}
	}, [sharedMapping?.current[name]?.value]);

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

	const onInput = (e) => {
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

		setVal(newValue);

		setTimeout(() => {
			inputRef.current.setSelectionRange(position, position);
			updateCursorPosition();
		});
		onInputFunc(name, newValue);
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
		const wrapper = inputRef.current.closest('.combobox');
		wrapper.style.width = `${textWidth + 50}px`;
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
			((filteredProperties != null && filteredProperties.length > 0) ||
				(filteredFunctions != null && filteredFunctions.length > 0)) &&
			isExpression
		);
	}, [filteredProperties, filteredFunctions]);

	useEffect(() => {
		if (hasTabs) {
			// set the initial value of the selected tab.
			setSelectedTab('properties');
		}
	}, []);

	const onSelect = (e, term: string, type: 'property' | 'function') => {
		e.preventDefault();
		e.stopPropagation();

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

		setVal(v);

		inputRef.current.focus();
		setTimeout(() => {
			if (autoResize) {
				resizeCombobox();
			}
			inputRef.current.setSelectionRange(position, position);
			updateCursorPosition();
		});
		onSelectFunc(name, v);
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
				} as React.CSSProperties
			}
		>
			<div className='combobox-input'>
				<SlInput
					data-is-combobox-input
					value={val}
					onSlInput={disabled ? undefined : onInput}
					onSlBlur={onInputBlur}
					disabled={disabled}
					autocomplete='off'
					size={size}
					ref={inputRef}
					{...rest}
				>
					{children}
					{error && val !== '' && (
						<SlIcon className='combobox-input__error-icon' name='exclamation-circle' slot='prefix'></SlIcon>
					)}
					{caret && (
						<SlIcon className='combobox-input__caret-icon' name='chevron-down' slot='suffix'></SlIcon>
					)}
				</SlInput>
				{error && <div className='combobox-input__error'>{error}</div>}
			</div>
			{isOpen && (
				<SlMenu
					data-is-combobox-list
					className='combobox-list'
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
							<div className='combobox-list__function-description'>{selectedFunction.description}</div>
						</div>
					)}
					{hasTabs ? (
						<SlTabGroup className='combobox-list__tabs' onSlTabShow={onTabClick} ref={tabGroupRef}>
							<SlTab slot='nav' panel='properties'>
								Properties ({filteredProperties.length})
							</SlTab>
							<SlTab slot='nav' panel='functions'>
								Functions ({filteredFunctions.length})
							</SlTab>
							<SlTabPanel name='properties'>
								{filteredProperties?.map((item) => {
									return (
										<SlMenuItem key={item.term} onClick={(e) => onSelect(e, item.term, 'property')}>
											{item.content}
										</SlMenuItem>
									);
								})}
							</SlTabPanel>
							<SlTabPanel name='functions'>
								{filteredFunctions?.map((item) => {
									return (
										<SlMenuItem key={item.term} onClick={(e) => onSelect(e, item.term, 'function')}>
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
