import { Variant } from '../components/routes/App/App.types';

// ButtonVariant is a button variant as the Admin names it, and as the server
// sends it for connector buttons. An empty variant is the default one.
type ButtonVariant = Variant | 'default' | 'text' | '';

// WebAwesomeButtonVariant is the Web Awesome variant and appearance of a
// button.
interface WebAwesomeButtonVariant {
	variant: 'neutral' | 'brand' | 'success' | 'warning' | 'danger';
	appearance: 'accent' | 'outlined' | 'plain';
}

// buttonVariant returns the Web Awesome variant and appearance of a button with
// the given variant.
const buttonVariant = (variant: ButtonVariant = 'default'): WebAwesomeButtonVariant => {
	switch (variant) {
		case '':
		case 'default':
			return { variant: 'neutral', appearance: 'outlined' };
		case 'text':
			return { variant: 'brand', appearance: 'plain' };
		case 'primary':
			return { variant: 'brand', appearance: 'accent' };
		default:
			return { variant, appearance: 'accent' };
	}
};

export { buttonVariant };
