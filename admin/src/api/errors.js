class NotFoundError extends Error {
	constructor(code, message, details) {
		super();
		this.name = 'NotFoundError';
		this.code = code;
		this.message = message;
		this.cause = details;
	}
}

class BadRequestError extends Error {
	constructor(code, message, details) {
		super();
		this.name = 'BadRequestError';
		this.code = code;
		this.message = message;
		this.cause = details;
	}
}

class UnprocessableError extends Error {
	constructor(code, message, details) {
		super();
		this.name = 'UnprocessableError';
		this.code = code;
		this.message = message;
		this.cause = details;
	}
}

export { NotFoundError, BadRequestError, UnprocessableError };
