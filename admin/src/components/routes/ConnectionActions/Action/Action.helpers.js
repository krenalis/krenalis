const rawTransformationFunction = `def transform($parameterName: dict) -> dict:
	return {}
`;

const getDefaultMappings = (schema) => {
	if (schema == null) {
		return null;
	}
	const getSubProperties = (parentName, properties, indentation) => {
		let subProperties = {};
		indentation += 1;
		for (const subP of properties) {
			const key = `${parentName}.${subP.name}`;
			subProperties[key] = {
				value: '',
				indentation: indentation,
				root: key.substring(0, key.indexOf('.')),
				disabled: false,
				required: subP.required != null ? subP.required : false,
				type: subP.type.name,
				label: subP.label,
				full: subP,
			};
			if (subP.type.name === 'Object') {
				const nestedSubProperties = getSubProperties(key, subP.type.properties, indentation);
				subProperties = { ...subProperties, ...nestedSubProperties };
			}
		}
		return subProperties;
	};
	let defaultMappings = {};
	for (const p of schema.properties) {
		const indentation = 0;
		defaultMappings[p.name] = {
			value: '',
			indentation: indentation,
			root: p.name,
			disabled: false,
			required: p.required != null ? p.required : false,
			type: p.type.name,
			label: p.label,
			full: p,
		};
		if (p.type.name === 'Object') {
			const subProperties = getSubProperties(p.name, p.type.properties, indentation);
			defaultMappings = { ...defaultMappings, ...subProperties };
		}
	}
	return defaultMappings;
};

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
	const properties = getDefaultMappings(schema);
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

const addPropertyToActionSchema = (action, side, property) => {
	let a = { ...action };
	if (side === 'input') {
		if (a.InSchema == null) {
			a.InSchema = { name: 'Object', properties: [{ ...property }] };
		} else {
			a.InSchema.properties.push({ ...property });
		}
	} else {
		if (a.OutSchema == null) {
			a.OutSchema = { name: 'Object', properties: [{ ...property }] };
		} else {
			a.OutSchema.properties.push({ ...property });
		}
	}
	return a;
};

const removePropertyFromActionSchema = (action, side, propertyName) => {
	let a = { ...action };
	if (side === 'input') {
		let filtered = a.InSchema.properties.filter((p) => p.name !== propertyName);
		if (filtered.length === 0) {
			a.InSchema = null;
		} else {
			a.InSchema.properties = filtered;
		}
	} else {
		let filtered = a.OutSchema.properties.filter((p) => p.name !== propertyName);
		if (filtered.length === 0) {
			a.OutSchema = null;
		} else {
			a.OutSchema.properties = filtered;
		}
	}
	return a;
};

function getExpressionVariables(expression) {
	const regex = /(["'])(?:\\.|(?!\1)[^\\])*\1|\b([a-zA-Z_][a-zA-Z0-9_]*\b)(?!\s*\()/g;
	const variables = [];
	let match;
	while ((match = regex.exec(expression)) !== null) {
		if (match[2]) {
			const variable = match[2].split('.')[0];
			variables.push(variable);
		}
	}
	return variables;
}

export {
	getDefaultMappings,
	rawTransformationFunction,
	updateMappingProperty,
	getSchemaComboboxItems,
	addPropertyToActionSchema,
	removePropertyFromActionSchema,
	getExpressionVariables,
};
