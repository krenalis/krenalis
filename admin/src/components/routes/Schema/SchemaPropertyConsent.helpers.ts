import { ConsentPurpose } from '../../../lib/api/types/workspace';
import { ObjectType } from '../../../lib/api/types/types';
import { flattenSchema, splitPropertyAndPath } from '../../../lib/core/pipeline';

const getConsentPurposeEventPath = (purpose: ConsentPurpose): string =>
	purpose.eventPath || `context.consents.${purpose.code}`;

const getConsentPurposeProfilePath = (purpose: ConsentPurpose): string =>
	purpose.profilePath || `consents.${purpose.code}`;

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
		const [propertyPath, insidePath] = splitPropertyAndPath(getConsentPurposeProfilePath(purpose), flatSchema);
		if (propertyPath === '') {
			continue;
		}
		const kind = flatSchema[propertyPath].type;
		const isBooleanConsent = kind === 'boolean' && insidePath === '';
		const isJSONConsent = kind === 'json' && insidePath !== '';
		if (!isBooleanConsent && !isJSONConsent) {
			continue;
		}
		const propertyPurposes = result.get(propertyPath) ?? [];
		propertyPurposes.push(purpose);
		result.set(propertyPath, propertyPurposes);
	}
	return result;
};

export { getConsentPurposeEventPath, getConsentPurposeProfilePath, getConsentPurposesByPropertyPath };
