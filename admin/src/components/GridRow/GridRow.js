import './GridRow.css';
import GridCell from '../GridCell/GridCell';

const GridRow = ({ cells, className }) => {
	let gridCells = [];
	for (let [i, cell] of cells.entries()) {
		let index = i + 1;
		let className = 'GridCell';
		if (index === 1) {
			className += ' firstCell';
		}
		if (index === cells.length) {
			className += ' lastCell';
		}
		gridCells.push(<GridCell value={cell} className={className} />);
	}

	return <div className={className}>{gridCells}</div>;
};

export default GridRow;
