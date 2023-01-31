export const defaultTransformation = `def transform(user: dict) -> dict:

	"""
	The input parameter is a dictionary representing the user
	imported from the connection.

	Returns a single parameter which is a dictionary representing
	the transformed user.
	"""

	return user
`;

export class Transformation {
	constructor(obj) {
		this.In = {
			name: 'Object',
			properties: [],
		};
		this.Out = {
			name: 'Object',
			properties: [],
		};
		this.PythonSource = defaultTransformation;
		if (obj == null) {
			return;
		}
		Object.assign(this, obj);
	}

	getProperties = (role) => {
		if (role === 'input') {
			return this.In.properties;
		} else {
			return this.Out.properties;
		}
	};

	setProperties = (role, properties) => {
		if (role === 'input') {
			this.In.properties = properties;
		} else {
			this.Out.properties = properties;
		}
	};

	containsProperty = (role, name) => {
		let properties = this.getProperties(role);
		return properties.findIndex((p) => p === name) !== -1;
	};

	addProperty = (role, name) => {
		let properties = this.getProperties(role);
		properties.push(name);
		this.setProperties(role, properties);
	};

	removeProperty = (role, name) => {
		let properties = this.getProperties(role);
		let updated = properties.filter((p) => p.name !== name);
		this.setProperties(role, updated);
	};
}
