import { ReactNode } from 'react';

interface ComboboxItem {
	content: ReactNode; // The content shown for the item.
	displayValue?: string; // The value shown in the input after selecting the item.
	term: string; // The canonical value returned when the item is selected.
}

export { ComboboxItem };
