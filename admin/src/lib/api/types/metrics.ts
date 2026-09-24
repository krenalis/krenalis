interface IdentityConnectionMetric {
	connection: string;
	anonymous: number;
	recognized: number;
	withoutProfile: number;
}

interface IdentityMetricDay {
	day: string;
	total: number;
	anonymous: number;
	recognized: number;
}

interface IdentityMetric {
	observedAt: string;
	total: number;
	anonymous: number;
	recognized: number;
	withoutProfile: number;
	connections: IdentityConnectionMetric[];
}

export type { IdentityConnectionMetric, IdentityMetric, IdentityMetricDay };
