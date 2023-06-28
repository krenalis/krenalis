import './GridHeaderRow.css';
import GridHeaderCell from '../GridHeaderCell/GridHeaderCell';
import getChildIndexClassname from '../../../../utils/getChildIndexClassname';

const GridHeaderRow = ({ columns }) => {
	const gridHeaderCells = [];
	for (const [i, column] of columns.entries()) {
		const className = getChildIndexClassname(i, columns.length);
		gridHeaderCells.push(
			<GridHeaderCell
				className={`gridHeaderCell ${className}`}
				value={column.name}
				alignment={column.alignment}
			/>
		);
	}

	return <div className='gridHeaderRow'>{gridHeaderCells}</div>;
};

export default GridHeaderRow;
