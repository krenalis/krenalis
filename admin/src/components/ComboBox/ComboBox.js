import { forwardRef, useRef, useEffect } from 'react';
import './ComboBox.css';
import { debounce } from '../../utils/debounce';
import { SlInput, SlMenu, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

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

const ComboBoxList = forwardRef(({ isOpen, searchTerm, comboBoxItems }, ref) => {
	const onMouseDown = (e) => {
		// prevent ComboBoxInput from losing focus
		if (document.activeElement.dataset.isComboboxInput) {
			e.preventDefault();
		}
	};

	let searchResults = [];
	for (let item of comboBoxItems) {
		let term = item.searchableTerm;
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
		let aTerm = a.searchableTerm;
		let bTerm = b.searchableTerm;
		if (aTerm === searchTerm) return -1;
		if (bTerm === searchTerm) return 1;
		if (aTerm.startsWith(searchTerm) && !bTerm.startsWith(searchTerm)) return -1;
		else if (!aTerm.startsWith(searchTerm) && bTerm.startsWith(searchTerm)) return 1;
		return 0;
	});

	return (
		<SlMenu
			tabIndex='-1' // menu items must be selected only via "ArrowDown" key. "Tab" press must instead focus the next input.
			ref={ref}
			data-is-combobox-list
			className='comboBoxList'
			data-isOpen={isOpen && searchResults.length > 0}
			onMouseDown={onMouseDown}
		>
			{searchResults.map((item) => item.content)}
		</SlMenu>
	);
});

const ComboBoxInput = ({
	comboBoxListRef,
	onInput,
	openComboBoxList,
	closeComboBoxList,
	setFocused,
	children,
	error,
	disabled,
	...delegated
}) => {
	const onKeyUpRef = useRef(null);
	const previousListSibling = useRef(null);

	const onKeyUp = (e) => {
		if (e.key === 'Escape') {
			onInputBlur();
		}
		if (e.key === 'ArrowDown') {
			let menuItems = comboBoxListRef.current.querySelectorAll('sl-menu-item');
			if (menuItems.length > 0) {
				menuItems[0].focus();
			}
		}
	};

	const onInputFocus = (e) => {
		window.addEventListener('keyup', onKeyUpRef.current);
		let input = e.currentTarget;
		setTimeout(() => {
			input.after(comboBoxListRef.current);
			openComboBoxList();
			setFocused(input);
		});
	};

	const debouncedOnInput = debounce(onInput, 300);

	const handleInput = (e) => {
		debouncedOnInput(e);
	};

	const onInputBlur = (e) => {
		if (e != null) {
			onInput(e);
		}
		window.removeEventListener('keyup', onKeyUpRef.current);
		setTimeout(() => {
			let isComboBoxListFocused = document.activeElement.closest('[data-is-combobox-list]');
			if (!isComboBoxListFocused) {
				closeComboBoxList();
				previousListSibling.current.after(comboBoxListRef.current);
				setFocused(null);
			}
		});
	};

	const onClick = () => {
		window.addEventListener('keyup', onKeyUpRef.current);
		openComboBoxList();
	};

	useEffect(() => {
		onKeyUpRef.current = onKeyUp;
		previousListSibling.current = comboBoxListRef.current.previousSibling;
	}, []);

	return (
		<div className='comboBoxInput'>
			<SlInput
				data-is-combobox-input
				onSlInput={disabled ? null : handleInput}
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
