const SUPPORTED_IDENTIFIERS_TYPES = [
	'Int',
	'Int8',
	'Int16',
	'Int24',
	'Int64',
	'UInt',
	'UInt8',
	'UInt16',
	'UInt24',
	'UInt64',
	'Decimal',
	'UUID',
	'Inet',
	'Text',
	'Array',
];

const DEFAULT_IDENTIFIERS_MAPPING = [[{ value: '', error: '' }, { value: '' }]];

const isTypeSupportedAsIdentifier = (type) => {
	if (SUPPORTED_IDENTIFIERS_TYPES.includes(type)) {
		return true;
	}
	return false;
};

const validateIdentifiersMapping = (identifiersMapping) => {
	for (let i = 0; i < identifiersMapping.length; i++) {
		const [mapped, identifier] = identifiersMapping[i];
		if (mapped.value === '') {
			return 'You cannot map an identifier with an empty value';
		}
		if (identifier.value === '') {
			return 'You cannot use an empty value as identifier';
		}
		if (mapped.error || identifier.error) {
			return 'You must fix the errors in the identifier mapping';
		}
		const otherAssociations = [...identifiersMapping.slice(0, i), ...identifiersMapping.slice(i + 1)];
		for (const [, otherIdentifier] of otherAssociations) {
			if (identifier.value === otherIdentifier.value) {
				return 'You cannot use the same identifier twice';
			}
		}
	}
};

const transformActionIdentifiers = (identifiers, mapping) => {
	return identifiers.map((identifier) => [{ value: mapping[identifier].value, error: '' }, { value: identifier }]);
};

const untransformActionIdentifiers = (transformed) => {
	return transformed.map(([, identifier]) => identifier.value);
};

const transformAnonymousIdentifiers = (identifiers) => {
	const transformed = [];
	if (identifiers.Priority.length === 0) {
		transformed.push([{ value: '', error: '' }, { value: '' }]);
	} else {
		for (const identifier of identifiers.Priority) {
			const mapped = identifiers.Mapping[identifier];
			transformed.push([{ value: mapped, error: '' }, { value: identifier }]);
		}
	}
	return transformed;
};

const untransformAnonymousIdentifiers = (transformed) => {
	const untransformed = { Priority: [], Mapping: {} };
	for (const [mapped, identifier] of transformed) {
		untransformed.Priority.push(identifier.value);
		untransformed.Mapping[identifier.value] = mapped.value;
	}
	return untransformed;
};

export {
	DEFAULT_IDENTIFIERS_MAPPING,
	isTypeSupportedAsIdentifier,
	validateIdentifiersMapping,
	transformActionIdentifiers,
	untransformActionIdentifiers,
	transformAnonymousIdentifiers,
	untransformAnonymousIdentifiers,
};
