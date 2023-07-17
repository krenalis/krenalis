const updateMappingProperty = (action, name, value, error) => {
	const getAlternativeProperties = (name, mapping) => {
		const indentation = mapping[name].indentation;
		const parentProperties = [];
		for (const k in mapping) {
			if (mapping[k].indentation < indentation && name.startsWith(k)) {
				parentProperties.push(k);
			}
		}
		const childrenProperties = [];
		for (const k in mapping) {
			if (mapping[k].indentation > indentation && k.startsWith(name)) {
				childrenProperties.push(k);
			}
		}
		return [...parentProperties, ...childrenProperties];
	};

	const a = { ...action };
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
