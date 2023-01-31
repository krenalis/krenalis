export class Mapping {
	constructor(obj) {
		this.InProperties = [];
		this.OutProperties = [];
		this.PredefinedFunc = null;
		this.CustomFunc = null;
		this.Position = 0;
		this.Type = '';
		if (obj) {
			Object.assign(this, obj);
			if (obj.PredefinedFunc !== null) {
				this.Type = 'predefined';
			} else if (obj.InProperties.length === 1 && obj.OutProperties.length === 1 && obj.CustomFunc == null) {
				this.Type = 'one-to-one';
			}
		}
	}

	static createOneToOneMapping(input, output, position) {
		return new Mapping({
			InProperties: [input],
			OutProperties: [output],
			PredefinedFunc: null,
			CustomFunc: null,
			Position: position,
		});
	}

	static createPredefinedMapping(predefined, position) {
		return new Mapping({
			InProperties: [],
			OutProperties: [],
			PredefinedFunc: predefined,
			CustomFunc: null,
			Position: position,
		});
	}

	toServerFormat = () => {
		return {
			InProperties: this.InProperties,
			OutProperties: this.OutProperties,
			PredefinedFunc: this.PredefinedFunc != null ? this.PredefinedFunc.ID : null,
			CustomFunc: this.CustomFunc,
		};
	};

	getProperties = (role) => {
		if (role === 'input') {
			return this.InProperties;
		} else {
			return this.OutProperties;
		}
	};

	setProperties = (role, properties) => {
		if (role === 'input') {
			this.InProperties = properties;
		} else {
			this.OutProperties = properties;
		}
	};

	containsProperty = (role, name) => {
		let properties = this.getProperties(role);
		return properties.findIndex((p) => p === name) !== -1;
	};

	addProperty = (role, name, parameter) => {
		switch (this.Type) {
			case 'one-to-one':
				return;
			case 'predefined':
				let parametersLength = this.getParametersLength(role);
				let parameterIndex = this.getParameterIndex(role, parameter);
				if (role === 'output' && parametersLength === 1) {
					// in this case it's possible to connect an arbitrary number
					// of output properties.
					this.OutProperties.push(name);
				} else {
					let properties = this.getProperties(role);
					if (properties.length === 0) {
						properties = Array(parametersLength);
						properties[parameterIndex] = name;
					} else {
						properties[parameterIndex] = name;
					}
					this.setProperties(role, properties);
				}
				break;
			default:
				return;
		}
	};

	removeProperty = (role, name) => {
		if (this.Type === 'predefined') {
			let parametersLength = this.getParametersLength(role);
			if (role !== 'output' || parametersLength !== 1) {
				// maintain the order.
				let properties = this.getProperties(role);
				let updated = [];
				for (let p of properties) {
					if (p === name) {
						updated.push(undefined);
					} else {
						updated.push(p);
					}
				}
				this.setProperties(role, updated);
				return;
			}
		}
		let properties = this.getProperties(role);
		let updated = properties.filter((p) => p !== name);
		this.setProperties(role, updated);
	};

	validateProperties = () => {
		if (this.Type !== 'predefined') return null;
		for (let [i, p] of this.PredefinedFunc.In.properties.entries()) {
			if (this.InProperties[i] === undefined) {
				return `The input parameter "${p.label}" of the predefined mapping "${this.PredefinedFunc.Name}" is not linked to any input property`;
			}
		}
		for (let [i, p] of this.PredefinedFunc.Out.properties.entries()) {
			if (this.OutProperties[i] === undefined) {
				return `The output parameter "${p.label}" of the predefined mapping "${this.PredefinedFunc.Name}" is not linked to any output property`;
			}
		}
		return null;
	};

	getParametersLength = (role) => {
		if (this.Type !== 'predefined') return 0;
		if (role === 'input') {
			return this.PredefinedFunc.In.properties.length;
		} else {
			return this.PredefinedFunc.Out.properties.length;
		}
	};

	getParameterIndex = (role, parameter) => {
		if (this.Type !== 'predefined') return 0;
		if (role === 'input') {
			return this.PredefinedFunc.In.properties.findIndex((p) => p.label === parameter);
		} else {
			return this.PredefinedFunc.Out.properties.findIndex((p) => p.label === parameter);
		}
	};
}
