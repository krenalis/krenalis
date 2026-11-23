import { Property } from '../../lib/api/types/types';

const getPropertyPathDisplayValue = (name: string, getProperty: (path: string) => Property | undefined): string => {
	let path = '';
	const labels: string[] = [];
	for (const fragment of name.split('.')) {
		path = path === '' ? fragment : `${path}.${fragment}`;
		const property = getProperty(path);
		const displayName = property?.displayName?.trim() ?? '';
		const propertyName = property?.name ?? '';
		labels.push(displayName !== '' ? displayName : propertyName !== '' ? propertyName : fragment);
	}
	return labels.join(' › ');
};

export { getPropertyPathDisplayValue };
