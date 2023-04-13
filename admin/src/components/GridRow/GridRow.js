import './GridRow.css';
import GridCell from '../GridCell/GridCell';
import getChildIndexClassname from '../../utils/getChildIndexClassname';

const GridRow = ({ cells, className }) => {
	let gridCells = [];
	for (let [i, cell] of cells.entries()) {
		let className = getChildIndexClassname(i, cells.length);
		gridCells.push(<GridCell value={cell} className={`GridCell ${className}`} />);
	}

	return <div className={className}>{gridCells}</div>;
};

export default GridRow;
