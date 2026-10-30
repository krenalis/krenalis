const DEFAULT_PREVIEW_DELAY_MS = 1000;
const MEDIUM_PREVIEW_DELAY_MS = 1500;
const SLOW_PREVIEW_DELAY_MS = 2500;
const PREVIEW_LATENCY_SAMPLE_LIMIT = 5;
const MINIMUM_PREVIEW_LATENCY_SAMPLES = 3;
const VERY_SLOW_PREVIEW_MS = 8000;
const FAST_PREVIEW_MS = 4000;

type ProfilesPreviewDelay =
	| typeof DEFAULT_PREVIEW_DELAY_MS
	| typeof MEDIUM_PREVIEW_DELAY_MS
	| typeof SLOW_PREVIEW_DELAY_MS;

interface ProfilesPreviewTiming {
	delayMs: ProfilesPreviewDelay;
	delayCandidate?: ProfilesPreviewDelay;
	delayCandidateConfirmations: number;
	fastRecoveryCount: number;
	latencySamples: number[];
	consecutiveSlowCount: number;
	suspended: boolean;
}

const createProfilesPreviewTiming = (): ProfilesPreviewTiming => ({
	delayMs: DEFAULT_PREVIEW_DELAY_MS,
	delayCandidateConfirmations: 0,
	fastRecoveryCount: 0,
	latencySamples: [],
	consecutiveSlowCount: 0,
	suspended: false,
});

const previewDelayForLatencies = (latencies: number[]): ProfilesPreviewDelay => {
	const sorted = [...latencies].sort((a, b) => a - b);
	const middle = Math.floor(sorted.length / 2);
	const median = sorted.length % 2 === 0 ? (sorted[middle - 1] + sorted[middle]) / 2 : sorted[middle];
	if (median < 1500) {
		return DEFAULT_PREVIEW_DELAY_MS;
	}
	if (median < FAST_PREVIEW_MS) {
		return MEDIUM_PREVIEW_DELAY_MS;
	}
	return SLOW_PREVIEW_DELAY_MS;
};

const recordProfilesPreviewLatency = (timing: ProfilesPreviewTiming, latencyMs: number): ProfilesPreviewTiming => {
	const latencySamples = [...timing.latencySamples, latencyMs].slice(-PREVIEW_LATENCY_SAMPLE_LIMIT);
	const consecutiveSlowCount = latencyMs > VERY_SLOW_PREVIEW_MS ? timing.consecutiveSlowCount + 1 : 0;

	if (timing.suspended) {
		const fastRecoveryCount = latencyMs < FAST_PREVIEW_MS ? timing.fastRecoveryCount + 1 : 0;
		if (fastRecoveryCount < 2) {
			return { ...timing, fastRecoveryCount, latencySamples, consecutiveSlowCount };
		}
		return {
			...timing,
			delayMs: SLOW_PREVIEW_DELAY_MS,
			delayCandidate: undefined,
			delayCandidateConfirmations: 0,
			fastRecoveryCount: 0,
			latencySamples,
			consecutiveSlowCount: 0,
			suspended: false,
		};
	}

	if (consecutiveSlowCount >= 3) {
		return { ...timing, fastRecoveryCount: 0, latencySamples, consecutiveSlowCount, suspended: true };
	}

	if (latencySamples.length < MINIMUM_PREVIEW_LATENCY_SAMPLES) {
		return { ...timing, latencySamples, consecutiveSlowCount };
	}

	const delayCandidate = previewDelayForLatencies(latencySamples);
	if (delayCandidate === timing.delayMs) {
		return {
			...timing,
			delayCandidate: undefined,
			delayCandidateConfirmations: 0,
			latencySamples,
			consecutiveSlowCount,
		};
	}

	const delayCandidateConfirmations =
		delayCandidate === timing.delayCandidate ? timing.delayCandidateConfirmations + 1 : 1;
	if (delayCandidateConfirmations < 2) {
		return { ...timing, delayCandidate, delayCandidateConfirmations, latencySamples, consecutiveSlowCount };
	}

	return {
		...timing,
		delayMs: delayCandidate,
		delayCandidate: undefined,
		delayCandidateConfirmations: 0,
		latencySamples,
		consecutiveSlowCount,
	};
};

export { createProfilesPreviewTiming, recordProfilesPreviewLatency };
