class NotFoundError extends Error {
	constructor(message) {
		super();
		this.name = 'NotFoundError';
		this.message = message;
	}
}

class BadRequestError extends Error {
	constructor(message, cause) {
		super();
		this.name = 'BadRequestError';
		this.message = message;
		this.cause = cause;
	}
}

class UnprocessableError extends Error {
	constructor(code, message, cause) {
		super();
		this.name = 'UnprocessableError';
		this.code = code;
		this.message = message;
		this.cause = cause;
	}
}

export { NotFoundError, BadRequestError, UnprocessableError };
