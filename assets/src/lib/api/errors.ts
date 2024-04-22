class NotFoundError extends Error {
	constructor(message: string) {
		super();
		this.name = 'NotFoundError';
		this.message = message ? message : 'The requested resource was not found';
	}
}

class BadRequestError extends Error {
	cause: string;

	constructor(message: string, cause: string) {
		super();
		this.name = 'BadRequestError';
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

class LoginRequiredError extends Error {
	constructor() {
		super();
		this.name = 'LoginRequiredError';
	}
}

export { NotFoundError, BadRequestError, UnprocessableError, LoginRequiredError };
