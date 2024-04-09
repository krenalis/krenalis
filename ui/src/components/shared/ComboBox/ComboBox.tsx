import React, { forwardRef, useRef, useEffect, useState, useImperativeHandle, ReactNode } from 'react';
import './ComboBox.css';
import { debounce } from '../../../lib/utils/debounce';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlMenu from '@shoelace-style/shoelace/dist/react/menu/index.js';
import SlMenuItem from '@shoelace-style/shoelace/dist/react/menu-item/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { ComboboxItem, Size } from '../../../types/internal/app';
import { autocompleteExpression } from './ComboBox.helpers';

interface ComboBoxListProps {
	items: ComboboxItem[];
	onSelect: (currentComboboxInput: ReactNode, term: string) => void;
}

interface ComboboxListMethods {
	open: () => void;
	close: () => void;
	updateSearchTerm: (term: string) => void;
	updateCurrentComboboxInput: (input: ReactNode) => void;
}

type ComboBoxListRef = ComboboxListMethods & any;

const ComboBoxList = forwardRef<ComboBoxListRef, ComboBoxListProps>(({ items, onSelect }, ref) => {
	const [isOpen, setIsOpen] = useState<boolean>(false);
	const [searchTerm, setSearchTerm] = useState<string>('');
	const [currentComboboxInput, setCurrentComboboxInput] = useState<ReactNode>();

	const comboBoxListMenuRef = useRef<ComboBoxListRef>();

	useImperativeHandle(
		ref,
		() => {
			return {
				...comboBoxListMenuRef.current,
				open() {
					setIsOpen(true);
				},
				close() {
					setIsOpen(false);
				},
				updateSearchTerm(term: string) {
					setSearchTerm(term);
				},
				updateCurrentComboboxInput(input: ReactNode) {
					setCurrentComboboxInput(input);
				},
			};
		},
		[],
	);

	const onMouseDown = (e: React.MouseEvent) => {
		// prevent ComboBoxInput from losing focus
		const activeElement = document.activeElement;
		if (activeElement && activeElement instanceof HTMLElement && activeElement.dataset.isComboboxInput) {
			e.preventDefault();
		}
	};

	const searchResults: ComboboxItem[] = [];
	for (const item of items) {
		const term = item.term;
		if (
			term.includes(searchTerm) ||
			term.includes(searchTerm.charAt(0).toUpperCase() + searchTerm.slice(1)) ||
			term.includes(searchTerm.toUpperCase()) ||
			term.includes(searchTerm.toLowerCase())
		) {
			searchResults.push(item);
		}
	}

	searchResults.sort((a, b) => {
		const aTerm = a.term;
		const bTerm = b.term;
		if (aTerm === searchTerm) return -1;
		if (bTerm === searchTerm) return 1;
		if (aTerm.startsWith(searchTerm) && !bTerm.startsWith(searchTerm)) return -1;
		else if (!aTerm.startsWith(searchTerm) && bTerm.startsWith(searchTerm)) return 1;
		return 0;
	});

	return (
		<SlMenu
			tabIndex={-1} // menu items must be selected only via "ArrowDown" key. "Tab" press must instead focus the next input.
			ref={comboBoxListMenuRef}
			data-is-combobox-list
			className='comboBoxList'
			data-isOpen={isOpen}
			onMouseDown={onMouseDown}
		>
			{searchResults.map((item) => {
				return (
					<SlMenuItem
						key={item.term}
						onClick={() => {
							setSearchTerm(item.term);
							onSelect(currentComboboxInput, item.term);
							setIsOpen(false);
						}}
					>
						{item.content}
					</SlMenuItem>
				);
			})}
		</SlMenu>
	);
});

interface ComboBoxInputProps {
	comboBoxListRef: React.RefObject<ComboBoxListRef>;
	onInput: (...args: any) => void;
	value: string;
	name?: string;
	label?: string;
	className?: string;
	children?: ReactNode;
	error?: string;
	size?: Size;
	disabled?: boolean;
	readonly?: boolean;
	autocompleteExpressions?: boolean;
	caret?: boolean;
}

const ComboBoxInput = ({
	comboBoxListRef,
	value,
	name,
	label,
	className,
	onInput: onInputProp,
	children,
	error,
	size = 'medium',
	disabled,
	readonly,
	autocompleteExpressions,
	caret,
	...delegated
}: ComboBoxInputProps) => {
	const onKeyUpRef = useRef<any>();
	const previousListSiblingRef = useRef<any>();
	const lastValue = useRef<string>(value);

	useEffect(() => {
		lastValue.current = value;
	}, [value]);

	const onKeyUp = (e) => {
		if (e.key === 'Escape') {
			onInputBlur();
		}
		if (e.key === 'ArrowDown') {
			const comboboxListShadowRoot = comboBoxListRef.current!.renderRoot as ShadowRoot;
			const menuItems: any = comboboxListShadowRoot.host.querySelectorAll('sl-menu-item');
			if (menuItems.length > 0) {
				menuItems[0].focus();
			}
		}
	};

	const onInputFocus = (e) => {
		window.addEventListener('keyup', onKeyUpRef.current!);
		const input = e.target;
		comboBoxListRef.current!.updateCurrentComboboxInput(input);
		comboBoxListRef.current!.updateSearchTerm('');
		setTimeout(() => {
			input.before(comboBoxListRef.current!.renderRoot.host);
			comboBoxListRef.current!.open();
		});
	};

	const onInputBlur = () => {
		window.removeEventListener('keyup', onKeyUpRef.current!);
		function onBlur() {
			const parentComboboxList = document.activeElement?.closest('[data-is-combobox-list]');
			if (parentComboboxList) {
				document.activeElement.addEventListener('focusout', () => setTimeout(() => onBlur()));
			} else {
				comboBoxListRef.current!.close();
				previousListSiblingRef.current!.after(comboBoxListRef.current!.renderRoot.host);
			}
		}
		setTimeout(() => {
			onBlur();
		});
	};

	const onClick = (e) => {
		const input = e.target;
		input?.focus();
	};

	const onInput = (e) => {
		let newValue = e.target.value;
		if (autocompleteExpressions) {
			const isPasted = Math.abs(lastValue.current.length - newValue.length) > 1;
			const isBackspaced = lastValue.current.length > newValue.length;
			const isEqual = lastValue.current.length === newValue.length;
			if (!isPasted && !isBackspaced && !isEqual) {
				const cursorPosition = e.target.shadowRoot.querySelector('input').selectionStart;
				const autocompleted = autocompleteExpression(newValue, cursorPosition);
				if (autocompleted != null) {
					newValue = autocompleted;
					e.target.value = autocompleted;
					setTimeout(() => {
						e.target.setSelectionRange(cursorPosition, cursorPosition);
					});
				}
			}
		}
		lastValue.current = newValue;
		comboBoxListRef.current!.updateSearchTerm(newValue);
		onInputProp(e);
	};

	const debouncedOnInput = debounce(onInput, 0);

	useEffect(() => {
		onKeyUpRef.current = onKeyUp;
		previousListSiblingRef.current = comboBoxListRef.current!.renderRoot.host.previousSibling;
	}, []);

	return (
		<div className='comboBoxInput'>
			<SlInput
				data-is-combobox-input
				value={value}
				name={name}
				label={label}
				className={className}
				onSlInput={disabled ? undefined : debouncedOnInput}
				onSlFocus={disabled ? undefined : onInputFocus}
				onSlBlur={disabled ? undefined : onInputBlur}
				onClick={disabled ? undefined : onClick}
				autocomplete='off'
				disabled={disabled}
				size={size}
				readonly={readonly}
				{...delegated}
			>
				{children}
				{error && value !== '' && (
					<SlIcon className='errorIcon' name='exclamation-circle' slot='prefix'></SlIcon>
				)}
				{caret && <SlIcon className='caretIcon' name='chevron-down' slot='suffix'></SlIcon>}
			</SlInput>
			{error && <div className='error'>{error}</div>}
		</div>
	);
};

export { ComboBoxList, ComboBoxInput };
