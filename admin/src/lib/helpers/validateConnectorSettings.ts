import { UIValues } from '../../types/external/api';
import ConnectorField, { InputField } from '../../types/external/ui';

const validateConnectorSettings = (values: UIValues, fields: ConnectorField[]) => {
	for (const key in values) {
		if (hasOnlyIntegerPart(key, fields)) {
			const value = values[key];
			const n = Number(value);
			if (isNaN(n)) {
				throw `${key} must be a valid number`;
			}
			if (!Number.isSafeInteger(n)) {
				throw `${key} must be a valid integer`;
			}
		}
	}
	return values;
};

const hasOnlyIntegerPart = (key: string, fields: ConnectorField[]): boolean => {
	for (const f of fields) {
		const isInput = f.ComponentType === 'Input';
		if (isInput) {
			const input = f as InputField;
			if (input.Name === key) {
				return input.OnlyIntegerPart;
			}
		}
	}
	return false;
};

export { validateConnectorSettings };
