import React, { ReactNode, useState, useMemo, useRef, useEffect } from 'react';
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

interface ComboboxProps {
	initialValue: string;
	items: ComboboxItem[];
	onInput: (name: string, value: string) => void;
	onSelect: (name: string, value: string) => void;
	name: string;
	isExpression: boolean;
	className?: string;
	error?: string;
	caret?: boolean;
	disabled?: boolean;
	children?: ReactNode;
	[key: string]: any;
}

// Combobox is a combobox component specifically designed to display and handle
// schema properties and expressions and is an uncontrolled component. The
// passed value is only used as the initial value, and any subsequent updates
// must be synced by the caller.
const Combobox = ({
	initialValue,
	items,
	onInput: onInputFunc,
	onSelect: onSelectFunc,
	name,
	isExpression,
	className,
	error,
	caret,
	disabled,
	children,
	...rest
}: ComboboxProps) => {
	const [value, setValue] = useState<string>(initialValue);
	const [cursorPosition, setCursorPosition] = useState<number>();
	const [isOpen, setIsOpen] = useState<boolean>(false);

	const inputRef = useRef<any>();
	const listRef = useRef<any>();

	const updateCursorPosition = () => {
		const inputElement = inputRef.current?.input;
		if (inputElement) {
			setCursorPosition(inputElement.selectionStart);
		}
	};

	const onKeyUp = (event: KeyboardEvent) => {
		if (
			['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Home', 'End', 'PageUp', 'PageDown'].includes(event.key)
		) {
			updateCursorPosition();
		} else if (event.key === 'Escape') {
			setIsOpen(false);
		}
	};

	const onClick = () => {
		setIsOpen(true);
		updateCursorPosition();
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

	useEffect(() => {}, [value, cursorPosition, items]);

	const onInput = (e) => {
		if (!isOpen) {
			// if the user has closed the list via escape button.
			setIsOpen(true);
		}

		let lastValue = value;
		let newValue = e.target.value;
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

		setValue(newValue);
		setTimeout(() => {
			inputRef.current.setSelectionRange(position, position);
			updateCursorPosition();
		});
		onInputFunc(name, newValue);
	};

	let functionItems = useMemo(() => {
		if (!isExpression) {
			return [];
		}
		return getFunctionsComboboxItems(MEERGO_FUNCTIONS);
	}, []);

	let fragment = useMemo(() => {
		return parseMapExpression(value, cursorPosition);
	}, [value, cursorPosition, items]);

	let { filteredProperties, filteredFunctions } = useMemo(() => {
		return filterComboboxItems(fragment, cursorPosition, value, items, isExpression ? functionItems : []);
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

	const onSelect = (e, term: string, type: 'property' | 'function') => {
		e.preventDefault();
		e.stopPropagation();

		let position = 0;
		let val = '';
		if (fragment.func != null && fragment.pos == null) {
			const expressionStart = value.slice(0, cursorPosition);
			const expressionEnd = value.slice(cursorPosition);
			val = `${expressionStart}${term}${expressionEnd}`;
			position = cursorPosition + term.length;
		} else if (fragment != null && fragment.pos != null) {
			const expressionStart = value.slice(0, fragment.pos.start);
			const expressionEnd = value.slice(fragment.pos.end);
			val = `${expressionStart}${term}${expressionEnd}`;
			if (value === '') {
				position = term.length;
			} else {
				position = fragment.pos.start + term.length;
			}
		} else {
			val = value + term;
			position = value.length + term.length;
		}

		if (type === 'function') {
			// add parenthesis if necessary.
			const expressionStart = val.slice(0, position);
			const expressionEnd = val.slice(position);
			const hasAlreadyParenthesis = val[position] === '(';
			if (!hasAlreadyParenthesis) {
				val = `${expressionStart}()${expressionEnd}`;
				position += 1;
			}
		} else if (type === 'property') {
			// remove parenthesis if necessary.
			if (val[position] === '(' && val[position + 1] === ')') {
				const expressionStart = val.slice(0, position);
				const expressionEnd = val.slice(position + 2);
				val = `${expressionStart}${expressionEnd}`;
			}
		}

		setValue(val);
		inputRef.current.focus();
		setTimeout(() => {
			inputRef.current.setSelectionRange(position, position);
			updateCursorPosition();
		});
		onSelectFunc(name, val);
	};

	const onTabClick = () => {
		inputRef.current.focus();
	};

	return (
		<div
			className={`combobox${isOpen ? ' combobox--open' : ''}${isExpression ? ' combobox--expression' : ''}`}
			data-id={name}
		>
			<div className='combobox-input'>
				<SlInput
					data-is-combobox-input
					value={value}
					onSlInput={disabled ? undefined : onInput}
					className={className}
					disabled={disabled}
					autocomplete='off'
					ref={inputRef}
					{...rest}
				>
					{children}
					{error && value !== '' && (
						<SlIcon className='combobox-input__error-icon' name='exclamation-circle' slot='prefix'></SlIcon>
					)}
					{caret && (
						<SlIcon className='combobox-input__caret-icon' name='chevron-down' slot='suffix'></SlIcon>
					)}
				</SlInput>
				{error && <div className='combobox-input__error'>{error}</div>}
			</div>
			{isOpen && (
				<SlMenu data-is-combobox-list className='combobox-list' ref={listRef}>
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
					{((filteredProperties != null && filteredProperties.length > 0) ||
						(filteredFunctions != null && filteredFunctions.length > 0)) &&
					isExpression ? (
						<SlTabGroup className='combobox-list__tabs' onSlTabShow={onTabClick}>
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
