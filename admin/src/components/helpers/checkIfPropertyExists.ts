import { FlatSchema } from '../../lib/core/pipeline';

// checkIfPropertyExists returns an error message if property is absent or hidden.
const checkIfPropertyExists = (property: string, schema: FlatSchema, toHide?: string[]): string => {
	if (schema == null || property === '' || property == null) {
		return '';
	}
	if (schema[property] == null || toHide?.includes(property)) {
		return `Property "${property}" does not exist`;
	}
	return '';
};

export { checkIfPropertyExists };
