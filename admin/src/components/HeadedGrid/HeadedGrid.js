import './HeadedGrid.css';
import StyledGrid from '../StyledGrid/StyledGrid';

const HeadedGrid = ({ columns, rows, title, children, isLoading }) => {
	return (
		<div className='HeadedGrid'>
			<div className='gridHead'>
				<div className='gridTitle'>{title}</div>
				<div className='headComponents'>{children}</div>
			</div>
			<StyledGrid columns={columns} rows={rows} isLoading={isLoading} />
		</div>
	);
};

export default HeadedGrid;
