import './MappingNode.css';
import { SlIconButton, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const MappingNode = ({ mapping: m, onConnect, onRemove }) => {
	let node;
	if (m.Type === 'one-to-one') {
		node = (
			<div className='MappingNode oneToOne'>
				<SlIconButton name='x-lg' onClick={onRemove} />
			</div>
		);
	} else if (m.Type === 'predefined') {
		node = (
			<div className='MappingNode predefined'>
				<div className='leftHandles'>
					{m.PredefinedFunc.In.properties.map((parameter) => {
						let trimmed = parameter.label.replace(/\s/g, '');
						return (
							<div className='handleWrapper'>
								<div className='label'>{parameter.label}</div>
								<div
									className='handle'
									onClick={onConnect != null ? () => onConnect(parameter.label) : null}
									id={`mapping-${m.Position}-input-${trimmed}`}
								></div>
							</div>
						);
					})}
				</div>
				<div className='info'>
					<SlIcon name={m.PredefinedFunc.Icon}></SlIcon>
					<div className='name'>{m.PredefinedFunc.Name}</div>
					<SlIconButton name='x-lg' onClick={onRemove} />
				</div>
				<div className='rightHandles'>
					{m.PredefinedFunc.Out.properties.map((parameter) => {
						let trimmed = parameter.label.replace(/\s/g, '');
						return (
							<div className='handleWrapper'>
								<div className='label'>{parameter.label}</div>
								<div
									className='handle'
									onClick={onConnect != null ? () => onConnect(parameter.label) : null}
									id={`mapping-${m.Position}-output-${trimmed}`}
								></div>
							</div>
						);
					})}
				</div>
			</div>
		);
	}
	return node;
};

export default MappingNode;
