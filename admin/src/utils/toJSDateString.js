const toJSDateString = (dateString) => {
	let s1 = dateString.replace(' ', 'T');
	let s2 = s1.replace(' ', 'Z');
	return s2.slice(0, s2.indexOf('Z') + 1);
};

export default toJSDateString;
