import { ProfileConsentLocation } from '../lib/api/types/workspace';

const INVISIBLE_KEY_CHARACTERS = new RegExp('[\\p{Cc}\\p{Cf}\\p{Zl}\\p{Zp}]', 'u');
const NON_WHITESPACE_CHARACTER = new RegExp('\\P{White_Space}', 'u');
const UNPAIRED_SURROGATE = new RegExp('\\p{Cs}', 'u');

// formatProfileConsentLocation displays the schema path and literal JSON key.
const formatProfileConsentLocation = (location: ProfileConsentLocation | null): string =>
	location == null ? '' : location.property + (location.jsonKey ? `[${JSON.stringify(location.jsonKey)}]` : '');

// validateConsentKey applies the Admin authoring policy without trimming the key.
const validateConsentKey = (key: string): void => {
	if (Array.from(key).length > 1024) {
		throw new Error('Consent keys must be no longer than 1024 characters');
	}
	if (!NON_WHITESPACE_CHARACTER.test(key)) {
		throw new Error('Consent keys must contain at least one non-whitespace character');
	}
	if (INVISIBLE_KEY_CHARACTERS.test(key) || UNPAIRED_SURROGATE.test(key)) {
		throw new Error('Consent keys must not contain invisible characters');
	}
};

export { formatProfileConsentLocation, validateConsentKey };
