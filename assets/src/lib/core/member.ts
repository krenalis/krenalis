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

const validateMemberEmail = (email: string) => {
	if (email === '') {
		throw new Error('email must not be empty');
	}
	if (email.length > 120) {
		throw new Error('email must be shorter than 120 characters');
	}
};

const validateMemberPassword = (password: string) => {
	if (password.length < 8) {
		throw new Error('password must be at least 8 characters long');
	}
	if (password.length > 72) {
		throw new Error('password must be shorter than 72 characters');
	}
};

const validateMemberToSet = (
	member: MemberToSet,
	validateEmail: boolean,
	validatePassword: boolean,
	password2?: string,
) => {
	if (member.name === '') {
		throw new Error('name must not be empty');
	}
	if (member.name.length > 45) {
		throw new Error('name must be shorter than 45 characters');
	}
	if (validateEmail) {
		try {
			validateMemberEmail(member.email);
		} catch (err) {
			throw err;
		}
	}
	if (validatePassword) {
		if (member.password === '') {
			throw new Error('password must not be empty');
		}
		try {
			validateMemberPassword(member.password);
		} catch (err) {
			throw err;
		}
		if (password2 != null && member.password !== password2) {
			throw new Error('Passwords must match');
		}
	}
};

export { transformMember, TransformedMember, validateMemberToSet, validateMemberEmail, validateMemberPassword };
