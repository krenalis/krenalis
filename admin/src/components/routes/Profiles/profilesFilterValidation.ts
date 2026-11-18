import { Filter, FilterOperator } from '../../../lib/api/types/pipeline';
import { ObjectType } from '../../../lib/api/types/types';
import {
	FILTER_OPERATORS,
	flattenSchema,
	getCompatibleFilterOperators,
	isBetweenOperator,
	isFilterGroup,
	isUnaryOperator,
	splitPropertyAndPath,
	validateAndNormalizeFilterCondition,
} from '../../../lib/core/pipeline';
import { isValidPropertyPath } from '../../../utils/filters';

// getProfilesFilterErrors associates validation errors with the editor fields.
const getProfilesFilterErrors = (filter: Filter | null, schema: ObjectType): Record<string, string> => {
	const errors: Record<string, string> = {};
	const flatSchema = flattenSchema(schema);
	const validateGroup = (group: Filter, groupPath: number[]) => {
		for (const [index, condition] of group.rules.entries()) {
			const conditionPath = [...groupPath, index];
			if (isFilterGroup(condition)) {
				validateGroup(condition, conditionPath);
				continue;
			}
			const id = conditionPath.join('-');
			const propertyName = condition.property.trim();
			if (propertyName === '') {
				errors[`property-${id}`] = 'Choose a property';
				continue;
			}
			const [base, path] = splitPropertyAndPath(propertyName, flatSchema);
			const property = flatSchema[base];
			// Validate the property, path and operator before checking individual values.
			try {
				validateAndNormalizeFilterCondition({ ...condition, values: [] }, property, path, propertyName);
			} catch (error) {
				const field =
					property == null
						? 'property'
						: property.type === 'json' && path !== '' && !isValidPropertyPath(path)
							? 'path'
							: 'operator';
				errors[`${field}-${id}`] = error instanceof Error ? error.message : String(error);
				continue;
			}
			if (
				!getCompatibleFilterOperators(property, path !== '').includes(
					FILTER_OPERATORS.indexOf(condition.operator as FilterOperator),
				)
			) {
				errors[`operator-${id}`] = 'Choose a compatible operator';
				continue;
			}
			if (isUnaryOperator(condition.operator)) {
				continue;
			}
			const values = condition.values ?? [];
			const valueCount = Math.max(values.length, isBetweenOperator(condition.operator) ? 2 : 1);
			for (let position = 0; position < valueCount; position++) {
				try {
					validateAndNormalizeFilterCondition(
						{ ...condition, values: [values[position] ?? ''] },
						property,
						path,
						propertyName,
					);
				} catch (error) {
					errors[`value-${id}-${position}`] = error instanceof Error ? error.message : String(error);
				}
			}
		}
	};
	if (filter != null) {
		validateGroup(filter, []);
	}
	return errors;
};

export { getProfilesFilterErrors };
