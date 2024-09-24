import { WarehouseType } from '../../../lib/api/types/warehouse';

interface Warehouse {
	icon: string;
	name: string;
	label: WarehouseType;
}

const warehouses: Warehouse[] = [
	{
		icon: `<svg></svg>`,
		name: 'postgresql',
		label: 'PostgreSQL',
	},
	{
		icon: `<svg></svg>`,
		name: 'snowflake',
		label: 'Snowflake',
	},
];

export { warehouses, Warehouse };
