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

export const debounceWithAbort = (func: (...args: any) => Promise<any>, time: number) => {
	const controller = new AbortController();
	let timer: number | null;
	let isFetching: boolean = false;
	return function (...args: any) {
		const context = this;
		if (timer) {
			clearTimeout(timer);
		}
		if (isFetching) {
			controller.abort();
		}
		timer = window.setTimeout(
			async () => {
				timer = null;
				isFetching = true;
				try {
					await func.apply(context, [...args, controller.signal]);
				} catch (err) {
					isFetching = false;
					throw err;
				}
				isFetching = false;
			},
			time == null ? 500 : time,
		);
	};
};
