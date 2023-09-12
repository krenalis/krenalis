function objectKeysToLower(obj: Record<string, any> | undefined): Record<string, any> | undefined {
	if (obj == null) return;
	const newObj = {};
	for (const k in obj) {
		newObj[k.toLowerCase()] = obj[k];
	}
	return newObj;
}

export default objectKeysToLower;
