const rawTransformationFunctions = {
	JavaScript: `const transform = ($parameterName) => {
    return {}
}`,
	Python: `def transform($parameterName: dict) -> dict:
	return {}
`,
};

const CONFIRM_ANIMATION_DURATION = 1200;

export { rawTransformationFunctions, CONFIRM_ANIMATION_DURATION };
