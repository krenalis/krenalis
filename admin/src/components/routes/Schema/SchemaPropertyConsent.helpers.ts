import { ConsentPurpose } from '../../../lib/api/types/workspace';
import { ObjectType } from '../../../lib/api/types/types';
import { flattenSchema } from '../../../lib/core/pipeline';

const getConsentPurposesByPropertyPath = (
	schema: ObjectType,
	purposes: ConsentPurpose[],
): Map<string, ConsentPurpose[]> => {
	const result = new Map<string, ConsentPurpose[]>();
	const flatSchema = flattenSchema(schema);
	if (flatSchema == null) {
		return result;
	}
	for (const purpose of purposes) {
		const location = purpose.profileConsentLocation;
		if (location == null) {
			continue;
		}
		const propertyPath = location.property;
		const key = location.jsonKey || null;
		if (flatSchema[propertyPath] == null) {
			continue;
		}
		const kind = flatSchema[propertyPath].type;
		const isBooleanConsent = kind === 'boolean' && key == null;
		const isJSONConsent = kind === 'json' && key != null;
		if (!isBooleanConsent && !isJSONConsent) {
			continue;
		}
		const propertyPurposes = result.get(propertyPath) ?? [];
		propertyPurposes.push(purpose);
		result.set(propertyPath, propertyPurposes);
	}
	return result;
};

export { getConsentPurposesByPropertyPath };
