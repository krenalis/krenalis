import { WarehouseSettings, WarehouseType } from '../../../lib/api/types/warehouse';

interface Warehouse {
	type: WarehouseType;
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
