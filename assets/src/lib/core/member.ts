import { Member, MemberAvatar, MemberToSet } from '../api/types/responses';

interface TransformedMember {
	id: number;
	name: string;
	email: string;
	avatar: MemberAvatar;
	initials: string;
	invitation: '' | 'Invited' | 'Expired';
	createdAt: string;
}

const transformMember = (member: Member): TransformedMember => {
	const split = member.name.split(' ');
	let initials = '';
	for (let i = 0; i < 2; i++) {
		const word = split[i];
		if (word) {
			initials += word[0];
		}
	}
	const transformed = { ...member, initials: initials };
	return transformed;
};

const validateMemberEmail = (email: string): string => {
	if (email === '') {
		return 'email must not be empty';
	}
	if (email.length > 120) {
		return 'email must be shorter than 120 characters';
	}
	return '';
};

const validateMemberPassword = (password: string): string => {
	if (password.length < 8) {
		return 'password must be at least 8 characters long';
	}
	if (password.length > 72) {
		return 'password must be shorter than 72 characters';
	}
	return '';
};

const validateMemberToSet = (member: MemberToSet, isPasswordRequired: boolean): string => {
	if (member.name === '') {
		return 'name must not be empty';
	}
	if (member.name.length > 45) {
		return 'name must be shorter than 45 characters';
	}
	const error = validateMemberEmail(member.email);
	if (error !== '') {
		return error;
	}
	if (isPasswordRequired && member.password === '') {
		return 'password must not be empty';
	}
	if (member.password != null) {
		return validateMemberPassword(member.password);
	}
	return '';
};

export { transformMember, TransformedMember, validateMemberToSet, validateMemberEmail, validateMemberPassword };
