const isPlainObject = (val: any): boolean => {
	return val !== null && typeof val === 'object';
};

export { isPlainObject };
