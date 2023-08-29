import { TransformedAction, TransformedMapping } from '../../../lib/helpers/transformedAction';

const updateMappingProperty = (action: TransformedAction, name: string, value: string, error: string) => {
	const getAlternativeProperties = (name: string, mapping: TransformedMapping): string[] => {
		const indentation = mapping[name].indentation;
		const parentProperties: string[] = [];
		for (const k in mapping) {
			if (mapping[k].indentation! < indentation! && name.startsWith(k)) {
				parentProperties.push(k);
			}
		}
		const childrenProperties: string[] = [];
		for (const k in mapping) {
			if (mapping[k].indentation! > indentation! && k.startsWith(name)) {
				childrenProperties.push(k);
			}
		}
		return [...parentProperties, ...childrenProperties];
	};

	const a = { ...action };

	if (a.Mapping == null) return a;

	if (a.Mapping[name].value === '' && value !== '') {
		const alternativeProperties = getAlternativeProperties(name, a.Mapping);
		// disable
		for (const k in a.Mapping) {
			if (alternativeProperties.includes(k)) {
				a.Mapping[k].disabled = true;
			}
		}
	} else if (value === '') {
		let hasFilledSiblings = false;
		const { root, indentation } = a.Mapping[name];
		for (const k in a.Mapping) {
			if (
				k !== name &&
				a.Mapping[k].root === root &&
				a.Mapping[k].indentation === indentation &&
				a.Mapping[k].value !== ''
			) {
				hasFilledSiblings = true;
			}
		}
		if (!hasFilledSiblings) {
			// enable
			const alternativeProperties = getAlternativeProperties(name, a.Mapping);
			for (const k in a.Mapping) {
				if (alternativeProperties.includes(k)) {
					a.Mapping[k].disabled = false;
				}
			}
		}
	}

	a.Mapping[name].error = error;
	a.Mapping[name].value = value;
	return a;
};

export { updateMappingProperty };
