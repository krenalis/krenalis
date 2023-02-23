import './StyledGrid.css';
import Grid from '../Grid/Grid';

const StyledGrid = ({ columns, rows, isLoading, actions }) => {
	return (
		<div className='StyledGrid'>
			<Grid columns={columns} rows={rows} isLoading={isLoading} actions={actions} />
		</div>
	);
};

export default StyledGrid;
