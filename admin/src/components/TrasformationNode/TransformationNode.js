import './TransformationNode.css';
import { SlIconButton, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const TransformationNode = ({ transformation: t, onSelect, onConnect, onRemove }) => {
	let node;
	if (t.Type === 'one-to-one') {
		node = (
			<div className='TransformationNode oneToOne'>
				<SlIconButton name='x-lg' onClick={onRemove} />
			</div>
		);
	} else if (t.Type === 'predefined') {
		node = (
			<div className='TransformationNode predefined'>
				<div className='leftHandles'>
					{t.PredefinedFunc.In.properties.map((parameter) => {
						let trimmed = parameter.label.replace(/\s/g, '');
						return (
							<div className='handleWrapper'>
								<div className='label'>{parameter.label}</div>
								<div
									className='handle'
									onClick={onConnect != null ? () => onConnect(parameter.label) : null}
									id={`transformation-${t.Position}-input-${trimmed}`}
								></div>
							</div>
						);
					})}
				</div>
				<div className='info'>
					<SlIcon name={t.PredefinedFunc.Icon}></SlIcon>
					<div className='name'>{t.PredefinedFunc.Name}</div>
					<SlIconButton name='x-lg' onClick={onRemove} />
				</div>
				<div className='rightHandles'>
					{t.PredefinedFunc.Out.properties.map((parameter) => {
						let trimmed = parameter.label.replace(/\s/g, '');
						return (
							<div className='handleWrapper'>
								<div className='label'>{parameter.label}</div>
								<div
									className='handle'
									onClick={onConnect != null ? () => onConnect(parameter.label) : null}
									id={`transformation-${t.Position}-output-${trimmed}`}
								></div>
							</div>
						);
					})}
				</div>
			</div>
		);
	} else if (t.Type === 'custom') {
		node = (
			<div className='TransformationNode custom'>
				<div className='handle left' onClick={onConnect}></div>
				<SlIconButton name='braces' onClick={onSelect} />
				<div className='handle right' onClick={onConnect}></div>
			</div>
		);
	}
	return node;
};

export default TransformationNode;
