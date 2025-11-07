const toBase64 = async (file: File): Promise<string> => {
	return new Promise((resolve) => {
		const r = new FileReader();
		r.onloadend = () => {
			const result = r.result as string;
			resolve(result.substring(result.indexOf(',') + 1));
		};
		r.readAsDataURL(file);
	});
};

export { toBase64 };
