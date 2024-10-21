function normalize(obj) {
	if (obj === undefined) {
		throw new Error("transformed value is undefined");
	}
	if (obj === null) {
		throw new Error("transformed value is null");
	}
	if (Array.isArray(obj)) {
		throw new Error("transformed value is array");
	}
	if (typeof obj !== "object") {
		throw new Error("transformed value is "+(typeof obj)+", not object");
	}
	function norm(obj, set) {
		if (set.has(obj)) {
			throw new Error("transformed value contains a circular reference");
		}
		set.add(obj);
		if (Array.isArray(obj)) {
			const len = obj.length;
			for (let i = 0; i < len; i++) {
				const v = obj[i];
				if (typeof v === "number") {
					if (!Number.isFinite(v)) {
						obj[i] = String(v);
					}
				} else if (v === undefined) {
					delete(obj[i]);
				} else if (typeof v === "object" && v !== null) {
					norm(v, set);
				} else if (typeof v === "bigint") {
					obj[i] = v.toString();
				}
			}
		} else {
			for (const k in obj) {
				if (obj.hasOwnProperty(k)) {
					const v = obj[k];
					if (typeof v === "number") {
						if (!Number.isFinite(v)) {
							obj[k] = String(v);
						}
					} else if (v === undefined) {
						delete(obj[k]);
					} else if (typeof v === "object" && v !== null) {
						norm(v, set);
					} else if (typeof v === "bigint") {
						obj[k] = v.toString();
					}
				}
			}
		}
		set.delete(obj);
	}
	norm(obj, new WeakSet());
}