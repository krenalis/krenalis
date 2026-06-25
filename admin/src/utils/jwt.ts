const isJwtExpired = (token: string): boolean => {
	if (!token) {
		return false;
	}
	try {
		const payload = JSON.parse(atob(token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/')));
		return payload.exp * 1000 < Date.now();
	} catch {
		return false;
	}
};

export { isJwtExpired };
