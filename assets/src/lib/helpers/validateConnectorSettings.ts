import { ConnectorValues } from '../../types/external/api';
import ConnectorField, { InputField } from '../../types/external/ui';

const validateConnectorSettings = (values: ConnectorValues, fields: ConnectorField[]) => {
	for (const key in values) {
		if (hasOnlyIntegerPart(key, fields)) {
			const value = values[key];
			const n = Number(value);
			if (isNaN(n)) {
				throw new Error(`${key} must be a valid number`);
			}
			if (!Number.isSafeInteger(n)) {
				throw new Error(`${key} must be a valid integer`);
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
