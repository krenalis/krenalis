import './GridHeaderRow.css';
import GridHeaderCell from '../GridHeaderCell/GridHeaderCell';

const GridHeaderRow = ({ columns }) => {
	let gridHeaderCells = [];
	for (let [i, column] of columns.entries()) {
		let index = i + 1;
		let className = 'GridHeaderCell';
		if (index === 1) {
			className += ' firstCell';
		}
		if (index === columns.length) {
			className += ' lastCell';
		}
		gridHeaderCells.push(<GridHeaderCell value={column.Name} className={className} />);
	}

	return <div className='GridHeaderRow'>{gridHeaderCells}</div>;
};

export default GridHeaderRow;
