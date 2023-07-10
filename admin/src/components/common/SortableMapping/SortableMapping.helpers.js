import { flattenSchema } from '../../../lib/connections/action';
import { getExpressionVariables } from '../../../lib/connections/action';

const checkInputValue = (value, schema) => {
	if (value === '') return;
	const flatSchema = flattenSchema(schema);
	const variables = getExpressionVariables(value);
	for (const variable of variables) {
		const doesValueExist = variable in flatSchema;
		if (!doesValueExist) {
			return `"${variable}" does not exist in schema`;
		}
	}
};

const checkOutputValue = (value, schema) => {
	if (value === '') return;
	const flatSchema = flattenSchema(schema);
	const doesValueExist = value in flatSchema;
	if (!doesValueExist) {
		return `"${value}" does not exist in schema`;
	}
};

export { checkInputValue, checkOutputValue };
