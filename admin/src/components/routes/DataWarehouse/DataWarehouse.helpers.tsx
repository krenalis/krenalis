import { WarehouseType } from '../../../types/external/warehouse';

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
		icon: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 9 8"><path d="M0 7h1v1H0z" fill="red"/><path d="M0 0h1v7H0zm2 0h1v8H2zm2 0h1v8H4zm2 0h1v8H6zm2 3.25h1v1.5H8z" fill="#fc0"/></svg>`,
		name: 'clickhouse',
		label: 'ClickHouse',
	},
	{
		icon: `<svg></svg>`,
		name: 'snowflake',
		label: 'Snowflake',
	},
];

export { warehouses, Warehouse };
