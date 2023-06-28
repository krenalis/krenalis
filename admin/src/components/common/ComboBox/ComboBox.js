import { forwardRef, useRef, useEffect, useState, useImperativeHandle } from 'react';
import './ComboBox.css';
import { debounce } from '../../../utils/debounce';
import { SlInput, SlMenu, SlMenuItem, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

//
// const selectItem(value) {
// 	// do something with the value
// 	// close the ComboBoxList
// }
//
// comboBoxItems = {
// 	content: <slMenuItem onClick={() => selectItem(value)}>{label}</slMenuItem>, --> The content actually shown.
// 	term: {label} --> the search term of the item. Used to filter after user input.
// }
//

const ComboBoxList = forwardRef(({ items, onSelect }, ref) => {
	const [isOpen, setIsOpen] = useState(false);
	const [searchTerm, setSearchTerm] = useState('');
	const [currentComboboxInput, setCurrentComboboxInput] = useState(null);

	const comboBoxListMenuRef = useRef(null);

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
				updateSearchTerm(term) {
					setSearchTerm(term);
				},
				updateCurrentComboboxInput(input) {
					setCurrentComboboxInput(input);
				},
			};
		},
		[]
	);

	const onMouseDown = (e) => {
		// prevent ComboBoxInput from losing focus
		if (document.activeElement.dataset.isComboboxInput) {
			e.preventDefault();
		}
	};

	const searchResults = [];
	for (const item of items) {
		const term = item.searchableTerm;
		if (
			term.includes(searchTerm) ||
			term.includes(searchTerm.charAt(0).toUpperCase() + searchTerm.slice(1)) ||
			term.includes(searchTerm.toUpperCase) ||
			term.includes(searchTerm.toLowerCase)
		) {
			searchResults.push(item);
		}
	}

	searchResults.sort((a, b) => {
		const aTerm = a.searchableTerm;
		const bTerm = b.searchableTerm;
		if (aTerm === searchTerm) return -1;
		if (bTerm === searchTerm) return 1;
		if (aTerm.startsWith(searchTerm) && !bTerm.startsWith(searchTerm)) return -1;
		else if (!aTerm.startsWith(searchTerm) && bTerm.startsWith(searchTerm)) return 1;
		return 0;
	});

	return (
		<SlMenu
			tabIndex='-1' // menu items must be selected only via "ArrowDown" key. "Tab" press must instead focus the next input.
			ref={comboBoxListMenuRef}
			data-is-combobox-list
			className='comboBoxList'
			data-isOpen={isOpen && searchResults.length > 0}
			onMouseDown={onMouseDown}
		>
			{searchResults.map((item) => {
				return (
					<SlMenuItem
						onClick={() => {
							setSearchTerm(item.searchableTerm);
							onSelect(currentComboboxInput, item.searchableTerm);
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

const ComboBoxInput = ({ comboBoxListRef, onInput: onInputProp, children, error, disabled, ...delegated }) => {
	const onKeyUpRef = useRef(null);
	const previousListSiblingRef = useRef(null);

	const onKeyUp = (e) => {
		if (e.key === 'Escape') {
			onInputBlur();
		}
		if (e.key === 'ArrowDown') {
			const menuItems = comboBoxListRef.current.renderRoot.host.querySelectorAll('sl-menu-item');
			if (menuItems.length > 0) {
				menuItems[0].focus();
			}
		}
	};

	const onInputFocus = (e) => {
		window.addEventListener('keyup', onKeyUpRef.current);
		const input = e.target;
		comboBoxListRef.current.updateCurrentComboboxInput(input);
		comboBoxListRef.current.updateSearchTerm('');
		setTimeout(() => {
			input.after(comboBoxListRef.current.renderRoot.host);
			comboBoxListRef.current.open();
		});
	};

	const onInputBlur = (e) => {
		if (e != null) {
			onInputProp(e);
		}
		window.removeEventListener('keyup', onKeyUpRef.current);
		setTimeout(() => {
			const isComboBoxListFocused = document.activeElement.closest('[data-is-combobox-list]');
			if (!isComboBoxListFocused) {
				comboBoxListRef.current.close();
				previousListSiblingRef.current.after(comboBoxListRef.current.renderRoot.host);
			}
		});
	};

	const onClick = (e) => {
		const input = e.target;
		input.focus();
	};

	const onInput = (e) => {
		comboBoxListRef.current.updateSearchTerm(e.target.value);
		onInputProp(e);
	};

	const debouncedOnInput = debounce(onInput, 300);

	useEffect(() => {
		onKeyUpRef.current = onKeyUp;
		previousListSiblingRef.current = comboBoxListRef.current.renderRoot.host.previousSibling;
	}, []);

	return (
		<div className='comboBoxInput'>
			<SlInput
				data-is-combobox-input
				onSlInput={disabled ? null : debouncedOnInput}
				onSlFocus={disabled ? null : onInputFocus}
				onSlBlur={disabled ? null : onInputBlur}
				onClick={disabled ? null : onClick}
				autocomplete='off'
				disabled={disabled}
				{...delegated}
			>
				{children}
				{error && <SlIcon name='exclamation-circle' slot='prefix'></SlIcon>}
			</SlInput>
			{error && <div className='error'>{error}</div>}
		</div>
	);
};

export { ComboBoxList, ComboBoxInput };
