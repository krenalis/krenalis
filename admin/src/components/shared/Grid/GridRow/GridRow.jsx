import './GridRow.css';
import GridCell from '../GridCell/GridCell';
import getChildIndexClassname from '../../../../lib/utils/getChildIndexClassname';

const GridRow = ({ row, columns, className }) => {
	const gridCells = [];
	for (const [i, cell] of row.cells.entries()) {
		const type = columns[i].type;
		const alignment = columns[i].alignment;
		const typedCell = { value: cell, type: type, alignment: alignment };
		const className = getChildIndexClassname(i, row.cells.length);
		gridCells.push(<GridCell cell={typedCell} className={`gridCell ${className}`} />);
	}

	return (
		<div
			key={row.key}
			className={`${className}${row.onClick ? ' clickable' : ''}`}
			onClick={row.onClick}
			data-animation={row.animation}
		>
			{gridCells}
		</div>
	);
};

export default GridRow;
