import { TransformedMapping } from '../../../lib/core/action';

const checkUIPreferences = (property: string, schema: TransformedMapping): string => {
	if (schema == null || property === '') {
		return '';
	}
	if (property.length > 100) {
		return `property "${property}" is longer than 100 characters`;
	}
	if (schema[property] == null) {
		return `property "${property}" does not exist in the user schema`;
	}
	return '';
};

export { checkUIPreferences };
