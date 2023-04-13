import './StyledGrid.css';
import Grid from '../Grid/Grid';

const StyledGrid = ({ columns, rows, isLoading, noRowsMessage }) => {
	return (
		<div className='StyledGrid'>
			<Grid columns={columns} rows={rows} isLoading={isLoading} noRowsMessage={noRowsMessage} />
		</div>
	);
};

export default StyledGrid;
