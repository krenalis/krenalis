const parseInitialUserRecords = (value: string): number => {
	if (!/^(?:\d+|\d{1,3}(?:,\d{3})+)$/.test(value)) {
		throw new Error('Initial user records must be an integer from 0 to 1,000,000.');
	}
	const count = Number(value.replaceAll(',', ''));
	if (!Number.isInteger(count) || count < 0 || count > 1000000) {
		throw new Error('Initial user records must be an integer from 0 to 1,000,000.');
	}
	return count;
};

const parseDuplicateRecordPercent = (value: string): number => {
	if (!/^\d+(?:\.\d{1,2})?$/.test(value)) {
		throw new Error('Duplicate records must have at most two decimal places.');
	}
	const percent = Number(value);
	if (percent < 0 || percent > 50) {
		throw new Error('Duplicate records must be between 0% and 50%.');
	}
	return percent;
};

export { parseInitialUserRecords, parseDuplicateRecordPercent };
