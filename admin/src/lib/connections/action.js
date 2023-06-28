class Action {
	constructor(id) {
		this.id = id;
	}

	static SCHEDULE_PERIODS = {
		5: '5m',
		15: '15m',
		30: '30m',
		60: '1h',
		120: '2h',
		180: '3h',
		360: '6h',
		480: '8h',
		720: '12h',
		1440: '24h',
	};
}

export default Action;
