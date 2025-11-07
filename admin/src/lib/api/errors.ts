class NotFoundError extends Error {
	constructor(message: string) {
		super();
		this.name = 'NotFoundError';
		this.message = message ? message : 'The requested resource was not found';
	}
}

class UnavailableError extends Error {
	cause: string;

	constructor(message: string, cause: string) {
		super();
		this.name = 'UnavailableError';
		this.message = message;
		this.cause = cause;
	}
}

class UnprocessableError extends Error {
	code: string;
	cause: string;

	constructor(code: string, message: string, cause: string) {
		super();
		this.name = 'UnprocessableError';
		this.code = code;
		this.message = message;
		this.cause = cause;
	}
}

class UnauthorizedError extends Error {
	constructor() {
		super();
		this.name = 'UnauthorizedError';
	}
}

export { NotFoundError, UnavailableError, UnprocessableError, UnauthorizedError };
