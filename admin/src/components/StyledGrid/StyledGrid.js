import './StyledGrid.css';
import Grid from '../Grid/Grid';

const StyledGrid = ({ columns, rows, isLoading, actions, noRowsMessage }) => {
	return (
		<div className='StyledGrid'>
			<Grid columns={columns} rows={rows} isLoading={isLoading} actions={actions} noRowsMessage={noRowsMessage} />
		</div>
	);
};

export default StyledGrid;
