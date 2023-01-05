export const getTransformationType = (t) => {
	if (t.PredefinedFunc !== 0) {
		return 'predefined';
	} else if (t.In.properties.length === 1 && t.Out.properties.length === 1 && t.SourceCode === '') {
		return 'one-to-one';
	} else {
		return 'custom';
	}
};
