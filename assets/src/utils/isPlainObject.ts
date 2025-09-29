const isPlainObject = (val: any) => {
	return val !== null && typeof val === 'object' && Object.getPrototypeOf(val) === Object.prototype;
};

export { isPlainObject };
