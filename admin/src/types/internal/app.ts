import { ReactNode } from 'react';

type Variant = 'neutral' | 'primary' | 'success' | 'warning' | 'danger' | 'default';

type Size = 'small' | 'medium' | 'large';

interface Status {
	variant: Variant;
	icon: string;
	text: string;
}

interface ShoelaceEventTarget extends EventTarget {
	value: string;
}

type ArrowAnchor = 'middle' | 'left' | 'right' | 'top' | 'bottom' | 'auto';

interface EventListenerEvent {
	id: number;
	err: string;
	type: string;
	path: string;
	time: string;
	full: string;
}

interface ComboboxItem {
	content: ReactNode; // The content shown for the item.
	term: string; // The search term used to find and show the item when filtering after user input.
}

export { Status, ShoelaceEventTarget, ArrowAnchor, EventListenerEvent, ComboboxItem, Size, Variant };
