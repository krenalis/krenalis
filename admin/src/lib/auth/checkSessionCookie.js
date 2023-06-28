// TODO: this is a mock session check needed to use the application in the early
// stages of development and is not intended to be used in production. Server
// side authentication must be implemented in order to correctly check the
// session cookie.
const checkSessionCookie = () => {
	if (document.cookie.includes('session=1')) {
		return true;
	} else {
		return false;
	}
};

export default checkSessionCookie;
