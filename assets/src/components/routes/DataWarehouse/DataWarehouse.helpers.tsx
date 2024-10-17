interface Warehouse {
	icon: string;
	name: string;
}

const warehouses: Warehouse[] = [
	{
		icon: `<svg></svg>`,
		name: 'PostgreSQL',
	},
	{
		icon: `<svg></svg>`,
		name: 'Snowflake',
	},
];

export { warehouses, Warehouse };
