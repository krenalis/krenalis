const transformAnonymousIdentifiers = (identifiers) => {
	const transformed = [];
	if (identifiers.Priority.length === 0) {
		transformed.push([{ value: '', error: '' }, { value: '' }]);
	} else {
		for (const property of identifiers.Priority) {
			const mapped = identifiers.Mapping[property];
			transformed.push([{ value: mapped, error: '' }, { value: property }]);
		}
	}
	return transformed;
};

const untransformAnonymousIdentifiers = (transformed) => {
	const untransformed = { Priority: [], Mapping: {} };
	for (const [mapped, property] of transformed) {
		untransformed.Priority.push(property.value);
		untransformed.Mapping[property.value] = mapped.value;
	}
	return untransformed;
};

const checkAnonymousIdentifiers = (anonymousIdentifiers) => {
	for (const [mapped] of anonymousIdentifiers) {
		if (mapped.error !== '') return false;
	}
	return true;
};

export { transformAnonymousIdentifiers, untransformAnonymousIdentifiers, checkAnonymousIdentifiers };
