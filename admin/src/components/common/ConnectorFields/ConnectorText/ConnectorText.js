import './ConnectorText.css';

const ConnectorText = ({ label, text }) => {
	return (
		<div className='connectorText'>
			<div className='label'>{label}</div>
			<div className='text'>{text}</div>
		</div>
	);
};

export default ConnectorText;
