const isMetaProperty = (name: string): boolean => {
	return name.length >= 5 && name.startsWith('__') && name.endsWith('__');
};

export { isMetaProperty };
