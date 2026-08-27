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

interface IdentityResolutionMetricDay {
	day: string;
	identities: number;
	profiles: number;
	profilesAnonymous: number;
	profilesRecognized: number;
	identitiesPerProfile: number;
	linkedIdentitiesRate: number;
}

interface IdentityResolutionComposition {
	one: number;
	two: number;
	three: number;
	fourToTen: number;
	elevenToTwenty: number;
	moreThanTwenty: number;
}

interface IdentityResolutionMetric {
	observedAt: string;
	identities: {
		total: number;
		anonymous: number;
		recognized: number;
	};
	profiles: {
		total: number;
		anonymous: number;
		recognized: number;
	};
	composition: IdentityResolutionComposition;
	identitiesPerProfile: number;
	linkedIdentitiesRate: number;
}

export type {
	IdentityConnectionMetric,
	IdentityMetric,
	IdentityMetricDay,
	IdentityResolutionComposition,
	IdentityResolutionMetric,
	IdentityResolutionMetricDay,
};
