import './ConnectorText.css';

const ConnectorText = ({ label, text }) => {
	return (
		<div className='ConnectorText'>
			<div className='label'>{label}</div>
			<div className='text'>{text}</div>
		</div>
	);
};

export default ConnectorText;
