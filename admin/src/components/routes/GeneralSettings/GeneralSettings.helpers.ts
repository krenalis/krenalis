import { FlatSchema } from '../../../lib/core/pipeline';

const checkUIPreferences = (property: string, schema: FlatSchema): string => {
	if (schema == null || property === '') {
		return '';
	}
	if (property.length > 100) {
		return `property "${property}" is longer than 100 characters`;
	}
	if (schema[property] == null) {
		return `property "${property}" does not exist in the profile schema`;
	}
	return '';
};

export { checkUIPreferences };
