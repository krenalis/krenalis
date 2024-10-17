import { WarehouseSettings } from '../../../lib/api/types/warehouse';

interface Warehouse {
	name: string;
	settings: WarehouseSettings;
}

type Variant = 'neutral' | 'primary' | 'success' | 'warning' | 'danger';

type Size = 'small' | 'medium' | 'large';

interface Status {
	variant: Variant;
	icon: string;
	text: string;
}

export { Warehouse, Size, Status, Variant };
