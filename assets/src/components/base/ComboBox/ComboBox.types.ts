import { ReactNode } from 'react';

interface ComboboxItem {
	content: ReactNode; // The content shown for the item.
	term: string; // The search term used to find and show the item when filtering after user input.
}

export { ComboboxItem };
