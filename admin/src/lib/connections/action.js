const SCHEDULE_PERIODS = {
	5: '5m',
	15: '15m',
	30: '30m',
	60: '1h',
	120: '2h',
	180: '3h',
	360: '6h',
	480: '8h',
	720: '12h',
	1440: '24h',
};

const EXPORT_MODE_OPTIONS = {
	CreateOnly: 'Create only',
	UpdateOnly: 'Update only',
	CreateOrUpdate: 'Create and update',
};

const flattenSchema = (schema) => {
	if (schema == null) return null;

	const flattenProperty = (property) => {
		return {
			value: '',
			required: property.required,
			type: property.type.name,
			label: property.label,
			full: { ...property },
		};
	};

	const flattenSubProperties = (parentName, parentIndentation, properties) => {
		let flattenedSubProperties = {};
		parentIndentation += 1;
		for (const property of properties) {
			const name = `${parentName}.${property.name}`;
			const flattened = flattenProperty(property);
			flattened.indentation = parentIndentation;
			flattened.root = name.substring(0, name.indexOf('.'));
			flattenedSubProperties[name] = flattened;
			if (property.type.name === 'Object') {
				const flattenedProperties = flattenSubProperties(name, parentIndentation, property.type.properties);
				flattenedSubProperties = { ...flattenedSubProperties, ...flattenedProperties };
			}
		}
		return flattenedSubProperties;
	};

	let flattenedSchema = {};
	for (const property of schema.properties) {
		const indentation = 0;
		const flattened = flattenProperty(property);
		flattened.indentation = indentation;
		flattened.root = property.name;
		flattenedSchema[property.name] = flattened;
		if (property.type.name === 'Object') {
			const flattenedSubProperties = flattenSubProperties(property.name, indentation, property.type.properties);
			flattenedSchema = { ...flattenedSchema, ...flattenedSubProperties };
		}
	}

	return flattenedSchema;
};

const convertActionMapping = (mapping, outputSchema) => {
	const properties = flattenSchema(outputSchema);
	for (const propertyName in properties) {
		const isPropertyMapped = mapping[propertyName] != null;
		if (isPropertyMapped) {
			const mappedValue = mapping[propertyName];
			properties[propertyName].value = mappedValue;

			// Disable family properties with different indentation.
			const { root, indentation } = properties[propertyName];
			for (const name in properties) {
				const isFamilyProperty = properties[name].root === root;
				const hasDifferentIndentation = properties[name].indentation !== indentation;
				if (isFamilyProperty && hasDifferentIndentation) {
					properties[name].disabled = true;
				}
			}
		}
	}

	return properties;
};

const computeDefaultAction = (actionType, outputSchema, fields) => {
	const action = {
		Name: actionType.Name,
		Enabled: false,
		Filter: null,
		Mapping: flattenSchema(outputSchema),
		InSchema: null,
		OutSchema: null,
		PythonSource: null,
	};
	if (fields.includes('Query')) {
		action.Query = '';
	}
	if (fields.includes('Path')) {
		action.Path = '';
	}
	if (fields.includes('Sheet')) {
		action.Sheet = '';
	}
	if (fields.includes('ExportMode')) {
		action.ExportMode = Object.keys(EXPORT_MODE_OPTIONS)[0];
	}
	if (fields.includes('MatchingProperties')) {
		action.MatchingProperties = { Internal: '', External: '' };
	}
	return action;
};

const computeActionFields = (connection, actionType, schemas) => {
	const fields = [];
	if (connection.type === 'App' && connection.role === 'Destination' && actionType.Target === 'Events') {
		fields.push('Filter');
	}
	if (
		(connection.type === 'App' && schemas.In != null && schemas.Out != null) ||
		(connection.type === 'Database' && connection.role === 'Source') ||
		(connection.type === 'File' && connection.role === 'Source')
	) {
		fields.push('Mapping');
	}
	if (
		connection.type === 'App' &&
		connection.role === 'Destination' &&
		(actionType.Target === 'Users' || actionType.Target === 'Groups')
	) {
		fields.push('MatchingProperties');
		fields.push('ExportMode');
		fields.push('Filter');
	}
	if (connection.type === 'Database' && connection.role === 'Source') {
		fields.push('Query');
	}
	if (connection.type === 'File') {
		if (connection.role === 'Destination') {
			fields.push('Filter');
		}
		fields.push('Path');
		if (connection.connector.hasSheets) {
			fields.push('Sheet');
		}
	}
	return fields;
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
	SCHEDULE_PERIODS,
	EXPORT_MODE_OPTIONS,
	flattenSchema,
	convertActionMapping,
	computeDefaultAction,
	computeActionFields,
	getExpressionVariables,
};
