import './GridCell.css';
import toJSDateString from '../../utils/toJSDateString';

const GridCell = ({ cell, className }) => {
	let value, date;
	switch (cell.type) {
		case 'Object':
			value = JSON.stringify(cell.value);
			break;
		case 'DateTime':
			date = new Date(toJSDateString(cell.value));
			value = date.toLocaleString('it-IT', { timeZone: 'Europe/Rome' });
			break;
		case 'Date':
			date = new Date(toJSDateString(cell.value));
			value = date.toLocaleDateString('it-IT', { timeZone: 'Europe/Rome' });
			break;
		case 'Time':
			date = new Date(toJSDateString(cell.value));
			value = date.toLocaleTimeString('it-IT', { timeZone: 'Europe/Rome' });
			break;
		default:
			value = cell.value;
			break;
	}

	return (
		<div className={className}>
			<div className='cellContent'>{value}</div>
		</div>
	);
};

export default GridCell;
