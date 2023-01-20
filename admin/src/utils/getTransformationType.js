export const getTransformationType = (t) => {
	if (t.PredefinedFunc !== null) {
		return 'predefined';
	} else if (t.InProperties.length === 1 && t.OutProperties.length === 1 && t.CustomFunc == null) {
		return 'one-to-one';
	} else {
		return 'custom';
	}
};
