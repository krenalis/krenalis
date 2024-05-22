import { Member, MemberAvatar, MemberToSet } from '../api/types/responses';

interface TransformedMember {
	ID: number;
	Name: string;
	Email: string;
	Avatar: MemberAvatar;
	Initials: string;
	Invitation: '' | 'Invited' | 'Expired';
	CreatedAt: string;
}

const transformMember = (member: Member): TransformedMember => {
	const split = member.Name.split(' ');
	let initials = '';
	for (let i = 0; i < 2; i++) {
		const word = split[i];
		if (word) {
			initials += word[0];
		}
	}
	const transformed = { ...member, Initials: initials };
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

const validateMemberToSet = (member: MemberToSet, isPasswordRequired: boolean): string => {
	if (member.Name === '') {
		return 'name must not be empty';
	}
	if (member.Name.length > 45) {
		return 'name must be shorter than 45 characters';
	}
	const error = validateMemberEmail(member.Email);
	if (error !== '') {
		return error;
	}
	if (isPasswordRequired && member.Password === '') {
		return 'password must not be empty';
	}
	if (member.Password != null) {
		if (member.Password.length < 8) {
			return 'password must be at least 8 characters long';
		}
		if (member.Password.length > 72) {
			return 'password must be shorter than 72 characters';
		}
	}
	return '';
};

export { transformMember, TransformedMember, validateMemberToSet, validateMemberEmail };
