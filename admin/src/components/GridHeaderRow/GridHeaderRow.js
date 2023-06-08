import './GridHeaderRow.css';
import GridHeaderCell from '../GridHeaderCell/GridHeaderCell';
import getChildIndexClassname from '../../utils/getChildIndexClassname';

const GridHeaderRow = ({ columns }) => {
	let gridHeaderCells = [];
	for (let [i, column] of columns.entries()) {
		let className = getChildIndexClassname(i, columns.length);
		gridHeaderCells.push(
			<GridHeaderCell
				className={`GridHeaderCell ${className}`}
				value={column.name}
				alignment={column.alignment}
			/>
		);
	}

	return <div className='GridHeaderRow'>{gridHeaderCells}</div>;
};

export default GridHeaderRow;
