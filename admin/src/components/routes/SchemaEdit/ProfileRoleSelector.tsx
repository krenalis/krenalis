import React from 'react';
import './ProfileRoleSelector.css';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlDivider from '@shoelace-style/shoelace/dist/react/divider/index.js';
import SlDropdown from '@shoelace-style/shoelace/dist/react/dropdown/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlMenu from '@shoelace-style/shoelace/dist/react/menu/index.js';
import SlMenuItem from '@shoelace-style/shoelace/dist/react/menu-item/index.js';
import { ProfileRoleAssignments, ProfileRoleID } from '../../../lib/api/types/workspace';
import { getCompatibleProfileRoles, getProfileRole, ProfileRoleProperty } from '../../helpers/profileRoles';

interface ProfileRoleSelectorProps {
	assignedRole: ProfileRoleID | null;
	assignedRoles: ProfileRoleAssignments;
	onChange: (role: ProfileRoleID | null) => void;
	property: ProfileRoleProperty & { key?: string };
	propertyPaths: Readonly<Record<string, string>>;
}

const ProfileRoleSelector = ({
	assignedRole,
	assignedRoles,
	onChange,
	property,
	propertyPaths,
}: ProfileRoleSelectorProps) => {
	const compatibleRoles = getCompatibleProfileRoles(property);
	if (compatibleRoles.length === 0 && assignedRole == null) {
		return null;
	}

	const onSelect = (event) => {
		const value = event.detail.item.value as ProfileRoleID | 'none';
		onChange(value === 'none' ? null : value);
	};

	return (
		<SlDropdown className='profile-role-selector' hoist placement='bottom-end' distance={6}>
			<SlButton className='profile-role-selector__trigger' slot='trigger' caret>
				{assignedRole == null ? 'Not assigned' : getProfileRole(assignedRole).label}
			</SlButton>
			<SlMenu className='profile-role-selector__menu' onSlSelect={onSelect}>
				<SlMenuItem
					className={`profile-role-selector__option profile-role-selector__none-option${
						assignedRole == null ? ' profile-role-selector__option--selected' : ''
					}`}
					data-profile-role-option='none'
					value='none'
				>
					Not assigned
					{assignedRole == null && <SlIcon slot='suffix' name='check-lg' />}
				</SlMenuItem>
				<SlDivider />
				{compatibleRoles.map((role) => {
					const assignedPropertyKey = assignedRoles[role.id];
					const assignedPath =
						assignedPropertyKey !== '' && assignedPropertyKey !== property.key
							? propertyPaths[assignedPropertyKey]
							: null;
					const assignmentStatus = assignedPath == null ? null : `Currently assigned to ${assignedPath}`;
					return (
						<SlMenuItem
							aria-label={[role.label, role.description, assignmentStatus].filter(Boolean).join('. ')}
							className={`profile-role-selector__option${
								assignedRole === role.id ? ' profile-role-selector__option--selected' : ''
							}`}
							data-profile-role-option={role.id}
							key={role.id}
							value={role.id}
						>
							<span className='profile-role-selector__option-content'>
								<span>{role.label}</span>
								<span className='profile-role-selector__option-description'>{role.description}</span>
								{assignedPath != null && (
									<span className='profile-role-selector__option-assignment'>
										Currently assigned to{' '}
										<span className='profile-role-selector__option-assignment-path'>
											{assignedPath}
										</span>
									</span>
								)}
							</span>
							{assignedRole === role.id && <SlIcon slot='suffix' name='check-lg' />}
						</SlMenuItem>
					);
				})}
			</SlMenu>
		</SlDropdown>
	);
};

export { ProfileRoleSelector };
