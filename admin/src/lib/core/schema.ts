const isMetaProperty = (name: string): boolean => {
	return name.startsWith('_');
};

export { isMetaProperty };
