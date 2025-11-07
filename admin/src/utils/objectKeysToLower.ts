function objectKeysToLower(obj: Record<string, any> | undefined): Record<string, any> | undefined {
	if (obj == null) return;
	const newObj = {};
	for (const k in obj) {
		if (Object.prototype.hasOwnProperty.call(obj, k)) {
			newObj[k.toLowerCase()] = obj[k];
		}
	}
	return newObj;
}

export default objectKeysToLower;
