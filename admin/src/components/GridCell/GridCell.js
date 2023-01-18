import './GridCell.css';

const GridCell = ({ value, className }) => {
	return (
		<div className={className}>
			<div className='cellContent'>{value}</div>
		</div>
	);
};

export default GridCell;
