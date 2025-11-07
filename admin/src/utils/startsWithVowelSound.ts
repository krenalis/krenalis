const startsWithVowelSound = (str: string) => {
	const firstChar = str.charAt(0).toLowerCase();
	switch (firstChar) {
		case 'a':
		case 'e':
		case 'i':
		case 'o':
		case 'u':
		case 'h':
			return true;
		default:
			return false;
	}
};

export { startsWithVowelSound };
