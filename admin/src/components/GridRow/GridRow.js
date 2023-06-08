import './GridRow.css';
import GridCell from '../GridCell/GridCell';
import getChildIndexClassname from '../../utils/getChildIndexClassname';

const GridRow = ({ row, columns, className }) => {
	let gridCells = [];
	for (let [i, cell] of row.cells.entries()) {
		let type = columns[i].type;
		let alignment = columns[i].alignment;
		let typedCell = { value: cell, type: type, alignment: alignment };
		let className = getChildIndexClassname(i, row.cells.length);
		gridCells.push(<GridCell cell={typedCell} className={`GridCell ${className}`} />);
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
