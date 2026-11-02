import React from 'react';
import './PropertyRoleSelector.css';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlDivider from '@shoelace-style/shoelace/dist/react/divider/index.js';
import SlDropdown from '@shoelace-style/shoelace/dist/react/dropdown/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlMenu from '@shoelace-style/shoelace/dist/react/menu/index.js';
import SlMenuItem from '@shoelace-style/shoelace/dist/react/menu-item/index.js';
import { ProfilePropertyRoles, ProfilePropertyRoleID } from '../../../lib/api/types/workspace';
import {
	getCompatibleProfilePropertyRoles,
	getProfilePropertyRole,
	PropertyRoleTarget,
} from '../../helpers/profilePropertyRoles';

interface PropertyRoleSelectorProps {
	propertyRole: ProfilePropertyRoleID | null;
	propertyRoles: ProfilePropertyRoles;
	onChange: (role: ProfilePropertyRoleID | null) => void;
	property: PropertyRoleTarget & { key?: string };
	propertyPaths: Readonly<Record<string, string>>;
}

const PropertyRoleSelector = ({
	propertyRole,
	propertyRoles,
	onChange,
	property,
	propertyPaths,
}: PropertyRoleSelectorProps) => {
	const compatibleRoles = getCompatibleProfilePropertyRoles(property);
	if (compatibleRoles.length === 0 && propertyRole == null) {
		return null;
	}

	const onSelect = (event) => {
		const value = event.detail.item.value as ProfilePropertyRoleID | 'none';
		onChange(value === 'none' ? null : value);
	};

	return (
		<SlDropdown className='property-role-selector' hoist placement='bottom-end' distance={6}>
			<SlButton className='property-role-selector__trigger' slot='trigger' caret>
				{propertyRole == null ? (
					<span className='property-form__placeholder'>Assign a role...</span>
				) : (
					getProfilePropertyRole(propertyRole).label
				)}
			</SlButton>
			<SlMenu className='property-role-selector__menu' onSlSelect={onSelect}>
				<SlMenuItem
					className={`property-role-selector__option property-role-selector__none-option${
						propertyRole == null ? ' property-role-selector__option--selected' : ''
					}`}
					data-property-role-option='none'
					value='none'
				>
					No role
					{propertyRole == null && <SlIcon slot='suffix' name='check-lg' />}
				</SlMenuItem>
				<SlDivider />
				{compatibleRoles.map((role) => {
					const assignedPropertyKey = propertyRoles[role.id];
					const assignedPath =
						assignedPropertyKey !== '' && assignedPropertyKey !== property.key
							? propertyPaths[assignedPropertyKey]
							: null;
					const assignmentStatus = assignedPath == null ? null : `Currently assigned to ${assignedPath}`;
					return (
						<SlMenuItem
							aria-label={[role.label, role.description, assignmentStatus].filter(Boolean).join('. ')}
							className={`property-role-selector__option${
								propertyRole === role.id ? ' property-role-selector__option--selected' : ''
							}`}
							data-property-role-option={role.id}
							key={role.id}
							value={role.id}
						>
							<span className='property-role-selector__option-content'>
								<span>{role.label}</span>
								<span className='property-role-selector__option-description'>{role.description}</span>
								{assignedPath != null && (
									<span className='property-role-selector__option-assignment'>
										Currently assigned to{' '}
										<span className='property-role-selector__option-assignment-path'>
											{assignedPath}
										</span>
									</span>
								)}
							</span>
							{propertyRole === role.id && <SlIcon slot='suffix' name='check-lg' />}
						</SlMenuItem>
					);
				})}
			</SlMenu>
		</SlDropdown>
	);
};

export { PropertyRoleSelector };
