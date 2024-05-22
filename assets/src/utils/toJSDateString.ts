const toJSDateString = (dateString: string) => {
	const s1 = dateString.replace(' ', 'T');
	const s2 = s1.replace(' ', 'Z');
	return s2.slice(0, s2.indexOf('Z') + 1);
};

export default toJSDateString;
