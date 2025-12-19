const RAW_TRANSFORMATION_FUNCTIONS = {
	JavaScript: `const transform = ($parameter) => {
    return {$return}
}`,
	Python: `def transform($parameter: dict) -> dict:
	return {$return}
`,
};

const CONFIRM_ANIMATION_DURATION = 1200;

const ERROR_ANIMATION_DURATION = 500;

export { RAW_TRANSFORMATION_FUNCTIONS, CONFIRM_ANIMATION_DURATION, ERROR_ANIMATION_DURATION };
