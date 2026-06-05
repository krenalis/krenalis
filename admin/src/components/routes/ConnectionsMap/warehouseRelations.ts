import TransformedConnection from '../../../lib/core/connection';

type WarehouseRelation = 'dwh-user' | 'dwh-event';

const getWarehouseRelations = (connection: TransformedConnection): WarehouseRelation[] => {
	const relations: WarehouseRelation[] = [];

	if (connection.pipelines.some((p) => p.target === 'User' && p.enabled)) {
		relations.push('dwh-user');
	}
	if (connection.isSource && connection.pipelines.some((p) => p.target === 'Event' && p.enabled)) {
		relations.push('dwh-event');
	}

	return relations;
};

export { getWarehouseRelations };
