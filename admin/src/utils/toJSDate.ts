const toJSDate = (dateString: string): Date | null => {
	if (!isNaN(Number(dateString))) {
		// It's a UNIX timestamp.
		const timestamp = Number(dateString);
		let date: Date;
		if (timestamp < 1e12) {
			// It's in seconds.
			date = new Date(timestamp * 1000);
		} else {
			date = new Date(timestamp);
		}
		return date;
	}

	const match = dateString.match(/^(\d{4}-\d{2}-\d{2}) (\d{2}:\d{2}:\d{2})(?: ([+-]\d{4}))?$/);
	if (match) {
		// It's a database timestamp.
		let [, datePart, timePart, timezone] = match;
		let isoDate = `${datePart}T${timePart}`;
		if (timezone) {
			// It's a database timestamp with timezone.
			timezone = timezone.replace(/([+-]\d{2})(\d{2})/, '$1:$2');
			isoDate += timezone;
		} else {
			isoDate += 'Z';
		}
		return new Date(isoDate);
	}

	const date = new Date(dateString);
	const isValid = !isNaN(date.getTime());

	if (isValid) {
		return date;
	}

	return null;
};

export default toJSDate;
