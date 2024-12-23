import { ConnectorSettings } from '../api/types/responses';
import ConnectorField, { InputField } from '../api/types/ui';

const validateConnectorSettings = (settings: ConnectorSettings, fields: ConnectorField[]) => {
	for (const key in settings) {
		if (hasOnlyIntegerPart(key, fields)) {
			const s = settings[key];
			const n = Number(s);
			if (isNaN(n)) {
				throw new Error(`${key} must be a valid number`);
			}
			if (!Number.isSafeInteger(n)) {
				throw new Error(`${key} must be a valid integer`);
			}
		}
	}
	return settings;
};

const hasOnlyIntegerPart = (key: string, fields: ConnectorField[]): boolean => {
	for (const f of fields) {
		const isInput = f.componentType === 'Input';
		if (isInput) {
			const input = f as InputField;
			if (input.name === key) {
				return input.onlyIntegerPart;
			}
		}
	}
	return false;
};

export { validateConnectorSettings };
