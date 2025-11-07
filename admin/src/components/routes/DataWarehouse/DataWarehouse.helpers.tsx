interface Warehouse {
	code: string;
	name: string;
}

const warehouses: Warehouse[] = [
	{
		code: 'postgresql',
		name: 'PostgreSQL',
	},
	{
		code: 'snowflake',
		name: 'Snowflake',
	},
];

export { warehouses, Warehouse };
