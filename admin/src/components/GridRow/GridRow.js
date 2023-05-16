import './GridRow.css';
import GridCell from '../GridCell/GridCell';
import getChildIndexClassname from '../../utils/getChildIndexClassname';

const GridRow = ({ cells, columns, className }) => {
	let gridCells = [];
	for (let [i, cell] of cells.entries()) {
		let type = columns[i].type;
		let typedCell = { value: cell, type: type };
		let className = getChildIndexClassname(i, cells.length);
		gridCells.push(<GridCell cell={typedCell} className={`GridCell ${className}`} />);
	}

	return <div className={className}>{gridCells}</div>;
};

export default GridRow;
