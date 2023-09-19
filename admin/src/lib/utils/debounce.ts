export const debounce = (func: (...args: any) => any, time: number) => {
	let timer: number | null;
	return function (...args: any) {
		const context = this;
		if (timer) {
			clearTimeout(timer);
		}
		timer = window.setTimeout(
			() => {
				timer = null;
				func.apply(context, args);
			},
			time == null ? 500 : time,
		);
	};
};
