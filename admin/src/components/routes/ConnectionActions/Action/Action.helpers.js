import { flattenSchema } from '../../../../lib/connections/action';

const updateMappingProperty = (action, name, value) => {
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

	a.Mapping[name].value = value;
	return a;
};

const getSchemaComboboxItems = (schema) => {
	if (schema == null) {
		return [];
	}
	const properties = flattenSchema(schema);
	const propertiesList = [];
	for (const k in properties) {
		let name;
		if (properties[k].label != null && properties[k].label !== '') {
			name = (
				<div className='propertiesItemName'>
					<div className='label'>{properties[k].label}</div>
					<div className='name'>{k}</div>
				</div>
			);
		} else {
			name = <div className='propertiesItemName'>{k}</div>;
		}
		const content = (
			<>
				{name}
				<div className='propertiesItemType'>{properties[k].type}</div>
			</>
		);
		propertiesList.push({
			content: content,
			searchableTerm: k,
		});
	}
	return propertiesList;
};

export { updateMappingProperty, getSchemaComboboxItems };
