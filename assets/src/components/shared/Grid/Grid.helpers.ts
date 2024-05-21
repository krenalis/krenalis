const getChildIndexClassname = (i: number, listLength: number) => {
	let index = i + 1;
	let className = '';
	if (index === 1) {
		className += ' grid-el--first';
	}
	if (index === listLength) {
		className += ' grid-el--last';
	}
	if (index % 2 === 0) {
		className += ' grid-el--even';
	} else {
		className += ' grid-el--odd';
	}
	return className.trim();
};

export { getChildIndexClassname };
