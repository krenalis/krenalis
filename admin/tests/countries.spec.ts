import { expect, test } from '@playwright/test';
import { getCountryName } from '../src/components/helpers/countries';

test('Resolve current, historical and ambiguous country codes', () => {
	expect(getCountryName('US', 'alpha-2')).toBe('United States');
	expect(getCountryName('CN', 'alpha-2')).toBe('China');
	expect(getCountryName('NL', 'alpha-2')).toBe('Netherlands');
	expect(getCountryName('FX', 'alpha-2')).toBe('Metropolitan France');
	expect(getCountryName('AN', 'alpha-2')).toBe('Netherlands Antilles');
	expect(getCountryName('CS', 'alpha-2')).toBeUndefined();
	for (const [code, name] of [
		['VDR', 'North Vietnam'],
		['PCZ', 'Panama Canal Zone'],
		['RHO', 'Southern Rhodesia'],
		['PUS', 'United States Miscellaneous Pacific Islands'],
		['SKM', 'Kingdom of Sikkim'],
		['ZAR', 'Zaire'],
		['NTZ', 'Saudi–Iraqi Neutral Zone'],
		['WAK', 'Wake Island'],
	]) {
		expect(getCountryName(code, 'alpha-3')).toBe(name);
	}
});
