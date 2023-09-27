import { ReactNode } from 'react';
import { WarehouseSettings, WarehouseType } from '../external/warehouse';

type Variant = 'neutral' | 'primary' | 'success' | 'warning' | 'danger';

type Size = 'small' | 'medium' | 'large';

interface StatusAction {
	name: string;
	onClick: () => void;
}

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

interface Warehouse {
	type: WarehouseType;
	settings: WarehouseSettings;
}

export type { Status, ShoelaceEventTarget, ArrowAnchor, EventListenerEvent, ComboboxItem, Size, Variant, Warehouse };
