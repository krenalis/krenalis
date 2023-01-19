const getChildIndexClassname = (i, listLength) => {
	let index = i + 1;
	console.log(i, listLength);
	let className = '';
	if (index === 1) {
		className += ' first';
	}
	if (index === listLength) {
		className += ' last';
	}
	if (index % 2 === 0) {
		className += ' even';
	} else {
		className += ' odd';
	}
	return className.trim();
};

export default getChildIndexClassname;
