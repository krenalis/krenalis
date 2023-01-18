import './StyledGrid.css';
import Grid from '../Grid/Grid';

const StyledGrid = ({ columns, rows, isLoading }) => {
	return (
		<div className='StyledGrid'>
			<Grid columns={columns} rows={rows} isLoading={isLoading} />
		</div>
	);
};

export default StyledGrid;
