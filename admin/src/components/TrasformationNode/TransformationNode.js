import './TransformationNode.css';
import { getTransformationType } from '../../utils/getTransformationType';
import { SlIconButton, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const TransformationNode = ({
	transformation: t,
	onSelect,
	onCustomTransformationConnect,
	onPredefinedTransformationConnect,
	onRemove,
}) => {
	let transformationType = getTransformationType(t);
	let node;
	if (transformationType === 'one-to-one') {
		node = (
			<div className='TransformationNode oneToOne'>
				<SlIconButton name='x-lg' onClick={onRemove} />
			</div>
		);
	} else if (transformationType === 'predefined') {
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
									onClick={
										onPredefinedTransformationConnect != null
											? () => onPredefinedTransformationConnect(parameter.label)
											: null
									}
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
									onClick={
										onPredefinedTransformationConnect != null
											? () => onPredefinedTransformationConnect(parameter.label)
											: null
									}
									id={`transformation-${t.Position}-output-${trimmed}`}
								></div>
							</div>
						);
					})}
				</div>
			</div>
		);
	} else if (transformationType === 'custom') {
		node = (
			<div className='TransformationNode custom'>
				<div className='handle left' onClick={onCustomTransformationConnect}></div>
				<SlIconButton name='braces' onClick={onSelect} />
				<div className='handle right' onClick={onCustomTransformationConnect}></div>
			</div>
		);
	}
	return node;
};

export default TransformationNode;
